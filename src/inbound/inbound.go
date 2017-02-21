package inbound

import (
	"net"
	"strings"

	"common"
)

type Inbound struct {
	Type      string `json:"type"`
	Address   string `json:"address"`
	Parameter string `json:"parameter"`
	Port      int    `json:"port"`
}

type TCPInboundHandler func(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error
type UDPInboundHandler func(conn net.PacketConn, outboundHandler common.UDPOutboundHandler) error

const (
	none = 0
)
const (
	socks5 = 1 << iota
	redir
	tunnel
)

var (
	modesEnabled = none
	modeMap      = map[string]int{
		"socks5": socks5,
		"socks":  socks5,
		"redir":  redir,
		"tunnel": tunnel,
	}
)

// ModeEnable set inbound mode mask
func ModeEnable(inboundType string) {
	mode, ok := modeMap[strings.ToLower(inboundType)]
	if ok {
		modesEnabled |= mode
	}
}

// Has is there any inbound configuration item
func Has() bool {
	return modesEnabled != none
}

// IsModeEnabled is the special inbound mode enabled
func IsModeEnabled(inboundType string) bool {
	mode, ok := modeMap[strings.ToLower(inboundType)]
	if ok {
		return (modesEnabled & mode) != 0
	}
	return false
}
