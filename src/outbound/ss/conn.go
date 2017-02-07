package ss

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"common"
	"common/ds"
	"outbound/ss/obfs"
	"outbound/ss/protocol"
)

var nl = NATlist{Conns: map[string]*CachedUDPConn{}}

type NATlist struct {
	sync.Mutex
	Conns      map[string]*CachedUDPConn
	AliveConns int
}

func (nl *NATlist) Delete(srcaddr string) {
	nl.Lock()
	defer nl.Unlock()
	c, ok := nl.Conns[srcaddr]
	if ok {
		c.Close()
		delete(nl.Conns, srcaddr)
		nl.AliveConns -= 1
	}
	ReqList = map[string]*ReqNode{} //del all
}

func (nl *NATlist) Get(srcaddr *net.UDPAddr, ss *UDPConn) (c *CachedUDPConn, ok bool, err error) {
	nl.Lock()
	defer nl.Unlock()
	index := srcaddr.String()
	_, ok = nl.Conns[index]
	if !ok {
		//NAT not exists or expired
		common.Debugf("new udp conn %v<-->%v\n", srcaddr, ss.LocalAddr())
		nl.AliveConns += 1
		ok = false
		//full cone
		addr, _ := net.ResolveUDPAddr("udp", ":0")
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, false, err
		}
		c = NewCachedUDPConn(conn)
		nl.Conns[index] = c
		c.SetTimer(index)
		go Pipeloop(ss, srcaddr, c)
	} else {
		//NAT exists
		c, _ = nl.Conns[index]
		c.Refresh()
	}
	err = nil
	return
}

const (
	idType  = 0 // address type index
	idIP0   = 1 // ip address start index
	idDmLen = 1 // domain address length index
	idDm0   = 2 // domain address start index

	typeIPv4 = 1 // type is ipv4 address
	typeDm   = 3 // type is domain address
	typeIPv6 = 4 // type is ipv6 address

	lenIPv4   = 1 + net.IPv4len + 2 // 1addrType + ipv4 + 2port
	lenIPv6   = 1 + net.IPv6len + 2 // 1addrType + ipv6 + 2port
	lenDmBase = 1 + 1 + 2           // 1addrType + 1addrLen + 2port, plus addrLen
)

type UDP interface {
	ReadFromUDP(b []byte) (n int, src *net.UDPAddr, err error)
	Read(b []byte) (n int, err error)
	WriteToUDP(b []byte, src *net.UDPAddr) (n int, err error)
	Write(b []byte) (n int, err error)
	Close() error
	SetWriteDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	ReadFrom(b []byte) (int, net.Addr, error)
}

type UDPConn struct {
	UDP
	*StreamCipher
}

func NewUDPConn(cn UDP, cipher *StreamCipher) *UDPConn {
	return &UDPConn{cn, cipher}
}

type CachedUDPConn struct {
	timer *time.Timer
	UDP
	i     string
}

func NewCachedUDPConn(cn UDP) *CachedUDPConn {
	return &CachedUDPConn{nil, cn, ""}
}

func (c *CachedUDPConn) Check() {
	nl.Delete(c.i)
}

func (c *CachedUDPConn) Close() error {
	c.timer.Stop()
	return c.UDP.Close()
}

func (c *CachedUDPConn) SetTimer(index string) {
	c.i = index
	c.timer = time.AfterFunc(120*time.Second, c.Check)
}

func (c *CachedUDPConn) Refresh() bool {
	return c.timer.Reset(120 * time.Second)
}

func ParseHeader(addr net.Addr) []byte {
	//what if the request address type is domain???
	ip, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return nil
	}
	buf := make([]byte, 20)
	IP := net.ParseIP(ip)
	b1 := IP.To4()
	iplen := 0
	if b1 == nil { //ipv6
		b1 = IP.To16()
		buf[0] = typeIPv6
		iplen = net.IPv6len
	} else { //ipv4
		buf[0] = typeIPv4
		iplen = net.IPv4len
	}
	copy(buf[1:], b1)
	port_i, _ := strconv.Atoi(port)
	binary.BigEndian.PutUint16(buf[1 + iplen:], uint16(port_i))
	return buf[:1 + iplen + 2]
}

