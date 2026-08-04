// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all runtime configuration for the ClassyFM server.
type Config struct {
	Env  string // "development" | "production"
	Host string
	Port string

	// Database (MySQL DSN, e.g. user:pass@tcp(127.0.0.1:3306)/classyfm?parseTime=true)
	DatabaseDSN string

	// Sessions
	SessionSecret string

	// External / radio
	StreamURL        string
	ShoutcastBaseURL string
	StationName      string
	// StationSlogan is the station's tagline, shown as the floating player's
	// last-resort subtitle when there is no song and no on-air program.
	StationSlogan string

	// SiteURL is the canonical public origin (no trailing slash), used to build
	// absolute URLs for Open Graph tags, canonical links, and sitemap.xml.
	SiteURL string

	// GAMeasurementID is the Google Analytics 4 property ID ("G-XXXXXXXXXX").
	// Blank disables analytics entirely — the public layout emits no tag at all,
	// so local dev never reports into the property. Admin pages are never tracked.
	GAMeasurementID string

	// UploadDir is the on-disk directory user-uploaded files (e.g. program
	// banner images) are written to, served at /uploads/*.
	UploadDir string

	// Feed aggregation
	YouTubeChannelID string
	FeedInterval     time.Duration

	// Listener sampling (internal/listeners). ListenerInterval is how often the
	// Shoutcast audience is read; ListenerRetention is how long the raw sample
	// trail is kept before pruning (the per-day rollup is kept forever).
	ListenerInterval  time.Duration
	ListenerRetention time.Duration

	ShutdownTimeout time.Duration

	// SMTP (password reset emails)
	SMTPHost              string
	SMTPPort              string
	SMTPUser              string
	SMTPPass              string
	SMTPFrom              string
	PasswordResetTokenTTL time.Duration
}

// Load reads configuration from environment variables, applying sane defaults for
// local development. It never fails; missing values fall back to defaults so the
// server can boot for early phases before a database is configured.
func Load() *Config {
	streamURL := getenv("STREAM_URL", "https://c4.siar.us:10340/stream.mp3")
	c := &Config{
		Env:              getenv("APP_ENV", "development"),
		Host:             getenv("HOST", "0.0.0.0"),
		Port:             getenv("PORT", "8080"),
		DatabaseDSN:      getenv("DATABASE_DSN", ""),
		SessionSecret:    getenv("SESSION_SECRET", "dev-insecure-secret-change-me"),
		StreamURL:        streamURL,
		ShoutcastBaseURL: getenv("SHOUTCAST_BASE_URL", deriveShoutcastBase(streamURL)),
		StationName:      getenv("STATION_NAME", "Classy 103.4 FM"),
		StationSlogan:    getenv("STATION_SLOGAN", "The Actual Radio - More Than Just Talk"),
		SiteURL:          strings.TrimRight(getenv("SITE_URL", "https://classyfm.remorac.com"), "/"),
		GAMeasurementID:  getenv("GA_MEASUREMENT_ID", ""),
		UploadDir:        getenv("UPLOAD_DIR", "web/uploads"),
		YouTubeChannelID: getenv("YOUTUBE_CHANNEL_ID", ""),
		FeedInterval:     getdur("FEED_INTERVAL", 30*time.Minute),
		// 5 minutes is 288 readings a day: fine enough that a daily peak is a real
		// peak, light enough to be nothing next to the audio the same box is
		// already serving.
		ListenerInterval:  getdur("LISTENER_INTERVAL", 5*time.Minute),
		ListenerRetention: getdur("LISTENER_RETENTION", 30*24*time.Hour),
		ShutdownTimeout:   getdur("SHUTDOWN_TIMEOUT", 10*time.Second),

		SMTPHost:              getenv("SMTP_HOST", ""),
		SMTPPort:              getenv("SMTP_PORT", "587"),
		SMTPUser:              getenv("SMTP_USER", ""),
		SMTPPass:              getenv("SMTP_PASS", ""),
		SMTPFrom:              getenv("SMTP_FROM", ""),
		PasswordResetTokenTTL: getdur("PASSWORD_RESET_TOKEN_TTL", time.Hour),
	}
	return c
}

// deriveShoutcastBase derives the Shoutcast server's base URL (scheme://host:port)
// from the audio stream URL, e.g. "https://c4.siar.us:10340/stream.mp3" ->
// "https://c4.siar.us:10340". Falls back to the raw stream URL if it doesn't parse,
// so the server still boots (now-playing fetches will simply fail until fixed).
func deriveShoutcastBase(streamURL string) string {
	u, err := url.Parse(streamURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return streamURL
	}
	return u.Scheme + "://" + u.Host
}

// Addr returns the host:port the HTTP server should listen on.
func (c *Config) Addr() string { return fmt.Sprintf("%s:%s", c.Host, c.Port) }

// IsProd reports whether the app runs in production mode.
func (c *Config) IsProd() bool { return c.Env == "production" }

func getenv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getdur(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		slog.Warn("invalid duration env var, using default", "key", key, "value", v)
	}
	return def
}
