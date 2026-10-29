package bench

import "crypto/rand"

func NewUUIDs(batch [][16]byte) {
	for i := range batch {
		_, _ = rand.Read(batch[i][:])
		batch[i][6] = batch[i][6]&0x0f | 0x40
		batch[i][8] = batch[i][8]&0x3f | 0x80
	}
}
