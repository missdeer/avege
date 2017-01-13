// +build freebsd dragonfly

package redir

import (
	"common"
	"log"
	"net"
)

func HandleInbound(conn *net.TCPConn, outboundHander common.OutboundHandler) {
	log.Printf("redir connect from %s, FreeBSD/DragonflyBSD is detected, use ipfw now.\n",
		conn.RemoteAddr().String())
}
