package obfs

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"strings"
	"time"

	"github.com/missdeer/avege/common"
	"github.com/missdeer/avege/outbound/ss/ssr"
)

func init() {
	register("tls1.2_ticket_auth", newTLS12TicketAuth)
	register("tls1.2_ticket_fastauth", newTLS12TicketFastAuth)
}

type tlsAuthData struct {
	localClientID [32]byte
}

// tls12TicketAuth tls1.2_ticket_auth obfs encapsulate
type tls12TicketAuth struct {
	ssr.ServerInfoForObfs
	data            *tlsAuthData
	handshakeStatus int
	sendBuffer      [][]byte
	recvLength      int
	fastAuth        bool
}

// newTLS12TicketAuth create a tlv1.2_ticket_auth object
func newTLS12TicketAuth() IObfs {
	return &tls12TicketAuth{}
}

// newTLS12TicketFastAuth create a tlv1.2_ticket_fastauth object
func newTLS12TicketFastAuth() IObfs {
	return &tls12TicketAuth{
		fastAuth: true,
	}
}

func (t *tls12TicketAuth) SetServerInfo(s *ssr.ServerInfoForObfs) {
	t.ServerInfoForObfs = *s
}

func (t *tls12TicketAuth) GetServerInfo() (s *ssr.ServerInfoForObfs) {
	return &t.ServerInfoForObfs
}

func (t *tls12TicketAuth) SetData(data interface{}) {
	if auth, ok := data.(*tlsAuthData); ok {
		t.data = auth
	}
}

func (t *tls12TicketAuth) GetData() interface{} {
	if t.data == nil {
		t.data = &tlsAuthData{}
		b := make([]byte, 32)
		rand.Read(b)
		copy(t.data.localClientID[:], b)
	}
	return t.data
}

func (t *tls12TicketAuth) getHost() string {
	host := t.Host
	if len(t.Param) > 0 {
		hosts := strings.Split(t.Param, ",")
		if len(hosts) > 0 {
			host = hosts[rand.Intn(len(hosts))]
			host = strings.TrimSpace(host)
		}
	}
	if len(host) > 0 && host[len(host)-1] >= byte('0') && host[len(host)-1] <= byte('9') && len(t.Param) == 0 {
		host = ""
	}
	return host
}

