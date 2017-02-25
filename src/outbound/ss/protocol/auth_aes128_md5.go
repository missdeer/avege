package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"encoding/base64"
	"encoding/binary"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"common"
	"outbound/ss/ssr"
)

type hmacMethod func(key []byte, data []byte) []byte
type hashDigestMethod func(data []byte) []byte

func init() {
	register("auth_aes128_md5", newAuthAES128MD5)
}

func newAuthAES128MD5() IProtocol {
	a := &authAES128{
		salt:       "auth_aes128_md5",
		hmac:       common.HmacMD5,
		hashDigest: common.MD5Sum,
		packID:     1,
		recvInfo: recvInfo{
			recvID: 1,
		},
	}
	return a
}

type recvInfo struct {
	recvID           uint32
	recvBuffer       []byte
	recvBufferLength int
}

type authAES128 struct {
	ssr.ServerInfoForObfs
	recvInfo
	data          *authData
	hasSentHeader bool
	packID        uint32
	userKey       []byte
	salt          string
	hmac          hmacMethod
	hashDigest    hashDigestMethod
}

func (a *authAES128) SetServerInfo(s *ssr.ServerInfoForObfs) {
	a.ServerInfoForObfs = *s
}

func (a *authAES128) GetServerInfo() (s *ssr.ServerInfoForObfs) {
	return &a.ServerInfoForObfs
}

func (a *authAES128) SetData(data interface{}) {
	if auth, ok := data.(*authData); ok {
		a.data = auth
	}
}

func (a *authAES128) GetData() interface{} {
	if a.data == nil {
		a.data = &authData{}
	}
	return a.data
}

func (a *authAES128) packData(data []byte) (outData []byte) {
	dataLength := len(data)
	randLength := 1
	if dataLength <= 1200 {
		if a.packID > 4 {
			randLength += rand.Intn(32)
		} else {
			if dataLength > 900 {
				randLength += rand.Intn(128)
			} else {
				randLength += rand.Intn(512)
			}
		}
	}

	outLength := randLength + dataLength + 8
	outData = make([]byte, outLength)
	// 0~1, out length
	binary.LittleEndian.PutUint16(outData[0:], uint16(outLength&0xFFFF))
	// 2~3, hmac
	key := make([]byte, len(a.userKey)+4)
	copy(key, a.userKey)
	binary.LittleEndian.PutUint32(key[len(key)-4:], a.packID)
	h := a.hmac(key, outData[0:2])
	copy(outData[2:4], h[:2])
	// 4~rand length+4, rand number
	rand.Read(outData[4 : 4+randLength])
	// 4, rand length
	if randLength < 128 {
		outData[4] = byte(randLength & 0xFF)
	} else {
		// 4, magic number 0xFF
		outData[4] = 0xFF
		// 5~6, rand length
		binary.LittleEndian.PutUint16(outData[5:], uint16(randLength&0xFFFF))
	}
	// rand length+4~out length-4, data
	if dataLength > 0 {
		copy(outData[randLength+4:], data)
	}
	a.packID++
	h = a.hmac(key, outData[:outLength-4])
	copy(outData[outLength-4:], h[:4])
	return
}

