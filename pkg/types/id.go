package types

import (
	"encoding/binary"
	"math/bits"

	"github.com/google/uuid"
)

const (
	idAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	idBase     = 57
	idEncLen   = 22
	idNDigits  = 10
	idDivisor  = 362033331456891249 // 57^10
)

// Id returns a new unique identifier as a 22-character base57-encoded UUIDv4.
func Id() string {
	return encodeShortUUID(uuid.New())
}

type uint128 struct {
	Lo, Hi uint64
}

func (u uint128) quoRem64(v uint64) (q uint128, r uint64) {
	q.Hi, r = bits.Div64(0, u.Hi, v)
	q.Lo, r = bits.Div64(r, u.Lo, v)
	return
}

func encodeShortUUID(u uuid.UUID) string {
	num := uint128{
		binary.BigEndian.Uint64(u[8:]),
		binary.BigEndian.Uint64(u[:8]),
	}

	var r uint64
	var i int
	buf := make([]byte, idEncLen)
	for i = idEncLen - 1; num.Hi > 0 || num.Lo > 0; {
		num, r = num.quoRem64(idDivisor)
		for j := 0; j < idNDigits && i >= 0; j++ {
			buf[i] = idAlphabet[r%idBase]
			r /= idBase
			i--
		}
	}
	for ; i >= 0; i-- {
		buf[i] = idAlphabet[0]
	}

	return string(buf)
}
