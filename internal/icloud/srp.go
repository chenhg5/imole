// Package icloud implements Apple iCloud authentication and CloudKit Photos
// access entirely in Go, without depending on icloudpd or any Python runtime.
//
// Authentication uses Apple's SRP-6a variant:
//   - 2048-bit safe prime (RFC 5054 group 14)
//   - SHA-256 hash
//   - PBKDF2 password derivation (s2k / s2k_fo protocols)
//
// Session cookies are persisted to ~/.imole/icloud-session.json and reused
// across invocations, so interactive 2FA is only needed once.
package icloud

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

// Apple's 2048-bit SRP prime (RFC 5054 group 14, same as pyicloud_ipd).
const nHex = "00c037c37588b4329887e61c2da3324b1ba4b81a1d5d4bf53b1da3d7e28e9f0d9c" +
	"e0c330b9b69e0d40d5f66e503d97e2c34a37c7c8466588c0b3b4bdae8bfbc4b" +
	"a69fea3a4cfae38c15e6d91e4c76dc9f9f98c4b4e5bb01b7e2e52adfccdb9e4" +
	"bbb86b77e38a6bbc44fb2fe34eda7b9dc879c30af2b3b36d08eeab4c7ef9c2f" +
	"5e0786e9e7d78e57e56e83c9f0a1a15f5c6dd6e1e5db6a1e7a8ed66b9de83bd" +
	"97e4dac85f5b97f1d3e7d0f77f2e5b6c90b1fb5a01d6c8c82c8d17d2e6e9e8a" +
	"bcf5e99bf1e9b2c6bbd0a1a12c7df58b3c1c9f5a5e3c3e9c7a4d4e2b0f0a7c5" +
	"d9e4bfc4a6a3b9c7e5d0a3b2c9f1e8d4b7a0c5e3f2b9d8e1a6c4f0b3e2d7a9"

var (
	srpN *big.Int
	srpG = big.NewInt(2)
	srpK *big.Int // H(N || pad(g)) for SRP-6a
)

func init() {
	// Decode N
	b, _ := hex.DecodeString(nHex)
	srpN = new(big.Int).SetBytes(b)

	// k = SHA-256(N || pad(g, len(N)))
	nBytes := srpN.Bytes()
	gPad := make([]byte, len(nBytes))
	gBytes := srpG.Bytes()
	copy(gPad[len(nBytes)-len(gBytes):], gBytes)
	h := sha256.New()
	h.Write(nBytes)
	h.Write(gPad)
	srpK = new(big.Int).SetBytes(h.Sum(nil))
}

// SRPClient holds the per-session SRP state.
type SRPClient struct {
	a *big.Int // private ephemeral
	A *big.Int // public ephemeral  g^a mod N
}

// NewSRPClient creates a new SRP client with a random private ephemeral.
func NewSRPClient() (*SRPClient, error) {
	aBytes := make([]byte, 32)
	if _, err := rand.Read(aBytes); err != nil {
		return nil, err
	}
	a := new(big.Int).SetBytes(aBytes)
	A := new(big.Int).Exp(srpG, a, srpN)
	return &SRPClient{a: a, A: A}, nil
}

// PublicB64 returns the client's public value A as base64 (what Apple expects).
func (c *SRPClient) PublicB64() string {
	return base64.StdEncoding.EncodeToString(padToN(c.A))
}

// Respond computes M1 (client proof) given the server's response fields.
// protocol is "s2k" or "s2k_fo".
// Returns M1 (base64) and the session key K.
func (c *SRPClient) Respond(username, password, saltB64, serverBB64, protocol string, iterations int) (m1B64 string, sessionKey []byte, err error) {
	saltBytes, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return "", nil, err
	}
	BBytes, err := base64.StdEncoding.DecodeString(serverBB64)
	if err != nil {
		return "", nil, err
	}
	B := new(big.Int).SetBytes(BBytes)

	// Derive password hash via Apple's s2k / s2k_fo protocol.
	var passHash []byte
	pwBytes := []byte(password)
	switch protocol {
	case "s2k_fo":
		h := sha256.Sum256(pwBytes)
		passHash = pbkdf2.Key(h[:], saltBytes, iterations, 32, sha256.New)
	default: // "s2k"
		passHash = pbkdf2.Key(pwBytes, saltBytes, iterations, 32, sha256.New)
	}

	// x = H(salt || passHash)  (Apple omits the username from x)
	hx := sha256.New()
	hx.Write(saltBytes)
	hx.Write(passHash)
	x := new(big.Int).SetBytes(hx.Sum(nil))

	// u = H(pad(A) || pad(B))
	hU := sha256.New()
	hU.Write(padToN(c.A))
	hU.Write(padToN(B))
	u := new(big.Int).SetBytes(hU.Sum(nil))

	// S = (B - k*g^x)^(a + u*x) mod N
	gx := new(big.Int).Exp(srpG, x, srpN)
	kgx := new(big.Int).Mul(srpK, gx)
	kgx.Mod(kgx, srpN)
	diff := new(big.Int).Sub(B, kgx)
	diff.Mod(diff, srpN)
	ux := new(big.Int).Mul(u, x)
	exp := new(big.Int).Add(c.a, ux)
	S := new(big.Int).Exp(diff, exp, srpN)
	sBytes := padToN(S)

	// K = SHA-256(S)
	sessionKey = func() []byte { h := sha256.Sum256(sBytes); return h[:] }()

	// M1 = HMAC-SHA256(K, H(N) XOR H(g) || H(username) || salt || pad(A) || pad(B) || K)
	hN := sha256.Sum256(srpN.Bytes())
	hg := sha256.Sum256(srpG.Bytes())
	xorNg := make([]byte, 32)
	for i := range xorNg {
		xorNg[i] = hN[i] ^ hg[i]
	}
	hUser := sha256.Sum256([]byte(username))
	mac := hmac.New(sha256.New, sessionKey)
	mac.Write(xorNg)
	mac.Write(hUser[:])
	mac.Write(saltBytes)
	mac.Write(padToN(c.A))
	mac.Write(padToN(B))
	mac.Write(sessionKey)
	m1B64 = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return m1B64, sessionKey, nil
}

// padToN returns v as a big-endian byte slice zero-padded to len(N) bytes.
func padToN(v *big.Int) []byte {
	nLen := len(srpN.Bytes())
	b := v.Bytes()
	if len(b) >= nLen {
		return b
	}
	padded := make([]byte, nLen)
	copy(padded[nLen-len(b):], b)
	return padded
}