func Pipeloop(ss *UDPConn, srcaddr *net.UDPAddr, remote UDP) {
	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	defer nl.Delete(srcaddr.String())
	for {
		n, raddr, err := remote.ReadFrom(buf)
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
				// log too many open file error
				// EMFILE is process reaches open file limits, ENFILE is system limit
				common.Error("[udp]read error:", err)
			} else if ne.Err.Error() == "use of closed network connection" {
				common.Debug("[udp]Connection Closing:", remote.LocalAddr())
			} else {
				common.Error("[udp]error reading from:", remote.LocalAddr(), err)
			}
			return
		}
		// need improvement here
		ReqListLock.RLock()
		N, ok := ReqList[raddr.String()]
		ReqListLock.RUnlock()
		if ok {
			ss.WriteToUDP(append(N.Req, buf[:n]...), srcaddr)
		} else {
			header := ParseHeader(raddr)
			ss.WriteToUDP(append(header, buf[:n]...), srcaddr)
		}
		// traffic statistic
	}
}

type ReqNode struct {
	Req    []byte
	ReqLen int
}

var ReqListLock sync.RWMutex
var ReqList = map[string]*ReqNode{}

func HandleUDPConnection(c *UDPConn, openvpn string) {
	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)
	for {
		n, src, err := c.ReadFromUDP(buf)
		if err != nil {
			return
		}

		var dstIP net.IP
		var reqLen int

		switch buf[idType] {
		case typeIPv4:
			reqLen = lenIPv4
			dstIP = net.IP(buf[idIP0: idIP0 + net.IPv4len])
		case typeIPv6:
			reqLen = lenIPv6
			dstIP = net.IP(buf[idIP0: idIP0 + net.IPv6len])
		case typeDm:
			reqLen = int(buf[idDmLen]) + lenDmBase
			dIP, err := net.ResolveIPAddr("ip", string(buf[idDm0:idDm0 + buf[idDmLen]]))
			if err != nil {
				fmt.Sprintf("[udp]failed to resolve domain name: %s\n", string(buf[idDm0:idDm0 + buf[idDmLen]]))
				return
			}
			dstIP = dIP.IP
		default:
			fmt.Sprintf("[udp]addr type %d not supported", buf[idType])
			return
		}
		ip := dstIP.String()
		p := strconv.Itoa(int(binary.BigEndian.Uint16(buf[reqLen - 2: reqLen])))
		if (strings.HasPrefix(ip, "127.") && (p != "1194" || openvpn != "ok")) ||
			strings.HasPrefix(ip, "10.8.") || ip == "::1" {
			common.Infof("[udp]illegal connect to local network(%s)\n", ip)
			return
		}
		dst, _ := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, p))
		ReqListLock.Lock()
		if _, ok := ReqList[dst.String()]; !ok {
			req := make([]byte, reqLen)
			copy(req, buf)
			ReqList[dst.String()] = &ReqNode{req, reqLen}
		}
		ReqListLock.Unlock()

		remote, _, err := nl.Get(src, c)
		if err != nil {
			return
		}
		_, err = remote.WriteToUDP(buf[reqLen:n], dst)
		if err != nil {
			if ne, ok := err.(*net.OpError); ok && (ne.Err == syscall.EMFILE || ne.Err == syscall.ENFILE) {
				// log too many open file error
				// EMFILE is process reaches open file limits, ENFILE is system limit
				common.Error("[udp]write error:", err)
			} else {
				common.Error("[udp]error connecting to:", dst, err)
			}
			return
		}
		// traffic statistic
		// Pipeloop
	} // for
}

//n is the size of the payload
func (c *UDPConn) ReadFromUDP(b []byte) (n int, src *net.UDPAddr, err error) {
	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)

	n, src, err = c.UDP.ReadFromUDP(buf)
	if err != nil {
		return
	}

	iv := buf[:c.info.ivLen]
	if err = c.initDecrypt(iv); err != nil {
		return
	}
	c.decrypt(b[0:n - c.info.ivLen], buf[c.info.ivLen:n])
	n = n - c.info.ivLen
	return
}

