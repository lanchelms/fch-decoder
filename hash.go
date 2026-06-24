package fch

import "crypto/sha512"

const payloadHashSize = sha512.Size

func payloadHash(payload []byte) []byte {
	sum := sha512.Sum512(payload)
	return sum[:]
}
