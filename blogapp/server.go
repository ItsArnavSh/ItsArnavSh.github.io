package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type App struct {
	cfg     Config
	store   *Store
	render  *Renderer
	site    SiteInfo
	mux     *http.ServeMux
	devices map[string]*deviceFlow
	devMu   sync.Mutex
}

func newApp(cfg Config, store *Store, render *Renderer) *App {
	now := time.Now().In(cfg.Location)
	a := &App{
		cfg:    cfg,
		store:  store,
		render: render,
		site: SiteInfo{
			URL:       cfg.SiteURL,
			MainURL:   cfg.MainSiteURL,
			Title:     "ARNΛV",
			AdminUser: cfg.AdminUser,
			Year:      now.Year(),
		},
		devices: map[string]*deviceFlow{},
	}
	a.routes()
	return a
}

func (a *App) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.handleRoot)
	mux.HandleFunc("GET /devlog", a.handleBlogIndex)
	mux.HandleFunc("GET /journal", a.handleBlogIndex)
	mux.HandleFunc("GET /devlog/feed.xml", a.handleFeed)
	mux.HandleFunc("GET /journal/feed.xml", a.handleFeed)
	mux.HandleFunc("GET /feed.xml", a.handleFeed)
	mux.HandleFunc("GET /devlog/{slug}", a.handlePost)
	mux.HandleFunc("GET /journal/{slug}", a.handlePost)
	mux.HandleFunc("GET /search", a.handleSearch)
	mux.HandleFunc("GET /sitemap.xml", a.handleSitemap)
	mux.HandleFunc("GET /robots.txt", a.handleRobots)
	mux.HandleFunc("GET /download/{blog}/{file}", a.handleDownload)
	mux.HandleFunc("GET /print/{blog}/{slug}", a.handlePrint)
	mux.HandleFunc("GET /api/search", a.handleAPISearch)
	mux.HandleFunc("POST /api/views", a.handleView)
	mux.HandleFunc("GET /media/", a.handleMedia)
	mux.HandleFunc("GET /assets/", a.handleAssets)
	mux.HandleFunc("GET /healthz", a.handleHealth)
	mux.HandleFunc("GET /favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/assets/favicon.svg", http.StatusMovedPermanently)
	})

	mux.HandleFunc("GET /admin/login", a.handleLoginPage)
	mux.HandleFunc("POST /admin/login/start", a.handleLoginStart)
	mux.HandleFunc("GET /admin/login/poll", a.handleLoginPoll)
	mux.HandleFunc("GET /admin/dev-login", a.handleDevLogin)
	mux.HandleFunc("POST /admin/logout", a.admin(a.handleLogout))

	mux.HandleFunc("GET /admin", a.admin(a.handleAdminDashboard))
	mux.HandleFunc("GET /admin/posts/new", a.admin(a.handleEditorNew))
	mux.HandleFunc("POST /admin/posts", a.admin(a.handlePostCreate))
	mux.HandleFunc("GET /admin/posts/{id}", a.admin(a.handleEditorEdit))
	mux.HandleFunc("POST /admin/posts/{id}", a.admin(a.handlePostUpdate))
	mux.HandleFunc("POST /admin/posts/{id}/delete", a.admin(a.handlePostDelete))
	mux.HandleFunc("GET /admin/media", a.admin(a.handleMediaList))
	mux.HandleFunc("POST /admin/upload", a.admin(a.handleUpload))
	mux.HandleFunc("POST /admin/preview", a.admin(a.handlePreview))
	mux.HandleFunc("GET /admin/export", a.admin(a.handleExportAll))

	a.mux = mux
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w}
	w = sw
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' https: data:; media-src 'self' https:; style-src 'self' 'unsafe-inline'; script-src 'self'; font-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")

	// Normalise a few trailing-slash URLs without a round trip for every path.
	if r.Method == http.MethodGet {
		switch {
		case r.URL.Path == "/devlog/", r.URL.Path == "/journal/", r.URL.Path == "/search/", r.URL.Path == "/admin/":
			http.Redirect(w, r, strings.TrimSuffix(r.URL.Path, "/"), http.StatusMovedPermanently)
			return
		}
	}

	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("panic serving request", "path", r.URL.Path, "panic", rec)
			if rec == http.ErrAbortHandler {
				panic(rec)
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", statusOf(w), "dur", time.Since(start).Round(time.Microsecond))
	}()

	// Reject oddly encoded paths early.
	if strings.Contains(r.URL.Path, "\x00") || strings.Contains(r.URL.Path, "..") {
		http.NotFound(w, r)
		return
	}
	a.mux.ServeHTTP(w, r)
}

// statusOf is a best-effort observer for the request log.
func statusOf(w http.ResponseWriter) int {
	if sw, ok := w.(*statusWriter); ok {
		if sw.status == 0 {
			return http.StatusOK
		}
		return sw.status
	}
	return http.StatusOK
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (a *App) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		a.notFound(w, r)
		return
	}
	http.Redirect(w, r, a.cfg.MainSiteURL, http.StatusFound)
}

func (a *App) now() time.Time {
	return time.Now().In(a.cfg.Location)
}

