package protocol

import (
	"common"
)

type Origin struct {
	common.ServerInfoForObfs
}

func NewOrigin() *Origin {
	a := &Origin{}
	return a
}

func (o *Origin) SetServerInfo(s *common.ServerInfoForObfs) {
	o.ServerInfoForObfs = *s
}

func (o *Origin) GetServerInfo() (s *common.ServerInfoForObfs) {
	return &o.ServerInfoForObfs
}

func (o *Origin) PreEncrypt(data []byte) (encryptedData []byte, err error) {
	return data, nil
}

func (o *Origin) PostDecrypt(data []byte) (decryptedData []byte, err error) {
	return data, nil
}

func (o *Origin) SetData(data interface{}) {

}

func (o *Origin) GetData() interface{} {
	return nil
}
