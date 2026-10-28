package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookie = "arnav_blog_session"
	deviceCookie  = "arnav_blog_device"
	sessionDays   = 30
)

type sessionData struct {
	User    string `json:"u"`
	Expires int64  `json:"e"`
}

type deviceFlow struct {
	ID              string
	DeviceCode      string
	UserCode        string
	VerificationURI string
	ExpiresAt       time.Time
	Interval        int
	LastPoll        time.Time
	Status          string
	Message         string
	Session         string
	mu              sync.Mutex
}

func (a *App) sign(value string) string {
	mac := hmac.New(sha256.New, a.cfg.Secret)
	mac.Write([]byte(value))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *App) newSessionToken(user string) string {
	payload, _ := json.Marshal(sessionData{User: user, Expires: time.Now().Add(sessionDays * 24 * time.Hour).Unix()})
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + a.sign(body)
}

func (a *App) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   sessionDays * 24 * 60 * 60,
		Expires:  time.Now().Add(sessionDays * 24 * time.Hour),
	})
}

func (a *App) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func (a *App) sessionFromRequest(r *http.Request) (*sessionData, string) {
	c, err := r.Cookie(sessionCookie)
	if err != nil || c.Value == "" {
		return nil, ""
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 2 {
		return nil, ""
	}
	expected := a.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[1])) != 1 {
		return nil, ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ""
	}
	var sess sessionData
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, ""
	}
	if sess.User == "" || sess.Expires < time.Now().Unix() {
		return nil, ""
	}
	if !strings.EqualFold(sess.User, a.cfg.AdminUser) {
		return nil, ""
	}
	return &sess, c.Value
}

func (a *App) csrfFor(token string) string {
	mac := hmac.New(sha256.New, a.cfg.Secret)
	mac.Write([]byte("csrf\x00" + token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *App) validCSRF(r *http.Request, candidates ...string) bool {
	_, token := a.sessionFromRequest(r)
	if token == "" {
		return false
	}
	expected := a.csrfFor(token)
	for _, got := range candidates {
		if got != "" && subtle.ConstantTimeCompare([]byte(expected), []byte(got)) == 1 {
			return true
		}
	}
	return false
}

func (a *App) adminBase(title, p string) AdminPageData {
	return AdminPageData{
		Site:     a.site,
		Title:    title,
		Path:     p,
		Admin:    a.cfg.AdminUser,
		DevLogin: a.cfg.DevLogin,
		DevMode:  a.cfg.Dev,
		Now:      a.now(),
	}
}

func (a *App) signedInAdminBase(w http.ResponseWriter, r *http.Request, title, p string) (AdminPageData, bool) {
	sess, token := a.sessionFromRequest(r)
	if sess == nil {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return AdminPageData{}, false
	}
	data := a.adminBase(title, p)
	data.Admin = sess.User
	data.SignedIn = true
	data.CSRF = a.csrfFor(token)
	data.CSRFToken = data.CSRF
	return data, true
}

func (a *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if sess, _ := a.sessionFromRequest(r); sess != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	data := a.adminBase("Editor's entrance — ARNΛV", "/admin/login")
	data.Error = strings.TrimSpace(r.URL.Query().Get("error"))
	if a.cfg.GitHubID == "" {
		data.Error = "GitHub sign-in is not configured on this server. Set GITHUB_CLIENT_ID."
	}
	a.renderAdmin(w, "admin_login", data, http.StatusOK)
}

func (a *App) handleLoginStart(w http.ResponseWriter, r *http.Request) {
	if sess, _ := a.sessionFromRequest(r); sess != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	if a.cfg.GitHubID == "" {
		http.Redirect(w, r, "/admin/login?error=GitHub+sign-in+is+not+configured", http.StatusSeeOther)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	flow, err := a.requestDeviceCode(ctx)
	if err != nil {
		slog.Error("github device code", "error", err)
		data := a.adminBase("Editor's entrance — ARNΛV", "/admin/login")
		data.Error = "GitHub could not start a device sign-in. Try again in a moment."
		a.renderAdmin(w, "admin_login", data, http.StatusBadGateway)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     deviceCookie,
		Value:    flow.ID,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   a.cfg.SecureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   900,
	})
	data := a.adminBase("Editor's entrance — ARNΛV", "/admin/login")
	data.Admin = a.cfg.AdminUser
	data.Device = &DeviceLoginView{
		Code:            flow.UserCode,
		VerificationURI: flow.VerificationURI,
		CompleteURI:     flow.VerificationURI,
		ExpiresIn:       int(time.Until(flow.ExpiresAt).Seconds()),
		Interval:        flow.Interval,
		Status:          "pending",
		Message:         "Waiting for GitHub…",
	}
	a.renderAdmin(w, "admin_login", data, http.StatusOK)
}

func (a *App) handleLoginPoll(w http.ResponseWriter, r *http.Request) {
	if sess, _ := a.sessionFromRequest(r); sess != nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "redirect": "/admin"})
		return
	}
	c, err := r.Cookie(deviceCookie)
	if err != nil || c.Value == "" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": "This sign-in attempt expired. Start again."})
		return
	}
	a.devMu.Lock()
	flow := a.devices[c.Value]
	a.devMu.Unlock()
	if flow == nil {
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": "This sign-in attempt expired. Start again."})
		return
	}

	flow.mu.Lock()
	defer flow.mu.Unlock()

	if flow.Status == "complete" && flow.Session != "" {
		a.setSessionCookie(w, flow.Session)
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "redirect": "/admin"})
		return
	}
	if flow.Status == "error" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": flow.Message})
		return
	}
	if time.Now().After(flow.ExpiresAt) {
		flow.Status = "error"
		flow.Message = "The GitHub code expired. Start a new sign-in."
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": flow.Message})
		return
	}
	wait := time.Duration(flow.Interval) * time.Second
	if wait < 5*time.Second {
		wait = 5 * time.Second
	}
	if !flow.LastPoll.IsZero() && time.Since(flow.LastPoll) < wait {
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending", "message": "Waiting for GitHub…"})
		return
	}
	flow.LastPoll = time.Now()

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	token, errCode, message, err := a.exchangeDeviceToken(ctx, flow.DeviceCode)
	if err != nil {
		slog.Warn("github token exchange", "error", err)
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending", "message": "GitHub is taking a moment…"})
		return
	}
	switch errCode {
	case "", "slow_down":
		if errCode == "slow_down" {
			flow.Interval += 5
		}
		if token == "" {
			writeJSON(w, http.StatusOK, map[string]any{"status": "pending", "message": "Waiting for GitHub…"})
			return
		}
	case "authorization_pending":
		writeJSON(w, http.StatusOK, map[string]any{"status": "pending", "message": "Waiting for GitHub…"})
		return
	default:
		flow.Status = "error"
		flow.Message = message
		if flow.Message == "" {
			flow.Message = "GitHub rejected the sign-in."
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": flow.Message})
		return
	}

	login, err := a.githubLogin(ctx, token)
	if err != nil {
		slog.Warn("github user lookup", "error", err)
		flow.Status = "error"
		flow.Message = "Signed in with GitHub, but the account could not be verified."
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": flow.Message})
		return
	}
	if !strings.EqualFold(login, a.cfg.AdminUser) {
		flow.Status = "error"
		flow.Message = fmt.Sprintf("The account @%s is not the editor of this paper.", login)
		slog.Warn("rejected admin sign-in", "github", login)
		writeJSON(w, http.StatusOK, map[string]any{"status": "error", "message": flow.Message})
		return
	}

	flow.Session = a.newSessionToken(login)
	flow.Status = "complete"
	a.setSessionCookie(w, flow.Session)
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "redirect": "/admin"})
}

