// +build windows solaris

package redir

import (
	"net"

	"common"
	"inbound"
)

func handleTCPInbound(conn *net.TCPConn, outboundHandler common.TCPOutboundHandler) error {
	conn.Close()
	common.Panicf("redir connect from %s, unsupported on this platform, only Linux is supported now\n",
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
