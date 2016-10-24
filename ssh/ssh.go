package ssh

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type SSH struct{
	Terminal Terminal
}

//go:generate mockgen -package mocks -destination mocks/terminal.go github.com/pivotal-cf/pcfdev-cli/ssh Terminal
type Terminal interface {
	MakeRaw(fd int) (*terminal.State, error)
	Restore(fd int, state *terminal.State) error
}

func (*SSH) GenerateAddress() (host string, port string, err error) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer conn.Close()
	address := strings.Split(conn.Addr().String(), ":")
	return address[0], address[1], nil
}

func (s *SSH) GetSSHOutput(command string, addresses []SSHAddress, privateKey []byte, timeout time.Duration) (string, error) {
	client, session, err := s.newSession(addresses, privateKey, timeout)
	if err != nil {
		return "", err
	}
	defer client.Close()
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}

func (s *SSH) RunSSHCommand(command string, addresses []SSHAddress, privateKey []byte, timeout time.Duration, stdout io.Writer, stderr io.Writer) (err error) {
	client, session, err := s.newSession(addresses, privateKey, timeout)
	if err != nil {
		return err
	}
	defer client.Close()
	defer session.Close()

	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	go io.Copy(stdout, sessionStdout)

	sessionStderr, err := session.StderrPipe()
	if err != nil {
		return err
	}
	go io.Copy(stderr, sessionStderr)

	return session.Run(command)
}

func (s *SSH) StartSSHSession(addresses []SSHAddress, privateKey []byte, timeout time.Duration, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	client, session, err := s.newSession(addresses, privateKey, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	oldState, err := s.Terminal.MakeRaw(0)
	if err != nil {
		return err
	}
	defer s.Terminal.Restore(0, oldState)

	if err := session.RequestPty("xterm", 50, 50, modes); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	return session.Wait()
}

func (s *SSH) WaitForSSH(addresses []SSHAddress, privateKey []byte, timeout time.Duration) error {
	_, err := s.waitForSSH(addresses, privateKey, timeout)
	return err
}

func (s *SSH) GenerateKeypair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	encodedPrivateKey := new(bytes.Buffer)
	marshaledPrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	if err = pem.Encode(encodedPrivateKey, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: marshaledPrivateKey}); err != nil {
		return nil, nil, err
	}

	publicKey, err := ssh.NewPublicKey(privateKey.Public())
	if err != nil {
		return nil, nil, err
	}

	return encodedPrivateKey.Bytes(), ssh.MarshalAuthorizedKey(publicKey), nil
}

func (s *SSH) newSession(addresses []SSHAddress, privateKey []byte, timeout time.Duration) (*ssh.Client, *ssh.Session, error) {
	client, err := s.waitForSSH(addresses, privateKey, timeout)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	return client, session, nil
}

func (*SSH) waitForSSH(addresses []SSHAddress, privateKey []byte, timeout time.Duration) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("could not parse private key: %s", err)
	}

	config := &ssh.ClientConfig{
		User: "vcap",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		Timeout: 10 * time.Second,
	}

	clientChan := make(chan *ssh.Client, len(addresses))
	errorChan := make(chan error, len(addresses))
	doneChan := make(chan bool)

	for _, address := range addresses {
		go func(ip string, port string) {
			var client *ssh.Client
			var dialErr error
			timeoutChan := time.After(timeout)
			for {
				select {
				case <-timeoutChan:
					clientChan <- nil
					errorChan <- fmt.Errorf("ssh connection timed out: %s", dialErr)
					return
				case <-doneChan:
					return
				default:
					if client, dialErr = ssh.Dial("tcp", ip+":"+port, config); dialErr == nil {
						clientChan <- client
						errorChan <- nil
						return
					}
					time.Sleep(time.Second)
				}
			}
		}(address.IP, address.Port)
	}

	client := <-clientChan
	err = <-errorChan
	close(doneChan)
	return client, err
}
