package bench

import (
	"encoding/base64"
	"math/bits"
	"sync/atomic"
)

const (
	IndexChars    = 5
	bitsPerChar   = 6
	encodedLength = 22
)

var encoding = base64.RawURLEncoding

var charValues = func() [256]uint8 {
	var values [256]uint8
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for i := range len(alphabet) {
		values[alphabet[i]] = uint8(i)
	}
	return values
}()

type Bucket struct {
	words []atomic.Uint64
	chars int
}

func NewBucket(chars int) *Bucket {
	size := uint64(1) << (bitsPerChar * chars)
	return &Bucket{words: make([]atomic.Uint64, max(1, size/64)), chars: chars}
}

func (b *Bucket) Indexes() uint64 {
	return uint64(1) << (bitsPerChar * b.chars)
}

func (b *Bucket) Bytes() uint64 {
	return uint64(len(b.words)) * 8
}

func (b *Bucket) Mark(key []byte) bool {
	i := Index(key, b.chars)
	mask := uint64(1) << (i % 64)
	return b.words[i/64].Or(mask)&mask != 0
}

func (b *Bucket) Set() uint64 {
	var total uint64
	for i := range b.words {
		total += uint64(bits.OnesCount64(b.words[i].Load()))
	}
	return total
}

func Index(key []byte, chars int) uint64 {
	var index uint64
	for _, c := range key[:chars] {
		index = index<<bitsPerChar | uint64(charValues[c])
	}
	return index
}

func Key(dst *[encodedLength]byte, uuid *[16]byte) []byte {
	encoding.Encode(dst[:], uuid[:])
	return dst[:]
}
