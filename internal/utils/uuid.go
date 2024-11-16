package utils

import (
	"crypto/rand"
	"fmt"
	"io"
)

func CreateSessionToken() (string, error) {

	uuid := make([]byte, 16)

	_, err := io.ReadFull(rand.Reader, uuid)
	if err != nil {
		return "", err
	}

	uuid[8] = (uuid[8] & 0xBF) | 0x80
	uuid[6] = (uuid[6] & 0x4F) | 0x40

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
