// SSHA password
package main

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
)

func generateSSHA(password string) string {
	salt := make([]byte, 8)
	rand.Read(salt)

	hash := createSSHAHash(password, salt)
	return fmt.Sprintf("{SSHA}%s", base64.StdEncoding.EncodeToString(hash))
}

func validateSSHA(password string, hash string) bool {
	if len(hash) < 7 || hash[:6] != "{SSHA}" {
		return false
	}

	data, err := base64.StdEncoding.DecodeString(hash[6:])
	if len(data) < 21 || err != nil {
		return false
	}

	newHash := createSSHAHash(password, data[20:])

	if subtle.ConstantTimeCompare(newHash, data) == 1 {
		return true
	}

	return false
}

func createSSHAHash(password string, salt []byte) []byte {
	pass := []byte(password)
	str := append(pass, salt...)
	sum := sha1.Sum(str)
	result := append(sum[:], salt...)
	return result
}
