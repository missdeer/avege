package protocol

import "common"

func init() {
	register("auth_aes128_sha1", newAuthAES128SHA1)
}

func newAuthAES128SHA1() IProtocol {
	a := &authAES128{
		salt:       "auth_aes128_sha1",
		hmac:       common.HmacSHA1,
		hashDigest: common.SHA1Sum,
		packID:     1,
		recvInfo: recvInfo{
			recvID: 1,
		},
	}
	return a
}
