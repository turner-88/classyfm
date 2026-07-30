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

// impersonateMarker is the first payload field of an impersonation token, taking
// the slot the root token uses for its email. It can never collide with
// VirtualRootEmail, so one signed envelope carries both token kinds.
const impersonateMarker = "impersonate"

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
	payload, ok := parseSignedPayload(token, secret)
	if !ok {
		return nil
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 3 || fields[0] != VirtualRootEmail || fields[1] != virtualRootRole {
		return nil
	}
	if !unexpired(fields[2]) {
		return nil
	}
	return &AuthUser{ID: 0, Email: VirtualRootEmail, Name: virtualRootName, Role: virtualRootRole, Virtual: true}
}

// NewImpersonationToken mints a session cookie value that acts as the users row
// with the given id. Like the root token it is self-contained and signed with the
// same secret (so rotating SESSION_SECRET invalidates both), but it carries only
// the user id: name, role and is_active are re-read from the database on every
// request, which is what lets deactivating a user end an impersonation session.
func NewImpersonationToken(secret string, userID uint64, ttl time.Duration) string {
	expiry := time.Now().Add(ttl).Unix()
	payload := impersonateMarker + "|" + strconv.FormatUint(userID, 10) + "|" + strconv.FormatInt(expiry, 10)
	return signVirtualPayload(payload, secret)
}

// verifyImpersonationToken checks the signature and expiry of a value produced by
// NewImpersonationToken and returns the impersonated user id. Like
// verifyVirtualSessionToken it reports failure (not an error) for anything
// unrecognizable - including plain DB session tokens - so callers fall through.
func verifyImpersonationToken(token, secret string) (uint64, bool) {
	payload, ok := parseSignedPayload(token, secret)
	if !ok {
		return 0, false
	}
	fields := strings.Split(payload, "|")
	if len(fields) != 3 || fields[0] != impersonateMarker {
		return 0, false
	}
	userID, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil || userID == 0 {
		return 0, false
	}
	if !unexpired(fields[2]) {
		return 0, false
	}
	return userID, true
}

// parseSignedPayload unwraps the "base64url(payload).hmac" envelope shared by the
// root and impersonation tokens, returning the payload only if the signature
// verifies.
func parseSignedPayload(token, secret string) (string, bool) {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return "", false
	}
	payloadB64, sigHex := parts[0], parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", false
	}
	payload := string(payloadBytes)
	if !hmac.Equal([]byte(sigHex), []byte(hmacHex(payload, secret))) {
		return "", false
	}
	return payload, true
}

// unexpired reports whether a unix-timestamp payload field is parseable and still
// in the future.
func unexpired(field string) bool {
	expiry, err := strconv.ParseInt(field, 10, 64)
	return err == nil && time.Now().Unix() <= expiry
}

func signVirtualPayload(payload, secret string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + hmacHex(payload, secret)
}

func hmacHex(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