func (t *tls12TicketAuth) Encode(data []byte) (encodedData []byte, err error) {
	// t.handshake:
	// bit 1 - Failed
	// bit 2 - Client Hello Sent
	// bit 3 - Client Finish Sent
	// bit 4 - Buffer Cleared
	// bit 5 - Server Hello Received
	if t.handshakeStatus == -1 {
		return data, nil
	}
	// buffer cleared
	if t.handshakeStatus & 4 == 4 {
		d := make([]byte, 0, len(data) + 100)
		for len(data) > 0 {
			length := len(data)
			// 16k record size
			if length > 16384 {
				length = 16384
			}
			d = append(d, 0x17, 0x3, 0x3)
			d = d[:5]
			binary.BigEndian.PutUint16(d[3:5], uint16(length&0xFFFF))
			d = append(d, data[:length]...)
			data = data[:length]
		}
		return d, nil
	}
	// Put data into send buffer
	if len(data) > 0 {
		t.sendBuffer = append(t.sendBuffer, data)
	}
	// Client Hello sent & Client Finished not sent
	if t.handshakeStatus & 3 == 1 {
		// No Server Hello Received & not FastAuth
		if t.handshakeStatus & 8 != 8 && !t.fastAuth {
			return make([]byte, 0), nil
		}
		hmacData := make([]byte, 43)
		handshakeFinish := []byte("\x14\x03\x03\x00\x01\x01\x16\x03\x03\x00\x20")
		copy(hmacData, handshakeFinish)
		rand.Read(hmacData[11:33])
		h := t.hmacSHA1(hmacData[:33])
		copy(hmacData[33:], h)
		t.handshakeStatus |= 2
		if !t.fastAuth {
			return hmacData, nil
		}
		// Clear buffer
		totalLength := 43 + len(t.sendBuffer) * 5 // len(hmacData) + header size of buffers
		for _, buf := range t.sendBuffer {
			totalLength += len(buf)
		}
		d := make([]byte, 0, totalLength)
		d = append(d, hmacData...)
		for _, buf := range t.sendBuffer {
			d = append(d, 0x17, 0x3, 0x3)
			d = d[:len(d)+2]
			binary.BigEndian.PutUint16(d[len(d)-2:], uint16(len(buf)&0xFFFF))
			d = append(d, buf...)
		}
		t.sendBuffer = nil
		t.handshakeStatus |= 4
		return d, nil
	}
	// Client Hello & Client Finish sent
	if t.handshakeStatus & 3 == 3 {
		// Clear buffer
		totalLength := len(t.sendBuffer) * 5 // header size
		for _, buf := range t.sendBuffer {
			totalLength += len(buf)
		}
		d := make([]byte, totalLength)
		for _, buf := range t.sendBuffer {
			d = append(d, 0x17, 0x3, 0x3)
			d = d[:len(d)+2]
			binary.BigEndian.PutUint16(d[len(d)-2:], uint16(len(buf)&0xFFFF))
			d = append(d, buf...)
		}
		t.sendBuffer = nil
		t.handshakeStatus |= 4
		return d, nil
	}
	// Not Started
	tlsData0 := []byte("\x00\x1c\xc0\x2b\xc0\x2f\xcc\xa9\xcc\xa8\xcc\x14\xcc\x13\xc0\x0a\xc0\x14\xc0\x09\xc0\x13\x00\x9c\x00\x35\x00\x2f\x00\x0a\x01\x00")
	tlsData1 := []byte("\xff\x01\x00\x01\x00")
	tlsData2 := []byte("\x00\x17\x00\x00\x00\x23\x00\xd0")
	tlsData3 := []byte("\x00\x0d\x00\x16\x00\x14\x06\x01\x06\x03\x05\x01\x05\x03\x04\x01\x04\x03\x03\x01\x03\x03\x02\x01\x02\x03\x00\x05\x00\x05\x01\x00\x00\x00\x00\x00\x12\x00\x00\x75\x50\x00\x00\x00\x0b\x00\x02\x01\x00\x00\x0a\x00\x06\x00\x04\x00\x17\x00\x18\x00\x15\x00\x66\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")

	sni := t.sni(t.getHost())
	// length + len(tlsData1) + sni length + len(tlsData2) + len(ticket) + len(tlsData3)
	extLen := 2 + 5 + len(sni) + 8 + 208 + 165
	// version + length + handshake protocol + handshake length + client version +
	// len(rnd) + byte(32) + len(t.data.localClientID) + len(tlsData0) + extLen
	sslLen := 3 + 2 + 2 + 2 + 2 + 32 + 1 + 32 + 32 + extLen

	sslBuf := make([]byte, 0, sslLen)
	// version (3)
	sslBuf = append(sslBuf, 0x16, 0x03, 0x01)
	// length (2)
	sslBuf = sslBuf[:5]
	binary.BigEndian.PutUint16(sslBuf[3:], uint16((sslLen - 3)&0xFFFF))
	// hand shake protocol (2)
	sslBuf = append(sslBuf, 0x01, 0x00)
	// hand shake length (2)
	sslBuf = sslBuf[:9]
	binary.BigEndian.PutUint16(sslBuf[7:], uint16((sslLen - 7)&0xFFFF))
	// client version 2
	sslBuf = append(sslBuf, 0x3, 0x3)

	// Auth Data (32)
	sslBuf = append(sslBuf, t.packAuthData()...)
	// Byte (1)
	sslBuf = append(sslBuf, byte(32))
	// localClientID (32)
	sslBuf = append(sslBuf, t.data.localClientID[:]...)
	// tlsData0 (32)
	sslBuf = append(sslBuf, tlsData0...)
	
	// extBuf
	sslBuf = sslBuf[:sslLen]
	// extBuf length (2)
	extBuf := sslBuf[sslLen-extLen:sslLen-extLen + 2]
	binary.BigEndian.PutUint16(extBuf, uint16(cap(extBuf)&0xFFFF))
	// tlsData1 (5)
	extBuf = append(extBuf, tlsData1...)
	// sni (len(sni))
	extBuf = append(extBuf, sni...)
	// tlsData2 (8)
	extBuf = append(extBuf, tlsData2...)
	// ticket (208)
	extBuf = extBuf[:len(extBuf)+208]
	rand.Read(extBuf[len(extBuf)-208:])
	// tlsData3 (165)
	extBuf = append(extBuf, tlsData3...)

	t.handshakeStatus |= 1
	return sslBuf, nil
}

