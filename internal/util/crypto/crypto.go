package crypto

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/base32"
	"fmt"
)

// GenerateHashedPassword simple obfuscation to avoid storing plain text password by using a salt
func GenerateHashedPassword(password string, salt string) string {
	h := sha512.New()
	passwordPlusSalt := fmt.Sprintf("%s%s", password, salt)
	h.Write([]byte(passwordPlusSalt))
	hashedPasswordPlusSalt := h.Sum(nil)
	return fmt.Sprintf("%x", hashedPasswordPlusSalt)
}

func GenerateSalt() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes)[:32]
}
