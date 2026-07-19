package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// VirtualRootEmail is the reserved email for the break-glass superadmin login. No
// real users row may ever use this email (enforced in the admin user-management
// handlers) so the identity can never collide with the virtual session below.
const VirtualRootEmail = "root@local.system"

const virtualRootRole = "superadmin"
const virtualRootName = "Root (virtual)"

// NewVirtualSessionToken mints a self-contained, HMAC-signed session cookie value
// for the break-glass root login. Unlike normal sessions (an opaque random token
// looked up in the sessions table, which has a foreign key to users), this token
// carries its own payload and expiry and is verified without touching the
// database - required because the virtual user has no users row to key a DB
// session off of.
func NewVirtualSessionToken(secret string, ttl time.Duration) string {
	expiry := time.Now().Add(ttl).Unix()
	payload := VirtualRootEmail + "|" + virtualRootRole + "|" + strconv.FormatInt(expiry, 10)
	return signVirtualPayload(payload, secret)
}

// verifyVirtualSessionToken checks the signature and expiry of a cookie value
// produced by NewVirtualSessionToken. It returns nil (not an error) whenever the
// value isn't a recognizable virtual token - including plain DB session tokens -
// so callers can fall through to the normal DB-backed session lookup.
func verifyVirtualSessionToken(token, secret string) *AuthUser {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil
	}
	payloadB64, sigHex := parts[0], parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil
	}
	payload := string(payloadBytes)
	wantSig := hmacHex(payload, secret)
	if !hmac.Equal([]byte(sigHex), []byte(wantSig)) {
		return nil
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 3 || fields[0] != VirtualRootEmail || fields[1] != virtualRootRole {
		return nil
	}
	expiry, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || time.Now().Unix() > expiry {
		return nil
	}
	return &AuthUser{ID: 0, Email: VirtualRootEmail, Name: virtualRootName, Role: virtualRootRole, Virtual: true}
}

func signVirtualPayload(payload, secret string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + hmacHex(payload, secret)
}

func hmacHex(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
