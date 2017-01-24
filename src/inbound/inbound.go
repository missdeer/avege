package inbound

import (
	"net"
	"common"
)

type InBound struct {
	Type      string `json:"type"`
	Address   string `json:"address"`
	Parameter string `json:"parameter"`
	Port      int    `json:"port"`
	Timeout   int    `json:"timeout"`
}

type InBoundHander func(conn *net.TCPConn, outboundHander common.OutboundHandler)
