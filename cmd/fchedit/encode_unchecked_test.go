package main

import (
	"crypto/sha512"

	"github.com/lanchelms/fch-decoder/binary"
	"github.com/lanchelms/fch-decoder/valheim"
)

func encodeUnchecked(character *valheim.Character) []byte {
	payload := binary.NewWriter()
	character.Encode(payload)
	payloadBytes := payload.Data()
	hash := sha512.Sum512(payloadBytes)

	w := binary.NewWriter()
	w.Uint32(uint32(len(payloadBytes)))
	w.Bytes(payloadBytes)
	w.Uint32(uint32(len(hash)))
	w.Bytes(hash[:])
	return w.Data()
}
