package mocks

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	User              string
	Password          string
	Host              string
	Port              string
	CommandChan       chan string
	CommandExitStatus byte
	RejectSession     bool
	Data              *gbytes.Buffer
	listener          net.Listener
	closeChan         chan struct{}
}

func (s *SSHServer) Start() {
	Expect(s.listener).To(BeNil(), "test server already started")

	var err error
	address := fmt.Sprintf("%s:%s", s.Host, s.Port)
	s.listener, err = net.Listen("tcp", address)

	Expect(err).NotTo(HaveOccurred())
	s.closeChan = make(chan struct{})

	s.CommandChan = make(chan string)
	s.Data = gbytes.NewBuffer()

	go s.listen()
}

func (s *SSHServer) Stop() {
	if s.listener == nil {
		return
	}
	Expect(s.listener.Close()).To(Succeed())
	<-s.closeChan
	s.listener = nil
}

func (s *SSHServer) listen() {
	defer GinkgoRecover()
	defer func() {
		close(s.closeChan)
	}()

	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == s.User && string(pass) == s.Password {
				return nil, nil
			}
			return nil, errors.New("invalid credentials")
		},
	}

	privateKey, err := ssh.ParsePrivateKey([]byte(sshPrivateKey))
	Expect(err).NotTo(HaveOccurred())
	config.AddHostKey(privateKey)

	for {
		tcpConn, err := s.listener.Accept()
		if err != nil {
			return
		}

		_, newChannels, requests, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			return
		}

		go ssh.DiscardRequests(requests)
		go func() {
			for newChannel := range newChannels {
				go s.handleChannel(newChannel)
			}
		}()
	}
}

func (s *SSHServer) handleChannel(newChannel ssh.NewChannel) {
	defer GinkgoRecover()

	Expect(newChannel.ChannelType()).To(Equal("session"))

	if s.RejectSession {
		Expect(newChannel.Reject(ssh.ConnectionFailed, "session rejected")).To(Succeed())
		return
	}

	channel, requests, err := newChannel.Accept()
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()

		for request := range requests {
			switch request.Type {
			case "exec":
				payloadLen := binary.BigEndian.Uint32(request.Payload[:4])
				Expect(request.Payload).To(HaveLen(int(payloadLen) + 4))

				s.CommandChan <- string(request.Payload[4:])

				Expect(request.Reply(true, nil)).To(Succeed())

				_, err := channel.SendRequest("exit-status", false, []byte{0, 0, 0, s.CommandExitStatus})
				Expect(err).To(Succeed())

				channel.Close()
				break
			}
		}
	}()
	go func() {
		defer GinkgoRecover()
		io.Copy(s.Data, channel)
	}()
}