func (a *App) handleDevLogin(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.DevLogin {
		http.NotFound(w, r)
		return
	}
	a.setSessionCookie(w, a.newSessionToken(a.cfg.AdminUser))
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.clearSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

func (a *App) githubPost(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "arnav-blog/1.0")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("github returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}

func (a *App) requestDeviceCode(ctx context.Context) (*deviceFlow, error) {
	body, err := a.githubPost(ctx, "https://github.com/login/device/code", url.Values{
		"client_id": {a.cfg.GitHubID},
		"scope":     {"read:user"},
	})
	if err != nil {
		return nil, err
	}
	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, err
	}
	if dc.DeviceCode == "" || dc.UserCode == "" {
		return nil, fmt.Errorf("github returned an empty device code")
	}
	if dc.Interval <= 0 {
		dc.Interval = 5
	}
	if dc.ExpiresIn <= 0 {
		dc.ExpiresIn = 900
	}
	flow := &deviceFlow{
		ID:              randomHex(16),
		DeviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		VerificationURI: dc.VerificationURI,
		ExpiresAt:       time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second),
		Interval:        dc.Interval,
		Status:          "pending",
	}
	a.devMu.Lock()
	a.devices[flow.ID] = flow
	// Keep the in-memory flow table small.
	for id, f := range a.devices {
		if time.Since(f.ExpiresAt) > 2*time.Hour {
			delete(a.devices, id)
		}
	}
	a.devMu.Unlock()
	return flow, nil
}

func (a *App) exchangeDeviceToken(ctx context.Context, deviceCode string) (string, string, string, error) {
	body, err := a.githubPost(ctx, "https://github.com/login/oauth/access_token", url.Values{
		"client_id":   {a.cfg.GitHubID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		return "", "", "", err
	}
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", "", "", err
	}
	if tok.Error != "" {
		msg := tok.ErrorDescription
		switch tok.Error {
		case "authorization_pending":
			msg = "Waiting for GitHub…"
		case "slow_down":
			msg = "Waiting for GitHub…"
		case "expired_token":
			msg = "The GitHub code expired. Start a new sign-in."
		case "access_denied":
			msg = "The sign-in was cancelled on GitHub."
		case "incorrect_client_credentials":
			msg = "This GitHub app is not configured correctly."
		case "device_flow_disabled":
			msg = "Device flow is disabled for this GitHub app."
		}
		return "", tok.Error, msg, nil
	}
	return tok.AccessToken, "", "", nil
}

func (a *App) githubLogin(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "arnav-blog/1.0")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github user endpoint returned %s", resp.Status)
	}
	var user struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &user); err != nil {
		return "", err
	}
	if user.Login == "" {
		return "", fmt.Errorf("github returned an empty login")
	}
	return user.Login, nil
}
