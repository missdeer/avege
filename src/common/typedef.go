package common

import (
	"bytes"
	"net"
)

type TCPOutboundHandler func(net.Conn, []byte, string) error
type UDPOutboundHandler func(net.PacketConn, []byte, string) error

type Buffer []*bytes.Buffer

type WebsocketMessage struct {
	Cmd    int    `json:"cmd"`
	WParam string `json:"wparam"`
	LParam string `json:"lparam"`
}

const (
	CMD_ERROR = iota
	CMD_RESPONSE
	CMD_AUTH
	CMD_START_REVERSE_SSH
	CMD_REVERSE_SSH_STARTED
	CMD_STOP_REVERSE_SSH
	CMD_REVERSE_SSH_STOPPED
	CMD_NEW_RULES
	CMD_ADD_SERVER
	CMD_DEL_SERVER
	CMD_SET_PORT
	CMD_SET_KEY
	CMD_SET_METHOD
)
