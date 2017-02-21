package local

import (
	"net"
	"sync"
	"time"

	"common"
	"outbound/ss"
)

const (
	udpBufSize = 64 * 1024
	udpTimeout = 5 * time.Minute
)

var (
	nm = newNATmap(udpTimeout)
)

func handleUDPOutbound(c net.PacketConn, rawaddr []byte, addr string) error {
	buf := make([]byte, udpBufSize)
	copy(buf, rawaddr)

	n, raddr, err := c.ReadFrom(buf)
	if err != nil {
		common.Error("UDP local read error:", err)
		return err
	}

	pc := nm.Get(raddr.String())
	if pc == nil {
		pc, err = net.ListenPacket("udp", "")
		if err != nil {
			common.Error("UDP local listen error:", err)
			return err
		}

		var cipher *ss.StreamCipher //! \TODO from loadbalance
		pc = ss.NewSSUDPConn(pc, cipher)
		nm.Add(raddr, c, pc)
	}

	var srvAddr net.Addr //! \TODO from loadbalance
	_, err = pc.WriteTo(buf[:len(rawaddr)+n], srvAddr)
	if err != nil {
		common.Error("UDP local write error:", err)
		return err
	}
	return nil
}

// Packet NAT table
type natmap struct {
	sync.RWMutex
	m       map[string]net.PacketConn
	timeout time.Duration
}

func newNATmap(timeout time.Duration) *natmap {
	m := &natmap{}
	m.m = make(map[string]net.PacketConn)
	m.timeout = timeout
	return m
}

func (m *natmap) Get(key string) net.PacketConn {
	m.RLock()
	defer m.RUnlock()
	return m.m[key]
}

func (m *natmap) Set(key string, pc net.PacketConn) {
	m.Lock()
	defer m.Unlock()

	m.m[key] = pc
}

func (m *natmap) Del(key string) net.PacketConn {
	m.Lock()
	defer m.Unlock()

	pc, ok := m.m[key]
	if ok {
		delete(m.m, key)
		return pc
	}
	return nil
}

func (m *natmap) Add(peer net.Addr, dst, src net.PacketConn) {
	m.Set(peer.String(), src)

	go func() {
		timedCopy(dst, peer, src, m.timeout)
		if pc := m.Del(peer.String()); pc != nil {
			pc.Close()
		}
	}()
}

// copy from src to dst with addr with read timeout
func timedCopy(dst net.PacketConn, addr net.Addr, src net.PacketConn, timeout time.Duration) error {
	buf := make([]byte, udpBufSize)

	for {
		src.SetReadDeadline(time.Now().Add(timeout))
		n, _, err := src.ReadFrom(buf)
		if err != nil {
			return err
		}

		_, err = dst.WriteTo(buf[:n], addr)
		if err != nil {
			return err
		}
	}
}
