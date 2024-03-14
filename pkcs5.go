package ai

import (
	"bytes"
)

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(encrypt []byte) []byte {
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}
