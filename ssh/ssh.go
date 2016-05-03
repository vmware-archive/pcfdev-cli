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

func (s *SSH) RunSSHCommand(command string, port string, timeout time.Duration) (output []byte, err error) {
	config := &ssh.ClientConfig{
		User: "vcap",
		Auth: []ssh.AuthMethod{
			ssh.Password("vcap"),
		},
		Timeout: 30 * time.Second,
	}

	client, err := s.waitForSSH(config, port, timeout)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	return session.Output(command)
}

func (*SSH) waitForSSH(config *ssh.ClientConfig, port string, timeout time.Duration) (client *ssh.Client, err error) {
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
