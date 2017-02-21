package ss

import (
	"net"
	"sync"

	"common"
	"common/ds"
	"outbound/ss/obfs"
	"outbound/ss/protocol"
)

// SSTCPConn the struct that override the net.Conn methods
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

func NewSSTCPConn(c net.Conn, cipher *StreamCipher) *SSTCPConn {
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
				b := make([]byte, len(c.left)+postDecryptedDataLen)
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
