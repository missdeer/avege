// +build freebsd dragonfly

package redir

import (
	"log"
	"net"

	"common"
	"inbound"
)

func handleTCPInbound(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error {
	log.Printf("redir connect from %s, FreeBSD/DragonflyBSD is detected, use ipfw now.\n",
		conn.RemoteAddr().String())
	return nil
}

func GetTCPInboundHandler(inbound *inbound.Inbound) inbound.TCPInboundHandler {
	return handleTCPInbound
}

func GetUDPInboundHandler(inbound *inbound.Inbound) inbound.UDPInboundHandler {
	return func(conn net.PacketConn, outboundHandler common.UDPOutboundHandler) error {
		common.Debugf("redir connect from %s\n", conn.LocalAddr().String())
		return nil
	}
}
