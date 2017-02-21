// +build android

package ss

import (
	"errors"
	"net"
	"time"

	"common"
	"github.com/ftrvxmtrx/fd"
)

var (
	// ProtectSocketPathPrefix prefix of file path that used for Unix socket communication
	ProtectSocketPathPrefix string
	ErrNotTCPConn           = errors.New("not a *net.TCPConn")
	ErrNotUnixConn          = errors.New("not a *net.UnixConn")
)

func ProtectSocket(clientConn net.Conn) (newTCPConn *net.TCPConn, err error) {
	tcpConn, ok := clientConn.(*net.TCPConn)
	if !ok {
		common.Warning("not a *net.TCPConn")
		return nil, ErrNotTCPConn
	}
	clientConnFile, err := tcpConn.File()
	if err != nil {
		common.Warning("can't get the File Handle of a *net.TCPConn")
		return tcpConn, err
	} else {
		tcpConn.Close()
	}

	c, e := net.Dial("unix", ProtectSocketPathPrefix+"/protect_path")
	if e != nil {
		common.Error("dialing unix failed:", e)
		return nil, e
	}
	defer c.Close()

	u, ok := c.(*net.UnixConn)
	if !ok {
		common.Error("not a *net.UnixConn")
		return nil, ErrNotUnixConn
	}
	if e = fd.Put(u, clientConnFile); e != nil {
		common.Error("sending fd failed:", e)
		return nil, e
	}

	b := make([]byte, 1)
	c.SetReadDeadline(time.Now().Add(1 * time.Second))
	if _, e = c.Read(b); e != nil {
		common.Error("receiving response failed:", e)
		return nil, e
	}

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
