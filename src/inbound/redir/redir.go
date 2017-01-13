// +build windows solaris

package redir

import (
	"common"
	"net"
)

func HandleInbound(conn *net.TCPConn, outboundHander common.OutboundHandler) {
	conn.Close()
	common.Panicf("redir connect from %s, unsupported on this platform, only Linux is supported now\n",
		conn.RemoteAddr().String())
}
