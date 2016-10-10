package ssh

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct{}

func (*SSH) GenerateAddress() (host string, port string, err error) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", "", err
	}
	defer conn.Close()
	address := strings.Split(conn.Addr().String(), ":")
	return address[0], address[1], nil
}

func (s *SSH) GetSSHOutput(command string, ip string, port string, timeout time.Duration) (string, error) {
	client, session, err := s.newSession(ip, port, timeout)
	if err != nil {
		return "", err
	}
	defer client.Close()
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}

func (s *SSH) RunSSHCommand(command string, ip string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) (err error) {
	client, session, err := s.newSession(ip, port, timeout)
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

func (s *SSH) StartSSHSession(ip string, port string, timeout time.Duration, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	client, session, err := s.newSession(ip, port, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr
	if err := session.RequestPty("xterm", 50, 50, modes); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	return session.Wait()
}

func (s *SSH) WaitForSSH(ip string, port string, timeout time.Duration) error {
	_, err := s.waitForSSH(ip, port, timeout)
	return err
}

func (s *SSH) newSession(ip string, port string, timeout time.Duration) (*ssh.Client, *ssh.Session, error) {
	client, err := s.waitForSSH(ip, port, timeout)
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

func (*SSH) waitForSSH(ip string, port string, timeout time.Duration) (client *ssh.Client, err error) {
	config := &ssh.ClientConfig{
		User: "vcap",
		Auth: []ssh.AuthMethod{
			ssh.Password("vcap"),
		},
		Timeout: timeout,
	}

	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return nil, fmt.Errorf("ssh connection timed out: %s", err)
		default:
			if client, err = ssh.Dial("tcp", ip+":"+port, config); err == nil {
				return client, nil
			}
		}
	}
}
