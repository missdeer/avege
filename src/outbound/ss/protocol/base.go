package protocol

import (
	"outbound/ss/ssr"
)

type IProtocol interface {
	SetServerInfo(s *ssr.ServerInfoForObfs)
	GetServerInfo() *ssr.ServerInfoForObfs
	PreEncrypt(data []byte) (encryptedData []byte, err error)
	PostDecrypt(data []byte) (decryptedData []byte, err error)
	SetData(data interface{})
	GetData() interface{}
}

type authData struct {
	clientID     []byte
	connectionID uint32
}

func NewProtocol(name string) IProtocol {
	switch name {
	case "origin":
		return NewOrigin()
	case "auth_sha1_v4":
		return NewAuthSHA1v4()
	case "auth_aes128_md5":
		return NewAuthAES128MD5()
	case "auth_aes128_sha1":
		return NewAuthAES128SHA1()
	case "ota", "verify_sha1":
		return NewVerifySHA1()
	}
	return nil
}
