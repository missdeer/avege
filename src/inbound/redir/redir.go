// +build windows solaris

package redir

import (
	"common"
	"net"
)

func handleInbound(conn *net.TCPConn, outboundHander common.OutboundHandler) {
	conn.Close()
	common.Panicf("redir connect from %s, unsupported on this platform, only Linux is supported now\n",
		conn.RemoteAddr().String())
}

func GetInboundHandler(inbound *inbound.InBound) inbound.InBoundHander {
	return handleInbound
}