func (c *UDPConn) Read(b []byte) (n int, err error) {
	buf := ds.GlobalLeakyBuf.Get()
	defer ds.GlobalLeakyBuf.Put(buf)

	n, err = c.UDP.Read(buf)
	if err != nil {
		return
	}

	iv := buf[:c.info.ivLen]
	if err = c.initDecrypt(iv); err != nil {
		return
	}
	c.decrypt(b[0:n - c.info.ivLen], buf[c.info.ivLen:n])
	n = n - c.info.ivLen
	return
}

//n = iv + payload
func (c *UDPConn) WriteToUDP(b []byte, src *net.UDPAddr) (n int, err error) {
	var cipherData []byte
	dataStart := 0

	var iv []byte
	iv, err = c.initEncrypt()
	if err != nil {
		return
	}
	// Put initialization vector in buffer, do a single write to send both
	// iv and data.
	cipherData = make([]byte, len(b) + len(iv))
	copy(cipherData, iv)
	dataStart = len(iv)

	c.encrypt(cipherData[dataStart:], b)
	n, err = c.UDP.WriteToUDP(cipherData, src)
	return
}

func (c *UDPConn) Write(b []byte) (n int, err error) {
	var cipherData []byte
	dataStart := 0

	var iv []byte
	iv, err = c.initEncrypt()
	if err != nil {
		return
	}
	// Put initialization vector in buffer, do a single write to send both
	// iv and data.
	cipherData = make([]byte, len(b) + len(iv))
	copy(cipherData, iv)
	dataStart = len(iv)

	c.encrypt(cipherData[dataStart:], b)
	n, err = c.UDP.Write(cipherData)
	return
}

type SSTCPConn struct {
	net.Conn
	sync.RWMutex
	*StreamCipher
	IObfs         obfs.IObfs
	IProtocol     protocol.IProtocol
	left          []byte
	readBuf       []byte
	writeBuf      []byte
	lastReadError error
}

func NewSSConn(c net.Conn, cipher *StreamCipher) *SSTCPConn {
	return &SSTCPConn{
		Conn:         c,
		StreamCipher: cipher,
		readBuf:      ds.GlobalLeakyBuf.Get(),
		writeBuf:     ds.GlobalLeakyBuf.Get()}
}

func (c *SSTCPConn) Close() error {
	ds.GlobalLeakyBuf.Put(c.readBuf)
	ds.GlobalLeakyBuf.Put(c.writeBuf)
	return c.Conn.Close()
}

func (c *SSTCPConn) GetIv() (iv []byte) {
	iv = make([]byte, len(c.iv))
	copy(iv, c.iv)
	return
}

func (c *SSTCPConn) GetKey() (key []byte) {
	key = make([]byte, len(c.key))
	copy(key, c.key)
	return
}

func (c *SSTCPConn) initEncryptor(b []byte) (iv []byte, err error) {
	if c.enc == nil {
		iv, err = c.initEncrypt()
		if err != nil {
			common.Error("generating IV failed", err)
			return nil, err
		}

		// should initialize obfs/protocol now, because iv is ready now
		obfsServerInfo := c.IObfs.GetServerInfo()
		obfsServerInfo.SetHeadLen(b, 30)
		obfsServerInfo.IV, obfsServerInfo.IVLen = c.IV()
		obfsServerInfo.Key, obfsServerInfo.KeyLen = c.Key()
		c.IObfs.SetServerInfo(obfsServerInfo)

		protocolServerInfo := c.IProtocol.GetServerInfo()
		protocolServerInfo.SetHeadLen(b, 30)
		protocolServerInfo.IV, protocolServerInfo.IVLen = c.IV()
		protocolServerInfo.Key, protocolServerInfo.KeyLen = c.Key()
		c.IProtocol.SetServerInfo(protocolServerInfo)
	}
	return
}

