package obfs

import (
	"math/rand"
)

// NewHttpPost create a http_post object
func NewHttpPost() *HttpSimplePost {
	// NewHttpSimple create a http_simple object
	t := &HttpSimplePost{
		userAgentIndex: rand.Intn(len(requestUserAgent)),
		getOrPost:      false,
	}
	return t
}
