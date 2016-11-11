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

	"github.com/docker/docker/pkg/term"
	"golang.org/x/crypto/ssh"
)

type SSH struct {
	Terminal      Terminal
	WindowResizer WindowResizer
}

//go:generate mockgen -package mocks -destination mocks/terminal.go github.com/pivotal-cf/pcfdev-cli/ssh Terminal
type Terminal interface {
	SetRawTerminal(fd uintptr) (*term.State, error)
	RestoreTerminal(fd uintptr, state *term.State) error
	GetFdInfo(in interface{}) uintptr
	GetWinSize(fd uintptr) (*term.Winsize, error)
}

//go:generate mockgen -package mocks -destination mocks/windows_resizer.go github.com/pivotal-cf/pcfdev-cli/ssh WindowResizer
type WindowResizer interface {
	StartResizing(session *ssh.Session)
	StopResizing()
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
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	stdinFd := s.Terminal.GetFdInfo(stdin)
	stdoutFd := s.Terminal.GetFdInfo(stdout)

	state, err := s.Terminal.SetRawTerminal(stdinFd)
	if err != nil {
		return err
	}
	defer s.Terminal.RestoreTerminal(stdinFd, state)

	winSize, err := s.Terminal.GetWinSize(stdoutFd)
	if err != nil {
		return err
	}

	if err := session.RequestPty("xterm", int(winSize.Height), int(winSize.Width), modes); err != nil {
		return err
	}

	s.WindowResizer.StartResizing(session)
	defer s.WindowResizer.StopResizing()

	if err := session.Shell(); err != nil {
		return err
	}

	session.Wait()
	return nil
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

func (s *SSH) WithSSHTunnel(remoteAddress string, sshAddresses []SSHAddress, privateKey []byte, timeout time.Duration, block func(forwardingAddress string)) error {
	client, err := s.waitForSSH(sshAddresses, privateKey, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	localListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer localListener.Close()

	var tunnelError error
	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				tunnelError = err
				return
			}

			go func(conn net.Conn) {
				defer conn.Close()

				sshTunnel, err := client.Dial("tcp", remoteAddress)
				if err != nil {
					tunnelError = err
					return
				}
				defer sshTunnel.Close()

				go func() {
					io.Copy(conn, sshTunnel)
				}()

				io.Copy(sshTunnel, conn)
			}(localConn)
		}
	}()

	block("http://" + localListener.Addr().String())
	return tunnelError
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
