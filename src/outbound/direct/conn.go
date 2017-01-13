package direct

import (
	"io"
	"net"
)

func Pipe(conn *net.TCPConn, addr string) error {
	remote, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go io.Copy(conn, remote)
	go io.Copy(remote, conn)
	return nil
}
