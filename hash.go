package ai

import (
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// MD5 计算md5值
func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}

// SHA1 计算sha1值
func SHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}

// SHA256 计算sha256值
func SHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))

	return hex.EncodeToString(h.Sum(nil))
}

// Hash 计算指定算法的hash值
func Hash(hash crypto.Hash, str string) (string, error) {
	if !hash.Available() {
		return "", fmt.Errorf("crypto: requested hash function (%s) is unavailable", hash.String())
	}

	h := hash.New()
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil)), nil
}

// HMacSHA256 计算hmac-sha256值
func HMacSHA256(key, str string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil))
}

// HMac 计算指定hash算法的hmac值
func HMac(hash crypto.Hash, key, str string) (string, error) {
	if !hash.Available() {
		return "", fmt.Errorf("crypto: requested hash function (%s) is unavailable", hash.String())
	}

	h := hmac.New(hash.New, []byte(key))
	h.Write([]byte(str))

	return hex.EncodeToString(h.Sum(nil)), nil
}
