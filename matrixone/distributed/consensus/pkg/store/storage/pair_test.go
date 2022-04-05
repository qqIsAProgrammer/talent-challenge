package storage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeAndDecode(t *testing.T) {
	pairs := []Pair{
		{
			Key: []byte("1"),
			Val: []byte("i"),
		},
		{
			Key: []byte("2"),
			Val: []byte("i"),
		},
		{
			Key: []byte("3"),
			Val: []byte("i"),
		},
		{
			Key: []byte("4"),
			Val: []byte("i"),
		},
		{
			Key: []byte("5"),
			Val: []byte("i"),
		},
	}
	enc := Encode(pairs)
	dec := Decode(enc)
	for i := range dec {
		assert.Equal(t, pairs[i].Key, dec[i].Key)
		assert.Equal(t, pairs[i].Val, dec[i].Val)
	}
}
