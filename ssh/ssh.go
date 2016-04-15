package ssh

import (
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct{}

func (*SSH) FreePort() (string, error) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	return strings.Split(conn.Addr().String(), ":")[1], nil
}

func (s *SSH) RunSSHCommand(command string, port string) error {
	config := &ssh.ClientConfig{
		User: "vagrant",
		Auth: []ssh.AuthMethod{
			ssh.Password("vagrant"),
		},
	}

	var err error
	client := s.WaitForSSH(config, port)
	session, err := client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	if err := session.Run(command); err != nil {
		return err
	}
	return nil
}

func (*SSH) WaitForSSH(config *ssh.ClientConfig, port string) *ssh.Client {
	var client *ssh.Client
	var err error
	for client, err = ssh.Dial("tcp", "127.0.0.1:"+port, config); err != nil; {
		time.Sleep(time.Second)
	}
	return client
}
