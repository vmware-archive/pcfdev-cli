package ping

import (
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Pinger struct{}

func (*Pinger) TryIP(ip string) (bool, error) {
	pingConn, err := icmp.ListenPacket("udp4", "0.0.0.0")
	if err != nil {
		return false, fmt.Errorf("failed to open connection: %s", err)
	}

	defer pingConn.Close()

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
		},
	}
	messageData, err := message.Marshal(nil)
	if err != nil {
		return false, fmt.Errorf("failed to marshal icmp message: %s", err)
	}

	_, err = pingConn.WriteTo(messageData, &net.UDPAddr{IP: net.ParseIP(ip)})
	if err != nil {
		return false, fmt.Errorf("failed to send icmp message: %s", err)
	}
	responseData := make([]byte, 1500)
	pingConn.SetReadDeadline(time.Now().Add(1 * time.Second))
	len, _, err := pingConn.ReadFrom(responseData)
	if err != nil {
		return false, nil
	}

	response, err := icmp.ParseMessage(1, responseData[:len])
	if err != nil {
		return false, fmt.Errorf("badly formatted response: %s", err)
	}

	switch response.Type {
	case ipv4.ICMPTypeEchoReply:
		return true, nil
	default:
		return false, errors.New("ping response did not have type 'echo reply'")
	}
}