func (a *App) basePage(blogKey, p, title, description string) PageData {
	now := a.now()
	_, week := now.ISOWeek()
	siteJSONLD := template.JS(`{"@context":"https://schema.org","@type":"WebSite","name":"ARNΛV","url":` + jsonString(a.cfg.canonical("/")) + `,"inLanguage":"en","publisher":{"@type":"Person","name":"Arnav Sharma","url":` + jsonString(a.cfg.MainSiteURL) + `}}`)
	return PageData{
		Site:        a.site,
		Blog:        blogInfo(blogKey),
		Blogs:       blogs(),
		Path:        p,
		Title:       title,
		Description: description,
		Canonical:   a.cfg.canonical(p),
		OGImage:     a.cfg.absolute("/assets/og-default.jpg"),
		OGType:      "website",
		Now:         now,
		Issue:       week,
		DevMode:     a.cfg.Dev,
		JSONLD:      siteJSONLD,
	}
}

func (a *App) renderPage(w http.ResponseWriter, name string, data PageData, status int) {
	var buf bytes.Buffer
	if err := a.render.RenderPage(&buf, name, data); err != nil {
		slog.Error("render page", "page", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 && status != http.StatusOK {
		w.WriteHeader(status)
	}
	_, _ = buf.WriteTo(w)
}

func (a *App) renderAdmin(w http.ResponseWriter, name string, data AdminPageData, status int) {
	if data.Admin == "" {
		data.Admin = a.cfg.AdminUser
	}
	if data.Site.URL == "" {
		data.Site = a.site
	}
	if data.Now.IsZero() {
		data.Now = a.now()
	}
	var buf bytes.Buffer
	if err := a.render.RenderAdmin(&buf, name, data); err != nil {
		slog.Error("render admin page", "page", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 && status != http.StatusOK {
		w.WriteHeader(status)
	}
	_, _ = buf.WriteTo(w)
}

func (a *App) renderXML(w http.ResponseWriter, name string, data any) {
	var buf bytes.Buffer
	if err := a.render.RenderXML(&buf, name, data); err != nil {
		slog.Error("render xml", "name", name, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = buf.WriteTo(w)
}

func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	data := a.basePage("devlog", "/404", "Page not found — ARNΛV", "This page went to press without us.")
	data.Canonical = a.cfg.canonical(r.URL.Path)
	data.NoIndex = true
	a.renderPage(w, "404", data, http.StatusNotFound)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "error": "database"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": a.now().Format(time.RFC3339)})
}

func (a *App) handleAssets(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/assets/")
	if name == "" || name == "." || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	sub, err := fs.Sub(embeddedFS, "assets")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, err := sub.Open(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if strings.Contains(name, "/vendor/") || strings.HasPrefix(name, "fonts/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		// Revalidate CSS/JS on every load so a deploy is visible immediately.
		w.Header().Set("Cache-Control", "no-cache")
	}
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, name, stat.ModTime(), rs)
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, name, stat.ModTime(), bytes.NewReader(data))
}

func (a *App) handleMedia(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/media/")
	if name == "" || name == "." || strings.Contains(name, "..") || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	if !allowedMediaName(name) {
		http.NotFound(w, r)
		return
	}

	local := filepath.Join(a.cfg.MediaDir, name)
	if f, err := os.Open(local); err == nil {
		defer f.Close()
		if stat, err := f.Stat(); err == nil && !stat.IsDir() {
			w.Header().Set("Cache-Control", "public, max-age=604800")
			http.ServeContent(w, r, name, stat.ModTime(), f)
			return
		}
	}

	data, err := fs.ReadFile(embeddedFS, "seed/"+name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

func allowedMediaName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".avif":
		return true
	}
	return false
}

func (a *App) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nDisallow: /admin\nDisallow: /api/\nDisallow: /print/\nDisallow: /download/\nSitemap: %s/sitemap.xml\n", a.cfg.SiteURL)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if first := strings.TrimSpace(strings.Split(xff, ",")[0]); first != "" {
			return first
		}
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func visitorHash(secret []byte, r *http.Request, day string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		"arnav-blog-visitor-v1",
		hex.EncodeToString(secret),
		clientIP(r),
		r.UserAgent(),
		day,
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}

func parsePositiveInt(raw string, fallback, max int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func (a *App) admin(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sess, _ := a.sessionFromRequest(r)
		if sess == nil {
			if r.Method == http.MethodGet {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			}
			return
		}
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 14<<20)
			if err := r.ParseForm(); err != nil {
				writeJSONError(w, http.StatusBadRequest, "Could not read the form.")
				return
			}
			if !a.validCSRF(r, r.FormValue("csrf"), r.Header.Get("X-CSRF-Token")) {
				if wantsJSON(r) {
					writeJSONError(w, http.StatusForbidden, "Your session token expired. Reload the page.")
				} else {
					http.Error(w, "Your session token expired. Reload the page.", http.StatusForbidden)
				}
				return
			}
		}
		fn(w, r)
	}
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json") ||
		strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") ||
		r.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

var errNotFound = errors.New("not found")
