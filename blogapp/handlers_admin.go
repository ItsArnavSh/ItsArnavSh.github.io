package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

func (a *App) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	data, ok := a.signedInAdminBase(w, r, "The Editor's Desk — ARNΛV", "/admin")
	if !ok {
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	blogKey := validBlogFilter(r.URL.Query().Get("blog"))
	stats, err := a.store.AdminStats(r.Context())
	if err != nil {
		slog.Error("admin stats", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	posts, err := a.store.AdminPosts(r.Context(), q, blogKey)
	if err != nil {
		slog.Error("admin posts", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	data.Stats = stats
	data.Posts = posts
	data.Query = q
	data.FilterBlog = blogKey
	data.Flash = flashFor(r.URL.Query().Get("flash"))
	a.renderAdmin(w, "admin_dashboard", data, http.StatusOK)
}

func flashFor(key string) string {
	switch key {
	case "created":
		return "Dispatch created and filed."
	case "updated":
		return "Dispatch saved."
	case "deleted":
		return "Dispatch deleted. The archive is lighter."
	case "exported":
		return "The archive has been packed."
	}
	return ""
}

func (a *App) handleEditorNew(w http.ResponseWriter, r *http.Request) {
	data, ok := a.signedInAdminBase(w, r, "New dispatch — ARNΛV", "/admin/posts/new")
	if !ok {
		return
	}
	blogKey := validBlogFilter(r.URL.Query().Get("blog"))
	if blogKey == "" {
		blogKey = "devlog"
	}
	p := &Post{
		PostCard: PostCard{Blog: blogKey, URL: blogInfo(blogKey).Path},
		Author:   "Arnav Sharma",
		Status:   "draft",
	}
	data.Post = p
	data.Editing = false
	a.renderAdmin(w, "admin_editor", data, http.StatusOK)
}

func (a *App) handleEditorEdit(w http.ResponseWriter, r *http.Request) {
	data, ok := a.signedInAdminBase(w, r, "Edit dispatch — ARNΛV", "/admin")
	if !ok {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	p, err := a.store.GetPostByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("load editor post", "id", id, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	data.Post = p
	data.Editing = true
	data.Path = fmt.Sprintf("/admin/posts/%d", id)
	data.Flash = flashFor(r.URL.Query().Get("flash"))
	a.renderAdmin(w, "admin_editor", data, http.StatusOK)
}

func (a *App) handlePostCreate(w http.ResponseWriter, r *http.Request) {
	data, ok := a.signedInAdminBase(w, r, "New dispatch — ARNΛV", "/admin/posts/new")
	if !ok {
		return
	}
	in, p, msg := a.postInputFromRequest(r, nil)
	if msg != "" {
		a.renderEditorError(w, data, p, false, msg)
		return
	}
	id, err := a.store.CreatePost(r.Context(), in, p.Author)
	if err != nil {
		slog.Error("create post", "error", err)
		a.renderEditorError(w, data, p, false, "Could not save the dispatch. Check the database and try again.")
		return
	}
	a.store.ClearSearchCache(r.Context())
	http.Redirect(w, r, fmt.Sprintf("/admin/posts/%d?flash=created", id), http.StatusSeeOther)
}

func (a *App) handlePostUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	data, ok := a.signedInAdminBase(w, r, "Edit dispatch — ARNΛV", fmt.Sprintf("/admin/posts/%d", id))
	if !ok {
		return
	}
	existing, err := a.store.GetPostByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	in, p, msg := a.postInputFromRequest(r, existing)
	if msg != "" {
		p.ID = existing.ID
		p.CreatedAt = existing.CreatedAt
		p.UpdatedAt = existing.UpdatedAt
		a.renderEditorError(w, data, p, true, msg)
		return
	}
	if err := a.store.UpdatePost(r.Context(), id, in); err != nil {
		slog.Error("update post", "id", id, "error", err)
		a.renderEditorError(w, data, p, true, "Could not save the dispatch. Check the database and try again.")
		return
	}
	a.store.ClearSearchCache(r.Context())
	http.Redirect(w, r, fmt.Sprintf("/admin/posts/%d?flash=updated", id), http.StatusSeeOther)
}

func (a *App) renderEditorError(w http.ResponseWriter, data AdminPageData, p *Post, editing bool, msg string) {
	data.Post = p
	data.Editing = editing
	data.Error = msg
	if editing {
		data.Path = fmt.Sprintf("/admin/posts/%d", p.ID)
		data.Title = "Edit dispatch — ARNΛV"
	} else {
		data.Path = "/admin/posts/new"
		data.Title = "New dispatch — ARNΛV"
	}
	a.renderAdmin(w, "admin_editor", data, http.StatusUnprocessableEntity)
}

func (a *App) postInputFromRequest(r *http.Request, existing *Post) (PostInput, *Post, string) {
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		return PostInput{}, &Post{}, "Every dispatch needs a headline."
	}
	if utf8.RuneCountInString(title) > 220 {
		return PostInput{}, &Post{}, "That headline is too long. Keep it under 220 characters."
	}
	blogKey := r.FormValue("blog")
	if _, ok := blogCatalog[blogKey]; !ok {
		blogKey = "devlog"
	}
	body := strings.ReplaceAll(r.FormValue("body_md"), "\r\n", "\n")
	if utf8.RuneCountInString(body) > 400000 {
		return PostInput{}, &Post{}, "That dispatch is too large. Keep the body under 400,000 characters."
	}
	status := r.FormValue("status")
	if status != "published" && status != "draft" {
		status = "draft"
	}
	if status == "published" && strings.TrimSpace(body) == "" {
		return PostInput{}, &Post{}, "A published dispatch cannot have an empty body. Save it as a draft instead."
	}
	slugInput := strings.TrimSpace(r.FormValue("slug"))
	if slugInput == "" {
		slugInput = title
	}
	slug := slugify(slugInput)
	if slug == "" {
		return PostInput{}, &Post{}, "That slug cannot be turned into a URL. Try letters and numbers."
	}
	excerpt := strings.TrimSpace(r.FormValue("excerpt"))
	if excerpt == "" {
		excerpt = excerptFrom(body, 240)
	}
	if utf8.RuneCountInString(excerpt) > 700 {
		excerpt = truncate(excerpt, 700)
	}
	cover := strings.TrimSpace(r.FormValue("cover_image"))
	if cover != "" && !validImageURL(cover) {
		return PostInput{}, &Post{}, "The cover image must be an /media/ path or an https:// URL."
	}
	author := strings.TrimSpace(r.FormValue("author"))
	if author == "" {
		author = "Arnav Sharma"
	}

	var existingID int64
	var createdAt time.Time
	if existing != nil {
		existingID = existing.ID
		createdAt = existing.CreatedAt
	}
	taken, err := a.store.SlugTaken(r.Context(), blogKey, slug, existingID)
	if err != nil {
		slog.Error("slug check", "error", err)
		return PostInput{}, &Post{}, "Could not check that URL slug. Try again."
	}
	if taken {
		return PostInput{}, &Post{}, "Another dispatch in this edition already uses that URL slug. Choose a different one."
	}

	var publishedAt *time.Time
	rawDate := strings.TrimSpace(r.FormValue("published_at"))
	if rawDate != "" {
		t, ok := parseLocalDateTime(rawDate, a.cfg.Location)
		if !ok {
			return PostInput{}, &Post{}, "The publish date is not valid. Use the date picker or leave it blank."
		}
		publishedAt = &t
	} else if status == "published" {
		if existing != nil && existing.HasPublishedAt {
			t := existing.PublishedAt
			publishedAt = &t
		} else {
			t := a.now()
			publishedAt = &t
		}
	} else if existing != nil && existing.HasPublishedAt {
		t := existing.PublishedAt
		publishedAt = &t
	}

	in := PostInput{
		Blog:       blogKey,
		Slug:       slug,
		Title:      title,
		Subtitle:   strings.TrimSpace(r.FormValue("subtitle")),
		Excerpt:    excerpt,
		CoverImage: cover,
		CoverAlt:   strings.TrimSpace(r.FormValue("cover_alt")),
		BodyMD:     body,
		Author:     author,
		Tags:       parseTags(r.FormValue("tags")),
		Status:     status,
		Featured:   r.FormValue("featured") == "1",
		Published:  publishedAt,
	}
	p := &Post{
		PostCard: PostCard{
			Blog:           in.Blog,
			Slug:           in.Slug,
			Title:          in.Title,
			Subtitle:       in.Subtitle,
			Excerpt:        in.Excerpt,
			CoverImage:     in.CoverImage,
			CoverAlt:       in.CoverAlt,
			Tags:           in.Tags,
			Featured:       in.Featured,
			ReadingMinutes: readingMinutes(in.BodyMD),
			URL:            blogInfo(in.Blog).Path + "/" + in.Slug,
		},
		BodyMD:         in.BodyMD,
		Author:         in.Author,
		Status:         in.Status,
		HasPublishedAt: in.Published != nil,
		CreatedAt:      createdAt,
		UpdatedAt:      a.now(),
	}
	if in.Published != nil {
		p.PublishedAt = *in.Published
	}
	if existing != nil {
		p.ID = existing.ID
	}
	return in, p, ""
}

func validImageURL(v string) bool {
	if strings.HasPrefix(v, "/media/") {
		return true
	}
	if strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "http://") {
		return true
	}
	return false
}

func parseLocalDateTime(raw string, loc *time.Location) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04"} {
		if t, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (a *App) handlePostDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}
	if err := a.store.DeletePost(r.Context(), id); err != nil {
		slog.Error("delete post", "id", id, "error", err)
		http.Error(w, "could not delete the dispatch", http.StatusInternalServerError)
		return
	}
	a.store.ClearSearchCache(r.Context())
	http.Redirect(w, r, "/admin?flash=deleted", http.StatusSeeOther)
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		writeJSONError(w, http.StatusBadRequest, "That upload was too large or malformed.")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Choose an image to upload.")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 12<<20+1))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Could not read that image.")
		return
	}
	if len(data) == 0 {
		writeJSONError(w, http.StatusBadRequest, "That image is empty.")
		return
	}
	if len(data) > 12<<20 {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "Images must be smaller than 12 MB.")
		return
	}
	contentType := http.DetectContentType(data)
	ext := ""
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	default:
		writeJSONError(w, http.StatusUnsupportedMediaType, "Upload a JPEG, PNG, WebP or GIF image.")
		return
	}

	base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	base = slugify(base)
	if base == "" || base == "dispatch" {
		base = "image"
	}
	name := fmt.Sprintf("%s-%s-%s%s", time.Now().In(a.cfg.Location).Format("20060102-150405"), base, randomHex(3), ext)
	if err := ensureDir(a.cfg.MediaDir); err != nil {
		slog.Error("create media dir", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "Could not create the media directory.")
		return
	}
	dest := filepath.Join(a.cfg.MediaDir, name)
	if err := writeFile(dest, data); err != nil {
		slog.Error("write upload", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "Could not save that image.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"url":  "/media/" + name,
		"name": name,
		"size": len(data),
		"type": contentType,
	})
}