func (t *tls12TicketAuth) Decode(data []byte) (decodedData []byte, needSendBack bool, err error) {
	if t.handshakeStatus == -1 {
		return data, false, nil
	}

	// Server Hello Had Received (Normal)
	if t.handshakeStatus & 8 != 0 {
		var d []byte
		for len(data) > 0 {
			if t.recvLength == 0 {
				if len(data) < 5 {
					common.Error("incomplete tls header")
					break
				}
				if !bytes.Equal(data[0:3], []byte{0x17, 0x3, 0x3}) {
					common.Error("incorrect magic number", data[0:3], ", 0x170303 is expected")
					return nil, false, ssr.ErrTLS12TicketAuthIncorrectMagicNumber
				}
				t.recvLength = int(binary.BigEndian.Uint16(data[3:5]))
				data = data[5:]
				continue
			}
			length := len(data)
			if len(data) > t.recvLength {
				length = t.recvLength
			}
			d = append(d, data[:length]...)
			data = data[:length]
		}
		return d, false, nil
	}

	if len(data) < 11+32+1+32 {
		common.Error("too short data:", len(data))
		return nil, false, ssr.ErrTLS12TicketAuthTooShortData
	}

	hash := t.hmacSHA1(data[11 : 11+22])

	if !bytes.Equal(data[33:33+ssr.ObfsHMACSHA1Len], hash) {
		common.Error("hmac verification failed:", hash, data[33:33+ssr.ObfsHMACSHA1Len], len(data), " bytes recevied:", data)
		return nil, false, ssr.ErrTLS12TicketAuthHMACError
	}
	return nil, true, nil
}

func (t *tls12TicketAuth) packAuthData() (outData []byte) {
	outSize := 32
	outData = make([]byte, outSize)

	now := time.Now().Unix()
	binary.BigEndian.PutUint32(outData[0:4], uint32(now))

	rand.Read(outData[4 : 4+18])

	hash := t.hmacSHA1(outData[:outSize-ssr.ObfsHMACSHA1Len])
	copy(outData[outSize-ssr.ObfsHMACSHA1Len:], hash)

	return
}

func (t *tls12TicketAuth) hmacSHA1(data []byte) []byte {
	key := make([]byte, t.KeyLen+32)
	copy(key, t.Key)
	copy(key[t.KeyLen:], t.data.localClientID[:])

	sha1Data := common.HmacSHA1(key, data)
	return sha1Data[:ssr.ObfsHMACSHA1Len]
}

func (t *tls12TicketAuth) sni(u string) []byte {
	bURL := []byte(u)
	length := len(bURL)
	ret := make([]byte, length+9)
	copy(ret[9:], bURL)
	binary.BigEndian.PutUint16(ret[7:], uint16(length&0xFFFF))
	length += 3
	binary.BigEndian.PutUint16(ret[4:], uint16(length&0xFFFF))
	length += 2
	binary.BigEndian.PutUint16(ret[2:], uint16(length&0xFFFF))
	return ret
}
