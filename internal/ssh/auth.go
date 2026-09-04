// Package ssh provides SSH server setup and public-key authentication.
package ssh

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	gossh "golang.org/x/crypto/ssh"
)

// Fingerprint computes the SHA256 fingerprint of an SSH public key,
// matching the output of `ssh-keygen -lf key.pub` (SHA256:...).
func Fingerprint(key gossh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return fmt.Sprintf("SHA256:%s", base64.RawStdEncoding.EncodeToString(hash[:]))
}
