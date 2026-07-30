package middleware

import (
	"context"
	"crypto/hmac"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// FlashCookieName holds a one-shot success message across the redirect that follows
// a successful admin save.
const FlashCookieName = "classyfm_flash"

type flashCtxKey int

const flashMsgCtxKey flashCtxKey = iota

// flashTTL is how long a flash stays valid. It only has to survive one redirect;
// anything longer would resurface on a much later page load.
const flashTTL = 60 * time.Second

// SetFlash stores msg for exactly the next admin page render. The value is signed
// with the same scheme as the virtual session token (see virtual.go) because an
// unsigned cookie lets any other site plant text that our panel would then display
// as if it came from us.
func SetFlash(w http.ResponseWriter, secret, msg string, secure bool) {
	payload := strconv.FormatInt(time.Now().Add(flashTTL).Unix(), 10) + "|" + msg
	http.SetCookie(w, &http.Cookie{
		Name:     FlashCookieName,
		Value:    signVirtualPayload(payload, secret),
		Path:     "/admin",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(flashTTL / time.Second),
	})
}

// Flash reads and immediately clears the flash cookie, attaching the message to the
// request context. Clearing here (rather than in the handler) is what makes it
// one-shot: the expiring Set-Cookie goes out on the same response that renders it,
// so a reload shows a clean page.
func Flash(secret string, secure bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(FlashCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name:     FlashCookieName,
				Value:    "",
				Path:     "/admin",
				HttpOnly: true,
				Secure:   secure,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   -1,
			})
			msg := verifyFlash(cookie.Value, secret)
			if msg == "" {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), flashMsgCtxKey, msg)))
		})
	}
}

// FlashMessage returns the pending flash message for this request, or "".
func FlashMessage(r *http.Request) string {
	msg, _ := r.Context().Value(flashMsgCtxKey).(string)
	return msg
}

// verifyFlash checks the signature and expiry, returning "" for anything it doesn't
// recognise so a stale or forged cookie simply renders no banner.
func verifyFlash(value, secret string) string {
	payloadB64, sigHex, ok := strings.Cut(value, ".")
	if !ok {
		return ""
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return ""
	}
	payload := string(payloadBytes)
	if !hmac.Equal([]byte(sigHex), []byte(hmacHex(payload, secret))) {
		return ""
	}
	expiry, msg, ok := strings.Cut(payload, "|")
	if !ok {
		return ""
	}
	ts, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil || time.Now().Unix() > ts {
		return ""
	}
	return msg
}
