package protocol

import "common"

func NewAuthAES128SHA1() *AuthAES128 {
	a := &AuthAES128{
		salt:       "auth_aes128_sha1",
		hmac:       common.HmacSHA1,
		hashDigest: common.SHA1Sum,
		packID:     1,
		recvID:     1,
	}
	return a
}
