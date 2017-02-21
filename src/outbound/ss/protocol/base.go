package protocol

import (
	"strings"

	"outbound/ss/ssr"
)

type creator func() IProtocol

var (
	creatorMap = make(map[string]creator)
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

func register(name string, c creator) {
	creatorMap[name] = c
}

func NewProtocol(name string) IProtocol {
	c, ok := creatorMap[strings.ToLower(name)]
	if ok {
		return c()
	}
	return nil
}
