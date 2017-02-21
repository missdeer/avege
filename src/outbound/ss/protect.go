// +build !android

package ss

import (
	"errors"
	"net"

	"common"
)

var (
	// ProtectSocketPathPrefix prefix of file path that used for Unix socket communication
	ProtectSocketPathPrefix string
)

func ProtectSocket(clientConn net.Conn) (newTCPConn *net.TCPConn, err error) {
	tcpConn, ok := clientConn.(*net.TCPConn)
	if !ok {
		common.Warning("not a *net.TCPConn")
		return nil, errors.New("not a *net.TCPConn")
	}
	clientConnFile, err := tcpConn.File()
	if err != nil {
		// seemly Windows fall through there
		common.Warning("can't get the File Handle of a *net.TCPConn")
		return tcpConn, nil
	} else {
		tcpConn.Close()
	}
	common.Debug("fd=", int(clientConnFile.Fd()))

	newConn, err := net.FileConn(clientConnFile)
	if err != nil {
		return nil, err
	}
	if _, ok := newConn.(*net.TCPConn); ok {
		newTCPConn = newConn.(*net.TCPConn)
		clientConnFile.Close()
	}
	return
}