func (a *authAES128) packAuthData(data []byte) (outData []byte) {
	dataLength := len(data)
	var randLength int
	if dataLength > 400 {
		randLength = rand.Intn(512)
	} else {
		randLength = rand.Intn(1024)
	}

	dataOffset := randLength + 16 + 4 + 4 + 7
	outLength := dataOffset + dataLength + 4
	outData = make([]byte, outLength)
	encrypt := make([]byte, 24)
	key := make([]byte, a.IVLen+a.KeyLen)
	copy(key, a.IV)
	copy(key[a.IVLen:], a.Key)

	rand.Read(outData[dataOffset-randLength:])

	if a.data.connectionID > 0xFF000000 {
		a.data.clientID = nil
	}
	if len(a.data.clientID) == 0 {
		a.data.clientID = make([]byte, 4)
		rand.Read(a.data.clientID)
		b := make([]byte, 4)
		rand.Read(b)
		a.data.connectionID = binary.LittleEndian.Uint32(b) & 0xFFFFFF
	}
	a.data.connectionID++
	copy(encrypt[4:], a.data.clientID)
	binary.LittleEndian.PutUint32(encrypt[8:], a.data.connectionID)

	now := time.Now().Unix()
	binary.LittleEndian.PutUint32(encrypt[0:4], uint32(now))

	binary.LittleEndian.PutUint16(encrypt[12:], uint16(outLength&0xFFFF))
	binary.LittleEndian.PutUint16(encrypt[14:], uint16(randLength&0xFFFF))

	params := strings.Split(a.Param, ":")
	uid := make([]byte, 4)
	if len(params) >= 2 {
		if userID, err := strconv.ParseUint(params[0], 10, 32); err != nil {
			common.Warning("parsing uint failed", params[0], err)
			rand.Read(uid)
		} else {
			binary.LittleEndian.PutUint32(uid, uint32(userID))
			a.userKey = a.hashDigest([]byte(params[1]))
		}
	} else {
		rand.Read(uid)
	}

	if a.userKey == nil {
		a.userKey = make([]byte, a.KeyLen)
		copy(a.userKey, a.Key)
	}

	encryptKey := make([]byte, len(a.userKey))
	copy(encryptKey, a.userKey)

	aesCipherKey := common.EVPBytesToKey(base64.StdEncoding.EncodeToString(encryptKey)+a.salt, 16)
	block, err := aes.NewCipher(aesCipherKey)
	if err != nil {
		common.Error("creating aes cipher failed", err)
		return nil
	}

	encryptData := make([]byte, 16)
	iv := make([]byte, aes.BlockSize)
	cbc := cipher.NewCBCEncrypter(block, iv)
	cbc.CryptBlocks(encryptData, encrypt[0:16])
	copy(encrypt[4:4+16], encryptData)
	copy(encrypt[0:4], uid)

	h := a.hmac(key, encrypt[0:20])
	copy(encrypt[20:], h[:4])

	rand.Read(outData[0:1])
	h = a.hmac(key, outData[0:1])
	copy(outData[1:], h[0:7-1])

	copy(outData[7:], encrypt)
	copy(outData[dataOffset:], data)

	h = a.hmac(a.userKey, outData[0:outLength-4])
	copy(outData[outLength-4:], h[:4])

	return
}

func (a *authAES128) PreEncrypt(plainData []byte) (outData []byte, err error) {
	dataLength := len(plainData)
	offset := 0
	if !a.hasSentHeader {
		authLength := dataLength
		if authLength > 1200 {
			authLength = 1200
		}
		packData := a.packAuthData(plainData[:authLength])
		a.hasSentHeader = true
		outData = append(outData, packData...)
		dataLength -= authLength
		offset += authLength
	}
	const blockSize = 4096
	for dataLength > blockSize {
		packData := a.packData(plainData[offset : offset+blockSize])
		outData = append(outData, packData...)
		dataLength -= blockSize
		offset += blockSize
	}
	if dataLength > 0 {
		packData := a.packData(plainData[offset:])
		outData = append(outData, packData...)
	}

	return
}

func (a *authAES128) PostDecrypt(plainData []byte) (outData []byte, err error) {
	dataLength := len(plainData)
	b := make([]byte, len(a.recvBuffer)+dataLength)
	copy(b, a.recvBuffer)
	copy(b[len(a.recvBuffer):], plainData)
	a.recvBuffer = b
	a.recvBufferLength = len(a.recvBuffer)
	key := make([]byte, len(a.userKey)+4)
	copy(key, a.userKey)
	for a.recvBufferLength > 4 {
		binary.LittleEndian.PutUint32(key[len(key)-4:], a.recvID)

		h := a.hmac(key, a.recvBuffer[0:2])
		if h[0] != a.recvBuffer[2] || h[1] != a.recvBuffer[3] {
			common.Error("client post decrypt hmac error")
			return nil, ssr.ErrAuthAES128HMACError
		}

		length := int(binary.LittleEndian.Uint16(a.recvBuffer[0:2]))
		if length >= 8192 || length < 8 {
			common.Error("client post decrypt length mismatch")
			return nil, ssr.ErrAuthAES128DataLengthError
		}

		if length > a.recvBufferLength {
			break
		}

		h = a.hmac(key, a.recvBuffer[0:length-4])
		if !hmac.Equal(h[0:4], a.recvBuffer[length-4:]) {
			common.Error("client post decrypt incorrect checksum")
			return nil, ssr.ErrAuthAES128IncorrectChecksum
		}

		a.recvID++
		pos := int(a.recvBuffer[4])
		if pos != 0xFF {
			pos += 4
		} else {
			pos = int(binary.LittleEndian.Uint16(a.recvBuffer[5:5+2])) + 4
		}
		outData = append(outData, a.recvBuffer[pos:length-4]...)
		a.recvBuffer = a.recvBuffer[length:]
		a.recvBufferLength -= length
	}

	return
}
