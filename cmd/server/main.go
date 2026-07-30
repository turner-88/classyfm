// Command server is the ClassyFM website entrypoint: it wires configuration, the
// database pool, the template renderer, the radio now-playing service, HTTP routes,
// and runs an HTTP server with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/classyfm/classyfm/internal/config"
	"github.com/classyfm/classyfm/internal/db"
	"github.com/classyfm/classyfm/internal/db/sqlc"
	"github.com/classyfm/classyfm/internal/feeds"
	adminh "github.com/classyfm/classyfm/internal/handlers/admin"
	pubh "github.com/classyfm/classyfm/internal/handlers/public"
	"github.com/classyfm/classyfm/internal/mail"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/render"
	"github.com/classyfm/classyfm/web"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	logger := newLogger(cfg)
	slog.SetDefault(logger)
	slog.Info("starting classyfm", "env", cfg.Env, "addr", cfg.Addr())

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Database (optional in early phases).
	pool, err := db.Open(rootCtx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	var queries *sqlc.Queries
	if pool != nil {
		defer pool.Close()
		queries = sqlc.New(pool)
		slog.Info("connected to mysql")
	} else {
		slog.Warn("no DATABASE_DSN set; running without database (admin panel + DB-backed pages disabled)")
	}

	// Template renderer (reload templates per-request in development).
	renderer, err := render.New(web.Templates(), !cfg.IsProd())
	if err != nil {
		return err
	}

	radioSvc := radio.NewService(cfg.StreamURL, cfg.ShoutcastBaseURL)
	publicH := pubh.New(renderer, radioSvc, queries, cfg.StationName, cfg.StationSlogan, cfg.SiteURL)

	var worker *feeds.Worker
	if queries != nil {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		worker = feeds.NewWorker(queries, cfg.FeedInterval,
			feeds.NewYouTubeSource(httpClient, cfg.YouTubeChannelID),
			feeds.NewWordPressSource(httpClient, "klikpositif", "https://klikpositif.com/feed/"),
			feeds.NewWordPressSource(httpClient, "katasumbar", "https://katasumbar.com/feed/"),
		)
		go worker.Run(rootCtx)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		slog.Warn("could not create upload dir", "err", err, "dir", cfg.UploadDir)
	}
	mailer := &mail.Mailer{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPass, From: cfg.SMTPFrom}
	adminH := adminh.New(renderer, queries, worker, radioSvc, cfg.StationName, cfg.IsProd(), cfg.UploadDir, mailer, cfg.SiteURL, cfg.PasswordResetTokenTTL, cfg.SessionSecret)

	router := newRouter(cfg, publicH, adminH, queries)

	srv := &http.Server{
		Addr:              cfg.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Run server and wait for shutdown signal.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-rootCtx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
		return err
	}
	slog.Info("server stopped cleanly")
	return nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if cfg.IsProd() {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}

func newRouter(cfg *config.Config, ph *pubh.Handler, ah *adminh.Handler, queries *sqlc.Queries) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(appmw.Recover(ph.ServerError))
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(appmw.SecurityHeaders(cfg.IsProd()))

	// Health check.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static assets. Cache-Control is set explicitly because assets are embedded via
	// go:embed, whose ModTime is always zero, so http.FileServer can't drive
	// conditional GETs off Last-Modified the way it would for real files on disk.
	staticFS := http.FileServer(http.FS(web.Static()))
	r.Handle("/static/*", http.StripPrefix("/static/", cacheControl("public, max-age=3600", staticFS)))

	// Browsers and crawlers probe the bare /favicon.ico regardless of the <link> tags,
	// so point it at the embedded icon rather than letting it 404. Registered for HEAD
	// as well as GET because crawlers probe with HEAD, and chi answers 405 (not 200)
	// for a method that has no route.
	faviconRedirect := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/img/favicon.ico", http.StatusMovedPermanently)
	}
	r.Get("/favicon.ico", faviconRedirect)
	r.Head("/favicon.ico", faviconRedirect)

	// User-uploaded files (e.g. program banner images). Unlike /static/*, this is a
	// real on-disk directory (not embed.FS), so filenames are random-per-upload and
	// never mutated in place - a long immutable cache lifetime is safe.
	uploadsFS := http.FileServer(http.Dir(cfg.UploadDir))
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", cacheControl("public, max-age=31536000, immutable", uploadsFS)))

	// Radio now-playing JSON (polled by the floating player).
	r.Get("/api/nowplaying", ph.NowPlayingJSON)

	// Today's schedule on-air/progress JSON (polled by schedule.js).
	r.Get("/api/schedule/today", ph.ScheduleTodayJSON)

	// Currently on-air program JSON (polled by now-playing-card.js on Home).
	r.Get("/api/schedule/current", ph.CurrentScheduleJSON)

	// SEO.
	r.Get("/robots.txt", ph.Robots)
	r.Get("/sitemap.xml", ph.Sitemap)

	// Public pages.
	r.Get("/", ph.Home)
	r.Get("/about", ph.About)
	r.Get("/program", ph.Program)
	r.Get("/program/{slug}", ph.ProgramDetail)
	r.Get("/live", ph.Live)
	r.Get("/news", ph.News)
	r.Get("/news/{slug}", ph.NewsDetail)
	r.Get("/broadcasters", ph.Broadcasters)
	r.Get("/broadcasters/{slug}", ph.BroadcasterDetail)

	// Admin panel: session auth + CSRF on every route; RequireAuth on everything
	// except the login/logout endpoints.
	r.Route("/admin", func(ar chi.Router) {
		ar.Use(appmw.Auth(queries, cfg.SessionSecret))
		ar.Use(appmw.CSRF(cfg.IsProd()))

		ar.Get("/login", ah.LoginPage)
		ar.With(appmw.RateLimit(10, time.Minute)).Post("/login", ah.Login)
		ar.Post("/logout", ah.Logout)

		ar.Get("/forgot-password", ah.ForgotPasswordPage)
		ar.With(appmw.RateLimit(5, time.Minute)).Post("/forgot-password", ah.ForgotPasswordSubmit)
		ar.Get("/reset-password", ah.ResetPasswordPage)
		ar.With(appmw.RateLimit(5, time.Minute)).Post("/reset-password", ah.ResetPasswordSubmit)

		ar.Group(func(pr chi.Router) {
			pr.Use(appmw.RequireAuth)
			pr.Get("/", ah.Dashboard)
			pr.Get("/programs", ah.ProgramsList)
			pr.Get("/programs/new", ah.ProgramNew)
			pr.Post("/programs", ah.ProgramCreate)
			pr.Get("/programs/{id}", ah.ProgramDetail)
			pr.Get("/programs/{id}/edit", ah.ProgramEdit)
			pr.Post("/programs/{id}", ah.ProgramUpdate)
			pr.Post("/programs/{id}/delete", ah.ProgramDelete)
			pr.Post("/programs/{id}/schedules", ah.ScheduleCreate)
			pr.Post("/programs/{id}/schedules/{scheduleID}", ah.ScheduleUpdate)
			pr.Post("/programs/{id}/schedules/{scheduleID}/delete", ah.ScheduleDelete)

			pr.Get("/broadcasters", ah.BroadcastersList)
			pr.Get("/broadcasters/new", ah.BroadcasterNew)
			pr.Post("/broadcasters", ah.BroadcasterCreate)
			pr.Get("/broadcasters/{id}", ah.BroadcasterDetail)
			pr.Get("/broadcasters/{id}/edit", ah.BroadcasterEdit)
			pr.Post("/broadcasters/{id}", ah.BroadcasterUpdate)
			pr.Post("/broadcasters/{id}/delete", ah.BroadcasterDelete)

			pr.Get("/hot-release", ah.HotReleaseList)
			pr.Get("/hot-release/new", ah.HotReleaseNew)
			pr.Post("/hot-release", ah.HotReleaseCreate)
			pr.Get("/hot-release/{id}", ah.HotReleaseDetail)
			pr.Get("/hot-release/{id}/edit", ah.HotReleaseEdit)
			pr.Post("/hot-release/{id}", ah.HotReleaseUpdate)
			pr.Post("/hot-release/{id}/feature", ah.HotReleaseToggleFeature)
			pr.Post("/hot-release/{id}/delete", ah.HotReleaseDelete)

			pr.Get("/newsfeed", ah.NewsfeedList)
			pr.Post("/newsfeed/{id}/publish", ah.NewsfeedTogglePublish)
			pr.Post("/newsfeed/{id}/feature", ah.NewsfeedToggleFeature)

			pr.Get("/media", ah.MediaLinksList)
			pr.Post("/media", ah.MediaLinksUpdate)

			pr.Get("/about", ah.AboutPage)
			pr.Post("/about/banner", ah.AboutBannerUpdate)
			pr.Post("/about/segments/{segment}", ah.AboutSegmentUpdate)

			pr.Get("/ads", ah.AdsList)
			pr.Get("/ads/new", ah.AdBannerNew)
			pr.Post("/ads", ah.AdBannerCreate)
			pr.Post("/ads/slots/{slot}", ah.AdSlotUpdate)
			pr.Get("/ads/{id}/edit", ah.AdBannerEdit)
			pr.Post("/ads/{id}", ah.AdBannerUpdate)
			pr.Post("/ads/{id}/delete", ah.AdBannerDelete)

			pr.Get("/feed-sources", ah.FeedSourcesList)
			pr.Post("/feed-sources/refresh", ah.FeedSourcesRefresh)
			pr.Post("/feed-sources/{source}", ah.FeedSourceUpdate)

			pr.Get("/profile", ah.ProfilePage)
			pr.Post("/profile", ah.ProfileUpdate)
			pr.Post("/profile/password", ah.ProfilePassword)

			pr.Group(func(sr chi.Router) {
				sr.Use(appmw.RequireRole(string(sqlc.UsersRoleSuperadmin)))
				sr.Get("/users", ah.UsersList)
				sr.Get("/users/new", ah.UserNew)
				sr.Post("/users", ah.UserCreate)
				sr.Get("/users/{id}/edit", ah.UserEdit)
				sr.Post("/users/{id}", ah.UserUpdate)
				sr.Post("/users/{id}/delete", ah.UserDelete)

				sr.Get("/audit-trail", ah.AuditTrailList)
			})
		})
	})

	r.NotFound(ph.NotFound)

	return r
}

// cacheControl sets a Cache-Control header on every response from next.
func cacheControl(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		next.ServeHTTP(w, r)
	})
}