func (a *App) handleMediaList(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(a.cfg.MediaDir)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Error("read media dir", "error", err)
		writeJSONError(w, http.StatusInternalServerError, "Could not read the media library.")
		return
	}
	items := make([]MediaItem, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !allowedMediaName(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		items = append(items, MediaItem{
			Name: entry.Name(),
			URL:  "/media/" + entry.Name(),
			Size: info.Size(),
			Mod:  info.ModTime(),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Mod.After(items[j].Mod)
	})
	if len(items) > 60 {
		items = items[:60]
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func ensureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func writeFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o644)
}

func (a *App) handlePreview(w http.ResponseWriter, r *http.Request) {
	var markdown string
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		var in struct {
			Markdown string `json:"markdown"`
		}
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
		if err := dec.Decode(&in); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Could not read the draft.")
			return
		}
		markdown = in.Markdown
	} else {
		markdown = r.FormValue("markdown")
	}
	if utf8.RuneCountInString(markdown) > 400000 {
		writeJSONError(w, http.StatusRequestEntityTooLarge, "That draft is too large to preview.")
		return
	}
	body, toc := a.render.RenderMarkdown(markdown)
	words := len(strings.Fields(stripMarkdown(markdown)))
	writeJSON(w, http.StatusOK, map[string]any{
		"html":    body,
		"words":   words,
		"reading": readingMinutes(markdown),
		"toc":     toc,
	})
}

