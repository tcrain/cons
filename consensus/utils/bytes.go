package utils

import "fmt"

func XORbytes(a, b []byte) ([]byte, error) {
	if len(a) != len(b) {
		return nil, fmt.Errorf("slices must be same size")
	}
	ret := make([]byte, len(a))
	for i, nxt := range a {
		ret[i] = nxt ^ b[i]
	}
	return ret, nil
}
