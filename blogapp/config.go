package main

import (
	"crypto/rand"
	"crypto/sha256"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	RedisURL     string
	SiteURL      string
	MainSiteURL  string
	MediaDir     string
	Secret       []byte
	AdminUser    string
	GitHubID     string
	GitHubSecret string
	DevLogin     bool
	Seed         bool
	Dev          bool
	SecureCookie bool
	Location     *time.Location
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func loadConfig() Config {
	addr := env("BLOG_ADDR", "")
	if addr == "" {
		p := strings.TrimSpace(os.Getenv("PORT"))
		if p == "" {
			p = "8080"
		}
		if !strings.HasPrefix(p, ":") {
			p = ":" + p
		}
		addr = p
	}

	siteURL := strings.TrimRight(env("SITE_URL", "http://localhost:8080"), "/")
	mainSiteURL := env("MAIN_SITE_URL", "https://itsarnavsh.github.io/")

	locName := env("SITE_TZ", "Asia/Kolkata")
	loc, err := time.LoadLocation(locName)
	if err != nil {
		slog.Warn("unknown SITE_TZ, falling back to UTC", "tz", locName)
		loc = time.UTC
	}

	secretMaterial := env("BLOG_SECRET", env("JWTKEY", ""))
	if secretMaterial == "" {
		secretMaterial = "arnav-blog-development-secret-do-not-use-in-production"
		slog.Warn("BLOG_SECRET/JWTKEY is not set; using the development session key")
	}
	// Domain-separate the session key from any other use of JWTKEY.
	sum := sha256.Sum256([]byte("arnav-blog-session-v1\x00" + secretMaterial))

	return Config{
		Addr:         addr,
		DatabaseURL:  env("DATABASE_URL", "postgres://user:password@localhost:5432/arnav_blog?sslmode=disable"),
		RedisURL:     env("REDIS_URL", "redis://localhost:6379/1"),
		SiteURL:      siteURL,
		MainSiteURL:  mainSiteURL,
		MediaDir:     env("MEDIA_DIR", "./data/media"),
		Secret:       sum[:],
		AdminUser:    env("ADMIN_USERNAME", "ItsArnavSh"),
		GitHubID:     env("GITHUB_CLIENT_ID", env("GITHUB_KEY", "")),
		GitHubSecret: env("GITHUB_CLIENT_SECRET", env("GITHUB_AUTH", "")),
		DevLogin:     envBool("BLOG_DEV_LOGIN", false),
		Seed:         envBool("BLOG_SEED", true),
		Dev:          envBool("BLOG_DEV", false),
		SecureCookie: envBool("BLOG_SECURE_COOKIES", strings.HasPrefix(siteURL, "https://")),
		Location:     loc,
	}
}

func (c Config) canonical(path string) string {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.SiteURL + path
}

func (c Config) absolute(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.SiteURL + path
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hexEncode(b)
}

func hexEncode(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = digits[v>>4]
		out[i*2+1] = digits[v&0x0f]
	}
	return string(out)
}

// databaseName extracts the database name from a Postgres URL.
func databaseName(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(u.Path, "/"), nil
}
