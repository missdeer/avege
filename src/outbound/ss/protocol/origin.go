package protocol

import (
	"outbound/ss/ssr"
)

func init() {
	register("origin", newOrigin)
}

type origin struct {
	ssr.ServerInfoForObfs
}

func newOrigin() IProtocol {
	a := &origin{}
	return a
}

func (o *origin) SetServerInfo(s *ssr.ServerInfoForObfs) {
	o.ServerInfoForObfs = *s
}

func (o *origin) GetServerInfo() (s *ssr.ServerInfoForObfs) {
	return &o.ServerInfoForObfs
}

func (o *origin) PreEncrypt(data []byte) (encryptedData []byte, err error) {
	return data, nil
}

func (o *origin) PostDecrypt(data []byte) (decryptedData []byte, err error) {
	return data, nil
}

func (o *origin) SetData(data interface{}) {

}

func (o *origin) GetData() interface{} {
	return nil
}
