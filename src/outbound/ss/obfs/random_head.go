package obfs

import (
	"math/rand"

	"outbound/ss/ssr"
)

type RandomHead struct {
	ssr.ServerInfoForObfs
	rawTransSent     bool
	rawTransReceived bool
	hasSentHeader    bool
	dataBuffer       []byte
}

func NewRandomHead() *RandomHead {
	p := &RandomHead{}
	return p
}

func (r *RandomHead) SetServerInfo(s *ssr.ServerInfoForObfs) {
	r.ServerInfoForObfs = *s
}

func (r *RandomHead) GetServerInfo() (s *ssr.ServerInfoForObfs) {
	return &r.ServerInfoForObfs
}

func (r *RandomHead) SetData(data interface{}) {

}

func (r *RandomHead) GetData() interface{} {
	return nil
}

func (r *RandomHead) Encode(data []byte) (encodedData []byte, err error) {
	if r.rawTransSent {
		return data, nil
	}

	dataLength := len(data)
	if r.hasSentHeader {
		if dataLength > 0 {
			d := make([]byte, len(r.dataBuffer) + dataLength)
			copy(d, r.dataBuffer)
			copy(d[len(r.dataBuffer):], data)
			r.dataBuffer = d
		} else {
			encodedData = r.dataBuffer
			r.dataBuffer = nil
			r.rawTransSent = true
		}
	} else {
		size := rand.Intn(96) + 8
		encodedData = make([]byte, size)
		rand.Read(encodedData)
		ssr.SetCRC32(encodedData, size)

		d := make([]byte, dataLength)
		copy(d, data)
		r.dataBuffer = d
	}
	r.hasSentHeader = true
	return
}

func (r *RandomHead) Decode(data []byte) (decodedData []byte, needSendBack bool, err error) {
	if r.rawTransReceived {
		return data, false, nil
	}
	r.rawTransReceived = true
	return data, true, nil
}
