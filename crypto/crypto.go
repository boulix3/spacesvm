package crypto

import "crypto/ed25519"

type PublicKey struct {
	PublicKey ed25519.PublicKey `serialize:"true" json:"publicKey"`
	Addr      string            `serialize:"true" json:"addr"`
}

type PrivateKey struct {
	PrivateKey ed25519.PrivateKey `serialize:"true" json:"privateKey"`

	pk *PublicKey
}
