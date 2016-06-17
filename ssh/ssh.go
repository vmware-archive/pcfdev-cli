package ssh

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/pivotal-cf/pcfdev-cli/helpers"

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

func (s *SSH) RunSSHCommand(command string, port string, timeout time.Duration, stdout io.Writer, stderr io.Writer) (err error) {
	config := &ssh.ClientConfig{
		User: "vcap",
		Auth: []ssh.AuthMethod{
			ssh.Password("vcap"),
		},
		Timeout: 30 * time.Second,
	}

	client, err := s.waitForSSH(config, port, timeout)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
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

func (*SSH) waitForSSH(config *ssh.ClientConfig, port string, timeout time.Duration) (client *ssh.Client, err error) {
	err = helpers.ExecuteWithTimeout(func() error {
		if client, err = ssh.Dial("tcp", "127.0.0.1:"+port, config); err != nil {
			return fmt.Errorf("ssh connection timed out: %s", err)
		}
		return nil
	},
		time.Minute,
		0,
	)

	return client, err
}
