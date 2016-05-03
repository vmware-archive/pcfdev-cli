package ssh

import (
	"fmt"
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

func (s *SSH) RunSSHCommand(command string, port string) error {
	config := &ssh.ClientConfig{
		User: "vcap",
		Auth: []ssh.AuthMethod{
			ssh.Password("vcap"),
		},
		Timeout: 30 * time.Second,
	}

	client, err := s.WaitForSSH(config, port, 2*time.Minute)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	return session.Run(command)
}

//TODO: make private

func (*SSH) WaitForSSH(config *ssh.ClientConfig, port string, timeout time.Duration) (client *ssh.Client, err error) {
	timeoutChan := time.After(timeout)

	for {
		select {
		case <-timeoutChan:
			return nil, fmt.Errorf("ssh connection timed out: %s", err)
		default:
			if client, err = ssh.Dial("tcp", "127.0.0.1:"+port, config); err == nil {
				return client, nil
			}
		}
	}
}
