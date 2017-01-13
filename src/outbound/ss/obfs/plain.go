package obfs

import (
	"common"
)

type PlainObfs struct {
	common.ServerInfoForObfs
}

func NewPlainObfs() *PlainObfs {
	p := &PlainObfs{}
	return p
}

func (p *PlainObfs) SetServerInfo(s *common.ServerInfoForObfs) {
	p.ServerInfoForObfs = *s
}

func (p *PlainObfs) GetServerInfo() (s *common.ServerInfoForObfs) {
	return &p.ServerInfoForObfs
}

func (p *PlainObfs) Encode(data []byte) (encodedData []byte, err error) {
	return data, nil
}

func (p *PlainObfs) Decode(data []byte) (decodedData []byte, needSendBack bool, err error) {
	return data, false, nil
}

func (p *PlainObfs) SetData(data interface{}) {

}

func (p *PlainObfs) GetData() interface{} {
	return nil
}
