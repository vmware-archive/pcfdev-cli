package ssh

import (
	"errors"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSH struct{}

func (*SSH) GenerateAddress() (string, string, error) {
	conn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
		return "", "", err
	}
	defer conn.Close()
	address := strings.Split(conn.Addr().String(), ":")
	return address[0], address[1], nil
}

func (s *SSH) RunSSHCommand(command string, port string) error {
	config := &ssh.ClientConfig{
		User: "vagrant",
		Auth: []ssh.AuthMethod{
			ssh.Password("vagrant"),
		},
	}

	client, err := s.WaitForSSH(config, port)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}

	defer session.Close()

	return session.Run(command)
}

func (*SSH) WaitForSSH(config *ssh.ClientConfig, port string) (*ssh.Client, error) {
	successChan := make(chan *ssh.Client)
	timeoutChan := time.After(time.Minute)

	go func() {
		var client *ssh.Client
		var err error
		for client, err = ssh.Dial("tcp", "127.0.0.1:"+port, config); err != nil; {
			time.Sleep(time.Second)
		}
		successChan <- client
	}()

	select {
	case client := <-successChan:
		return client, nil
	case <-timeoutChan:
		return nil, errors.New("ssh connection timed out")
	}
}