func (c *SSTCPConn) doRead() (err error) {
	if c.lastReadError != nil {
		return c.lastReadError
	}
	c.Lock()
	defer c.Unlock()
	inData := c.readBuf
	var n int
	n, c.lastReadError = c.Conn.Read(inData)
	if n > 0 {
		var decodedData []byte
		var needSendBack bool
		decodedData, needSendBack, err = c.IObfs.Decode(inData[:n])
		if err != nil {
			return
		}

		if needSendBack {
			common.Debug("do send back")
			//buf := c.IObfs.Encode(make([]byte, 0))
			//c.Conn.Write(buf)
			c.Write(make([]byte, 0))
			return nil
		}

		if decodedDataLen := len(decodedData); decodedDataLen > 0 {
			if c.dec == nil {
				iv := decodedData[0:c.info.ivLen]
				if err = c.initDecrypt(iv); err != nil {
					common.Error("init decrypt failed", err)
					return err
				}

				if len(c.iv) == 0 {
					c.iv = iv
				}
				decodedDataLen -= c.info.ivLen
				decodedData = decodedData[c.info.ivLen:]
			}
			//c.decrypt(b[0:n], inData[0:n])
			buf := make([]byte, decodedDataLen)
			c.decrypt(buf, decodedData)

			var postDecryptedData []byte
			postDecryptedData, err = c.IProtocol.PostDecrypt(buf)
			if err != nil {
				return
			}
			postDecryptedDataLen := len(postDecryptedData)
			if postDecryptedDataLen > 0 {
				b := make([]byte, len(c.left) + postDecryptedDataLen)
				copy(b, c.left)
				copy(b[len(c.left):], postDecryptedData)
				c.left = b
				return
			}
		}
	}
	return
}

func (c *SSTCPConn) Read(b []byte) (n int, err error) {
	c.RLock()
	leftLength := len(c.left)
	c.RUnlock()
	if leftLength == 0 {
		if err = c.doRead(); err != nil {
			return 0, err
		}
	}
	if c.lastReadError != nil {
		defer func() {
			go c.doRead()
		}()
	}

	if leftLength := len(c.left); leftLength > 0 {
		maxLength := len(b)
		if leftLength > maxLength {
			c.Lock()
			copy(b, c.left[:maxLength])
			c.left = c.left[maxLength:]
			c.Unlock()
			return maxLength, nil
		}

		c.Lock()
		copy(b, c.left)
		c.left = nil
		c.Unlock()
		return leftLength, c.lastReadError
	}
	return 0, c.lastReadError
}

func (c *SSTCPConn) preWrite(b []byte) (outData []byte, err error) {
	var iv []byte
	if iv, err = c.initEncryptor(b); err != nil {
		return
	}

	var preEncryptedData []byte
	preEncryptedData, err = c.IProtocol.PreEncrypt(b)
	if err != nil {
		return
	}
	preEncryptedDataLen := len(preEncryptedData)
	//c.encrypt(cipherData[len(iv):], b)
	encryptedData := make([]byte, preEncryptedDataLen)
	//! \attention here the expected output buffer length MUST be accurate, it is preEncryptedDataLen now!
	c.encrypt(encryptedData[0:preEncryptedDataLen], preEncryptedData)

	//common.Info("len(b)=", len(b), ", b:", b,
	//	", pre encrypted data length:", preEncryptedDataLen,
	//	", pre encrypted data:", preEncryptedData,
	//	", encrypted data length:", preEncryptedDataLen)

	cipherData := c.writeBuf
	dataSize := len(encryptedData) + len(iv)
	if dataSize > len(cipherData) {
		cipherData = make([]byte, dataSize)
	} else {
		cipherData = cipherData[:dataSize]
	}

	if iv != nil {
		// Put initialization vector in buffer before be encoded
		copy(cipherData, iv)
	}
	copy(cipherData[len(iv):], encryptedData)

	return c.IObfs.Encode(cipherData)
}

func (c *SSTCPConn) Write(b []byte) (n int, err error) {
	outData, err := c.preWrite(b)
	if err == nil {
		n, err = c.Conn.Write(outData)
	}
	return
}
