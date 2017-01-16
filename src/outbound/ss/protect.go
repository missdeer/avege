// +build !android

package ss

import (
	"net"
	"common"
)

var (
	ProtectSocketPathPrefix string
)

func ProtectSocket(clientConn net.Conn) error {
	tcpConn, ok := clientConn.(*net.TCPConn)
	if !ok {
		common.Warning("not a *net.TCPConn")
		return nil
	}
	clientConnFile, err := tcpConn.File()
	if err != nil {
		common.Warning("can't get the File Handle of a *net.TCPConn")
		return nil
	} else {
		tcpConn.Close()
	}
	common.Debug("fd=", int(clientConnFile.Fd()))
	return nil
}
