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
	"strconv"
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
	"github.com/classyfm/classyfm/internal/listeners"
	"github.com/classyfm/classyfm/internal/mail"
	appmw "github.com/classyfm/classyfm/internal/middleware"
	"github.com/classyfm/classyfm/internal/radio"
	"github.com/classyfm/classyfm/internal/render"
	"github.com/classyfm/classyfm/internal/tiktok"
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

	// Fail closed on insecure production configuration (e.g. a default/weak
	// SESSION_SECRET) rather than booting into an exploitable state.
	if err := cfg.Validate(); err != nil {
		return err
	}

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
	// No config of its own: the account it watches is the admin-managed TikTok
	// link (see /admin/media), handed to it per request by the public handler.
	tiktokSvc := tiktok.NewService()

	publicH := pubh.New(renderer, radioSvc, tiktokSvc, queries, cfg.StationName, cfg.StationSlogan, cfg.SiteURL, cfg.GAMeasurementID)

	var worker *feeds.Worker
	if queries != nil {
		// Feed endpoints are admin-supplied URLs fetched server-side; the safe client
		// refuses to connect to private/link-local addresses to prevent SSRF.
		httpClient := feeds.NewSafeHTTPClient(10 * time.Second)
		worker = feeds.NewWorker(queries, cfg.FeedInterval,
			feeds.NewYouTubeSource(httpClient, cfg.YouTubeChannelID),
			feeds.NewWordPressSource(httpClient, "klikpositif", "https://klikpositif.com/feed/"),
			feeds.NewWordPressSource(httpClient, "katasumbar", "https://katasumbar.com/feed/"),
		)
		go worker.Run(rootCtx)
		// Listener history has to be collected continuously: the Shoutcast server
		// reports its audience live but forgets it on restart, so nothing recovers
		// a day the sampler wasn't running.
		go listeners.NewSampler(queries, radioSvc, cfg.ListenerInterval, cfg.ListenerRetention).Run(rootCtx)
	}
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		slog.Warn("could not create upload dir", "err", err, "dir", cfg.UploadDir)
	}
	mailer := &mail.Mailer{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPass, From: cfg.SMTPFrom}
	adminH := adminh.New(renderer, queries, worker, radioSvc, cfg.StationName, cfg.IsProd(), cfg.UploadDir, mailer, cfg.SiteURL, cfg.PasswordResetTokenTTL, cfg.SessionSecret, cfg.FeedInterval)

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

	// TikTok live state JSON (polled by tiktok-live.js on every page).
	r.Get("/api/tiktok/live", ph.TikTokLiveJSON)

	// Public JSON API for the mobile app. Versioned, read-oriented, and cookie-free,
	// so it sits outside the HTML pages' group and carries its own permissive CORS.
	// (Chat is not here: both the app and the website now use Firebase directly.)
	r.Route("/api/v1", func(ar chi.Router) {
		ar.Use(appmw.CORS(cfg.APICORSOrigin))

		ar.Get("/programs", ph.APIPrograms)
		ar.Get("/programs/{slug}", ph.APIProgramDetail)
		ar.Get("/broadcasters", ph.APIBroadcasters)
		ar.Get("/broadcasters/{slug}", ph.APIBroadcasterDetail)
		ar.Get("/news", ph.APINews)
		ar.Get("/news/{slug}", ph.APINewsDetail)
		ar.Get("/podcasts", ph.APIPodcasts)
		ar.Get("/podcasts/{slug}", ph.APIPodcastDetail)
		ar.Get("/podcast-series", ph.APIPodcastSeries)
		ar.Get("/about", ph.APIAbout)
		ar.Get("/now-playing", ph.APINowPlaying)
		ar.Get("/schedule/today", ph.APIScheduleToday)
		ar.Get("/schedule/current", ph.APIScheduleCurrent)
		ar.Get("/tiktok/live", ph.APITikTokLive)
		ar.Get("/home", ph.APIHome)
		ar.Get("/config", ph.APIConfig)
		ar.Get("/ads", ph.APIAds)
	})

	// SEO.
	r.Get("/robots.txt", ph.Robots)
	r.Get("/sitemap.xml", ph.Sitemap)

	// Public pages. Kept under CSRF so any future public form has a token cookie; the
	// read-only GET pages just receive it. The live chat widget is entirely client-side
	// (Firebase), so there are no chat routes or chat identity middleware here.
	r.Group(func(pr chi.Router) {
		pr.Use(appmw.CSRF(cfg.IsProd()))

		pr.Get("/", ph.Home)
		pr.Get("/about", ph.About)
		pr.Get("/program", ph.Program)
		pr.Get("/program/{slug}", ph.ProgramDetail)
		pr.Get("/live", ph.Live)
		pr.Get("/news", ph.News)
		pr.Get("/news/{slug}", ph.NewsDetail)
		pr.Get("/podcast", ph.Podcast)
		pr.Get("/podcast/{slug}", ph.PodcastDetail)
		pr.Get("/broadcasters", ph.Broadcasters)
		pr.Get("/broadcasters/{slug}", ph.BroadcasterDetail)
		pr.Get("/privacy-policy", ph.PrivacyPolicy)
		pr.Get("/terms-and-conditions", ph.TermsAndConditions)
	})

	// Admin panel: session auth + CSRF on every route; RequireAuth on everything
	// except the login/logout endpoints.
	r.Route("/admin", func(ar chi.Router) {
		ar.Use(appmw.Auth(queries, cfg.SessionSecret))
		ar.Use(appmw.CSRF(cfg.IsProd()))
		ar.Use(appmw.Flash(cfg.SessionSecret, cfg.IsProd()))

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
			// The one admin-only polling endpoint: the listener count is deliberately
			// not on the public /api/nowplaying, so the dashboard can't read it there.
			pr.Get("/api/listeners", ah.ListenersJSON)
			pr.Get("/programs", ah.ProgramsList)
			pr.Get("/programs/new", ah.ProgramNew)
			pr.Post("/programs", ah.ProgramCreate)
			// The read-only detail pages are gone - the edit form shows everything
			// they did. Their URLs redirect rather than 404 because the POST route
			// on the same path would otherwise answer a stale bookmark with 405.
			pr.Get("/programs/{id}", redirectToEdit("/admin/programs"))
			pr.Get("/programs/{id}/edit", ah.ProgramEdit)
			pr.Post("/programs/{id}", ah.ProgramUpdate)
			pr.Post("/programs/{id}/delete", ah.ProgramDelete)

			pr.Get("/broadcasters", ah.BroadcastersList)
			pr.Get("/broadcasters/new", ah.BroadcasterNew)
			pr.Post("/broadcasters", ah.BroadcasterCreate)
			pr.Get("/broadcasters/{id}", redirectToEdit("/admin/broadcasters"))
			pr.Get("/broadcasters/{id}/edit", ah.BroadcasterEdit)
			pr.Post("/broadcasters/{id}", ah.BroadcasterUpdate)
			pr.Post("/broadcasters/{id}/delete", ah.BroadcasterDelete)

			pr.Get("/hero", ah.HeroList)
			pr.Get("/hero/new", ah.HeroSlideNew)
			pr.Post("/hero", ah.HeroSlideCreate)
			pr.Post("/hero/settings", ah.HeroSettingsUpdate)
			pr.Get("/hero/{id}", redirectToEdit("/admin/hero"))
			pr.Get("/hero/{id}/edit", ah.HeroSlideEdit)
			pr.Post("/hero/{id}", ah.HeroSlideUpdate)
			pr.Post("/hero/{id}/delete", ah.HeroSlideDelete)

			pr.Get("/hot-release", ah.HotReleaseList)
			pr.Get("/hot-release/new", ah.HotReleaseNew)
			pr.Post("/hot-release", ah.HotReleaseCreate)
			pr.Get("/hot-release/{id}", redirectToEdit("/admin/hot-release"))
			pr.Get("/hot-release/{id}/edit", ah.HotReleaseEdit)
			pr.Post("/hot-release/{id}", ah.HotReleaseUpdate)
			pr.Post("/hot-release/{id}/feature", ah.HotReleaseToggleFeature)
			pr.Post("/hot-release/{id}/delete", ah.HotReleaseDelete)

			pr.Get("/podcasts", ah.PodcastsList)
			pr.Get("/podcasts/new", ah.PodcastNew)
			pr.Post("/podcasts", ah.PodcastCreate)
			pr.Get("/podcasts/{id}", redirectToEdit("/admin/podcasts"))
			pr.Get("/podcasts/{id}/edit", ah.PodcastEdit)
			pr.Post("/podcasts/{id}", ah.PodcastUpdate)
			pr.Post("/podcasts/{id}/refresh", ah.PodcastRefresh)
			pr.Post("/podcasts/{id}/delete", ah.PodcastDelete)

			pr.Get("/podcast-series", ah.PodcastSeriesList)
			pr.Get("/podcast-series/new", ah.PodcastSeriesNew)
			pr.Post("/podcast-series", ah.PodcastSeriesCreate)
			pr.Get("/podcast-series/{id}", redirectToEdit("/admin/podcast-series"))
			pr.Get("/podcast-series/{id}/edit", ah.PodcastSeriesEdit)
			pr.Post("/podcast-series/{id}", ah.PodcastSeriesUpdate)
			pr.Post("/podcast-series/{id}/delete", ah.PodcastSeriesDelete)

			pr.Get("/newsfeed", ah.NewsfeedList)
			pr.Post("/newsfeed/{id}/publish", ah.NewsfeedTogglePublish)
			pr.Post("/newsfeed/{id}/feature", ah.NewsfeedToggleFeature)

			pr.Get("/media", ah.MediaLinksList)
			pr.Post("/media", ah.MediaLinksUpdate)

			// Connect chat moderation is view + delete only, and runs entirely
			// client-side against Firebase RTDB (see admin/chat.go), so it needs
			// just this one GET shell — no POST/delete route.
			pr.Get("/chat", ah.ChatModeration)

			pr.Get("/about", ah.AboutPage)
			pr.Post("/about", ah.AboutUpdate)

			pr.Get("/legal", ah.LegalPagesList)
			pr.Get("/legal/{slug}", ah.LegalPageEdit)
			pr.Post("/legal/{slug}", ah.LegalPageUpdate)

			pr.Get("/ads", ah.AdsList)
			pr.Get("/ads/new", ah.AdBannerNew)
			pr.Post("/ads", ah.AdBannerCreate)
			pr.Post("/ads/slots", ah.AdSlotsUpdate)
			pr.Get("/ads/{id}/edit", ah.AdBannerEdit)
			pr.Post("/ads/{id}", ah.AdBannerUpdate)
			pr.Post("/ads/{id}/delete", ah.AdBannerDelete)

			pr.Get("/feed-sources", ah.FeedSourcesList)
			pr.Post("/feed-sources", ah.FeedSourcesUpdate)
			pr.Post("/feed-sources/refresh", ah.FeedSourcesRefresh)

			pr.Get("/profile", ah.ProfilePage)
			pr.Post("/profile", ah.ProfileUpdate)
			pr.Post("/profile/password", ah.ProfilePassword)

			// Outside the superadmin group below: the user being impersonated may be
			// a plain admin, and they still need the way back to the root session.
			pr.Post("/impersonate/stop", ah.StopImpersonating)

			pr.Group(func(sr chi.Router) {
				sr.Use(appmw.RequireRole(string(sqlc.UsersRoleSuperadmin)))
				sr.Get("/users", ah.UsersList)
				sr.Get("/users/new", ah.UserNew)
				sr.Post("/users", ah.UserCreate)
				sr.Get("/users/{id}/edit", ah.UserEdit)
				sr.Post("/users/{id}", ah.UserUpdate)
				sr.Post("/users/{id}/delete", ah.UserDelete)
				// Root-only: superadmin alone is not enough to act as another user.
				sr.With(appmw.RequireVirtualRoot).Post("/users/{id}/impersonate", ah.Impersonate)

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

// redirectToEdit sends a bare admin entity URL to that entity's edit form, for the
// read-only detail pages that no longer exist. base is the list path, e.g.
// "/admin/programs".
func redirectToEdit(base string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if _, err := strconv.ParseUint(id, 10, 64); err != nil {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, base+"/"+id+"/edit", http.StatusSeeOther)
	}
}
