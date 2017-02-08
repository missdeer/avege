package inbound

import (
	"common"
	"net"
)

type InBound struct {
	Type      string `json:"type"`
	Address   string `json:"address"`
	Parameter string `json:"parameter"`
	Port      int    `json:"port"`
}

type InBoundHander func(conn *net.TCPConn, outboundHander common.OutboundHandler)

const (
	inBoundNone = 0
)
const (
	inBoundSocks5 = 1 << iota
	inBoundRedir
	inBoundTunnel
)

var (
	inBoundModesEnabled = inBoundNone
)

func InBoundModeEnable(inboundType string) {
	switch inboundType {
	case "socks5", "socks":
		inBoundModesEnabled |= inBoundSocks5
	case "redir":
		inBoundModesEnabled |= inBoundRedir
	case "tunnel":
		inBoundModesEnabled |= inBoundTunnel
	}
}

func HasInBound() bool {
	return inBoundModesEnabled != inBoundNone
}

func IsInBoundModeEnabled(inboundType string) bool {
	switch inboundType {
	case "socks5", "socks":
		return (inBoundModesEnabled & inBoundSocks5) != 0
	case "redir":
		return (inBoundModesEnabled & inBoundRedir) != 0
	case "tunnel":
		return (inBoundModesEnabled & inBoundTunnel) != 0
	}
	return false
}
