package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover replaces a panicking handler with a friendly error response instead of
// crashing the connection or leaking a stack trace to the client. The panic itself
// is logged server-side only; onErr renders the user-facing error page.
func Recover(onErr func(http.ResponseWriter, *http.Request)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if rec == http.ErrAbortHandler {
						// Handler wants to abort silently (e.g. hijacked conns); don't
						// mask that signal.
						panic(rec)
					}
					slog.Error("panic recovered", "err", rec, "stack", string(debug.Stack()), "path", r.URL.Path)
					onErr(w, r)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
