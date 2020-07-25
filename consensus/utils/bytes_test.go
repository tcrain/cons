package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestXORbytes(t *testing.T) {
	a, err := XORbytes([]byte{1, 0, 1, 1}, []byte{0, 1, 1, 2})
	assert.Nil(t, err)
	assert.Equal(t, a, []byte{1, 1, 0, 3})

	_, err = XORbytes([]byte{}, []byte{0})
	assert.NotNil(t, err)
}
