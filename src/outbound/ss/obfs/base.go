package obfs

import (
	"outbound/ss/ssr"
)

type IObfs interface {
	SetServerInfo(s *ssr.ServerInfoForObfs)
	GetServerInfo() (s *ssr.ServerInfoForObfs)
	Encode(data []byte) (encodedData []byte, err error)
	Decode(data []byte) (decodedData []byte, needSendBack bool, err error)
	SetData(data interface{})
	GetData() interface{}
}

// NewObfs create an obfs object by name and return as an IObfs interface
func NewObfs(name string) IObfs {
	switch name {
	case "plain":
		return NewPlainObfs()
	case "tls1.2_ticket_auth":
		return NewTLS12TicketAuth()
	case "http_simple":
		return NewHttpSimple()
	case "http_post":
		return NewHttpPost()
	case "random_head":
		return NewRandomHead()
	}
	return nil
}
