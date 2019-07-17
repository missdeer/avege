package ss

import (
	"errors"
	"io"
	"net"
)

// ErrShortPacket means the packet is too short to be a valid encrypted packet.
var ErrShortPacket = errors.New("short packet")

// Pack encrypts plaintext using stream cipher s and a random IV.
// Returns a slice of dst containing random IV and ciphertext.
// Ensure len(dst) >= s.IVSize() + len(plaintext).
func Pack(dst, plaintext []byte, s *StreamCipher) ([]byte, error) {
	if len(dst) < s.info.ivLen+len(plaintext) {
		return nil, io.ErrShortBuffer
	}
	iv, err := s.initEncrypt()
	if err != nil {
		return nil, err
	}
	copy(dst[:s.info.ivLen], iv)
	s.encrypt(dst[s.info.ivLen:], plaintext)
	return dst[:s.info.ivLen+len(plaintext)], nil
}

// Unpack decrypts pkt using stream cipher s.
// Returns a slice of dst containing decrypted plaintext.
func Unpack(dst, pkt []byte, s *StreamCipher) ([]byte, error) {
	if len(pkt) < s.info.ivLen {
		return nil, ErrShortPacket
	}

	if len(dst) < len(pkt)-s.info.ivLen {
		return nil, io.ErrShortBuffer
	}

	iv := pkt[:s.info.ivLen]
	s.initDecrypt(iv)
	s.decrypt(dst, pkt[s.info.ivLen:])
	return dst[:len(pkt)-s.info.ivLen], nil
}

type SSUDPConn struct {
	net.PacketConn
	*StreamCipher
}

// NewPacketConn wraps a net.PacketConn with stream cipher encryption/decryption.
func NewSSUDPConn(c net.PacketConn, cipher *StreamCipher) net.PacketConn {
	return &SSUDPConn{PacketConn: c, StreamCipher: cipher}
}

func (c *SSUDPConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := make([]byte, len(c.iv)+len(b))
	_, err := Pack(buf, b, c.StreamCipher)
	if err != nil {
		return 0, err
	}
	_, err = c.PacketConn.WriteTo(buf, addr)
	return len(b), err
}

func (c *SSUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	b, err = Unpack(b, b[:n], c.StreamCipher)
	return len(b), addr, err
}