func (a *App) handleExportAll(w http.ResponseWriter, r *http.Request) {
	cards, err := a.store.AllPublished(r.Context(), "")
	if err != nil {
		slog.Error("export all", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	var index strings.Builder
	index.WriteString("# ARNΛV — all dispatches\n\n")
	index.WriteString("A plain-Markdown snapshot of the archive, packed from the editor's desk.\n\n")
	for _, c := range cards {
		p, err := a.store.GetPost(r.Context(), c.Blog, c.Slug)
		if err != nil {
			continue
		}
		md := markdownExport(p, a.cfg.SiteURL)
		name := fmt.Sprintf("%s/%s.md", p.Blog, p.Slug)
		zwFile, err := zw.Create(name)
		if err == nil {
			_, _ = zwFile.Write([]byte(md))
		}
		fmt.Fprintf(&index, "- [%s](%s) — %s, %s\n", p.Title, name, blogInfo(p.Blog).Name, p.PublishedAt.In(a.cfg.Location).Format("02 Jan 2006"))
	}
	readme, err := zw.Create("README.md")
	if err == nil {
		_, _ = readme.Write([]byte(index.String()))
	}
	if err := zw.Close(); err != nil {
		slog.Error("close export zip", "error", err)
		http.Error(w, "could not pack the archive", http.StatusInternalServerError)
		return
	}
	downloadHeader(w, "arnav-blog-"+a.now().Format("2006-01-02")+".zip", "application/zip")
	_, _ = buf.WriteTo(w)
}
