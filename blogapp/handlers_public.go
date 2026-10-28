package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (a *App) handleBlogIndex(w http.ResponseWriter, r *http.Request) {
	blogKey := strings.TrimPrefix(r.URL.Path, "/")
	if _, ok := blogCatalog[blogKey]; !ok {
		a.notFound(w, r)
		return
	}
	b := blogInfo(blogKey)
	cards, err := a.store.ListPublished(r.Context(), blogKey, 60)
	if err != nil {
		slog.Error("list posts", "blog", blogKey, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	tags, err := a.store.Tags(r.Context(), blogKey)
	if err != nil {
		slog.Warn("load tags", "error", err)
	}
	trending, err := a.store.Trending(r.Context(), blogKey, 5)
	if err != nil {
		slog.Warn("load trending", "error", err)
	}
	total, err := a.store.CountPublished(r.Context(), blogKey)
	if err != nil {
		total = len(cards)
	}

	data := a.basePage(blogKey, b.Path, b.Name+" — ARNΛV", b.Description)
	data.Posts = cards
	if len(cards) > 0 {
		lead := cards[0]
		data.Lead = &lead
		if len(cards) > 1 {
			end := 3
			if end > len(cards) {
				end = len(cards)
			}
			data.Secondary = cards[1:end]
		}
		if len(cards) > 3 {
			data.More = cards[3:]
		}
	}
	data.Tags = tags
	data.Trending = trending
	data.TotalPosts = total
	data.JSONLD = template.JS(`{"@context":"https://schema.org","@type":"Blog","name":` + jsonString(b.Name) + `,"url":` + jsonString(a.cfg.canonical(b.Path)) + `,"description":` + jsonString(b.Description) + `}`)
	a.renderPage(w, "index", data, http.StatusOK)
}

func (a *App) handlePost(w http.ResponseWriter, r *http.Request) {
	blogKey := strings.TrimPrefix(r.URL.Path, "/")
	blogKey = strings.Split(blogKey, "/")[0]
	slug := r.PathValue("slug")
	if _, ok := blogCatalog[blogKey]; !ok || slug == "" {
		a.notFound(w, r)
		return
	}
	p, err := a.store.GetPost(r.Context(), blogKey, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			a.notFound(w, r)
			return
		}
		slog.Error("load post", "blog", blogKey, "slug", slug, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	body, toc := a.render.RenderMarkdown(p.BodyMD)
	p.BodyHTML = template.HTML(body)
	p.TOC = toc
	p.WordCount = len(strings.Fields(stripMarkdown(p.BodyMD)))

	related, err := a.store.Related(r.Context(), p, 4)
	if err != nil {
		slog.Warn("related posts", "error", err)
	}
	filtered := related[:0]
	for _, c := range related {
		if c.ID != p.ID {
			filtered = append(filtered, c)
		}
	}
	prev, next, err := a.store.Neighbors(r.Context(), p)
	if err != nil {
		slog.Warn("neighbors", "error", err)
	}

	b := blogInfo(blogKey)
	desc := strings.TrimSpace(p.Excerpt)
	if desc == "" {
		desc = excerptFrom(p.BodyMD, 180)
	}
	data := a.basePage(blogKey, p.URL, p.Title+" — "+b.Name+" — ARNΛV", desc)
	data.Post = p
	data.Prev = prev
	data.Next = next
	data.Related = filtered
	data.Canonical = a.cfg.canonical(p.URL)
	data.OGType = "article"
	if p.CoverImage != "" {
		data.OGImage = a.cfg.absolute(p.CoverImage)
	}
	data.JSONLD = postJSONLD(p, data.Canonical, data.OGImage)
	a.renderPage(w, "post", data, http.StatusOK)
}

func (a *App) handlePrint(w http.ResponseWriter, r *http.Request) {
	blogKey := r.PathValue("blog")
	slug := r.PathValue("slug")
	if _, ok := blogCatalog[blogKey]; !ok {
		a.notFound(w, r)
		return
	}
	p, err := a.store.GetPost(r.Context(), blogKey, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			a.notFound(w, r)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	body, toc := a.render.RenderMarkdown(p.BodyMD)
	p.BodyHTML = template.HTML(body)
	p.TOC = toc
	p.WordCount = len(strings.Fields(stripMarkdown(p.BodyMD)))

	data := a.basePage(blogKey, p.URL, p.Title+" — print — ARNΛV", p.Excerpt)
	data.Post = p
	data.Canonical = a.cfg.canonical(p.URL)
	data.PrintMode = true
	data.NoIndex = true
	data.OGType = "article"
	if p.CoverImage != "" {
		data.OGImage = a.cfg.absolute(p.CoverImage)
	}
	a.renderPage(w, "post", data, http.StatusOK)
}

func postJSONLD(p *Post, canonical, image string) template.JS {
	ld := map[string]any{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         p.Title,
		"description":      p.Excerpt,
		"datePublished":    p.PublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"dateModified":     p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"author":           map[string]any{"@type": "Person", "name": p.Author},
		"publisher":        map[string]any{"@type": "Organization", "name": "ARNΛV"},
		"mainEntityOfPage": canonical,
		"url":              canonical,
		"articleSection":   blogInfo(p.Blog).Name,
		"keywords":         strings.Join(p.Tags, ", "),
	}
	if image != "" {
		ld["image"] = image
	}
	raw, _ := json.Marshal(ld)
	return template.JS(raw)
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	blogKey := validBlogFilter(r.URL.Query().Get("blog"))
	results, err := a.store.Search(r.Context(), q, blogKey, 50)
	if err != nil {
		slog.Error("search", "q", q, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	tags, _ := a.store.Tags(r.Context(), blogKey)
	trending, _ := a.store.Trending(r.Context(), blogKey, 5)

	title := "Search the archive — ARNΛV"
	if q != "" {
		title = "“" + q + "” — search — ARNΛV"
	}
	data := a.basePage("devlog", "/search", title, "Search both editions of ARNΛV: the Devlog, the Journal, and the whole archive.")
	data.SearchQuery = q
	data.SearchBlog = blogKey
	data.SearchResults = results
	data.SearchCount = len(results)
	data.Tags = tags
	data.Trending = trending
	data.NoIndex = q != "" || blogKey != ""
	a.renderPage(w, "search", data, http.StatusOK)
}

func validBlogFilter(v string) string {
	if _, ok := blogCatalog[v]; ok {
		return v
	}
	return ""
}

func (a *App) handleAbout(w http.ResponseWriter, r *http.Request) {
	data := a.basePage("journal", "/about", "About this paper — ARNΛV", "Two editions, one desk. How ARNΛV is written, printed and kept.")
	a.renderPage(w, "about", data, http.StatusOK)
}

func (a *App) handleFeed(w http.ResponseWriter, r *http.Request) {
	blogKey := ""
	p := "/feed.xml"
	feedPath := "/feed.xml"
	title := "ARNΛV — All editions"
	description := "Every dispatch from the Devlog and the Journal."
	if r.URL.Path != "/feed.xml" {
		blogKey = strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/feed.xml"), "/")
		if _, ok := blogCatalog[blogKey]; !ok {
			a.notFound(w, r)
			return
		}
		b := blogInfo(blogKey)
		p = b.Path
		feedPath = b.Path + "/feed.xml"
		title = "ARNΛV — " + b.Name
		description = b.Description
	}

	cards, err := a.store.ListPublished(r.Context(), blogKey, 40)
	if err != nil {
		slog.Error("feed posts", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	items := make([]FeedItem, 0, len(cards))
	for _, c := range cards {
		full, err := a.store.GetPost(r.Context(), c.Blog, c.Slug)
		if err != nil {
			continue
		}
		body, _ := a.render.RenderMarkdown(full.BodyMD)
		items = append(items, FeedItem{PostCard: c, BodyHTML: template.HTML(absolutizeHTML(body, a.cfg.SiteURL))})
	}
	a.renderXML(w, "feed.xml", FeedData{
		Site:        a.site,
		Title:       title,
		Description: description,
		Path:        p,
		FeedPath:    feedPath,
		Now:         a.now(),
		Posts:       items,
	})
}

func (a *App) handleSitemap(w http.ResponseWriter, r *http.Request) {
	posts, err := a.store.AllPublished(r.Context(), "")
	if err != nil {
		slog.Error("sitemap", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	a.renderXML(w, "sitemap.xml", SitemapData{Posts: posts, Now: a.now()})
}

func (a *App) handleAPISearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	limit := parsePositiveInt(r.URL.Query().Get("limit"), 6, 12)
	blogKey := validBlogFilter(r.URL.Query().Get("blog"))
	type hit struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Blog    string `json:"blog"`
		Date    string `json:"date"`
		Snippet string `json:"snippet,omitempty"`
	}
	hits := make([]hit, 0, limit)
	if len([]rune(q)) >= 2 {
		results, err := a.store.Search(r.Context(), q, blogKey, limit)
		if err != nil {
			slog.Error("api search", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "Search is unavailable.")
			return
		}
		for _, res := range results {
			hits = append(hits, hit{
				Title:   res.Title,
				URL:     res.URL,
				Blog:    blogInfo(res.Blog).Name,
				Date:    res.PublishedAt.In(a.cfg.Location).Format("02 Jan 2006"),
				Snippet: truncate(res.Excerpt, 120),
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"query": q, "results": hits})
}

func (a *App) handleView(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Blog string `json:"blog"`
		Slug string `json:"slug"`
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	if err := dec.Decode(&in); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Bad request.")
		return
	}
	in.Blog = strings.TrimSpace(in.Blog)
	in.Slug = strings.TrimSpace(in.Slug)
	if _, ok := blogCatalog[in.Blog]; !ok || in.Slug == "" {
		writeJSONError(w, http.StatusBadRequest, "Unknown dispatch.")
		return
	}
	p, err := a.store.GetPost(r.Context(), in.Blog, in.Slug)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "Unknown dispatch.")
		return
	}
	now := a.now()
	views, counted, err := a.store.IncrementView(r.Context(), p.ID, visitorHash(a.cfg.Secret, r, now.Format("2006-01-02")), now)
	if err != nil {
		slog.Error("increment view", "post", p.ID, "error", err)
		writeJSONError(w, http.StatusInternalServerError, "Could not count that view.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"views": views, "counted": counted})
}

func (a *App) handleDownload(w http.ResponseWriter, r *http.Request) {
	blogKey := r.PathValue("blog")
	file := r.PathValue("file")
	if _, ok := blogCatalog[blogKey]; !ok || file == "" {
		http.NotFound(w, r)
		return
	}
	ext := strings.ToLower(file[strings.LastIndex(file, ".")+1:])
	slug := strings.TrimSuffix(file, "."+ext)
	if slug == "" || slug == file || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	p, err := a.store.GetPost(r.Context(), blogKey, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("download post", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	body, _ := a.render.RenderMarkdown(p.BodyMD)
	p.BodyHTML = template.HTML(body)
	p.WordCount = len(strings.Fields(stripMarkdown(p.BodyMD)))

	base := slug
	switch ext {
	case "md":
		markdown := markdownExport(p, a.cfg.SiteURL)
		downloadHeader(w, base+".md", "text/markdown; charset=utf-8")
		_, _ = w.Write([]byte(markdown))
	case "txt":
		text := textExport(p)
		downloadHeader(w, base+".txt", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(text))
	case "html":
		var buf bytes.Buffer
		p.BodyHTML = template.HTML(absolutizeHTML(body, a.cfg.SiteURL))
		if err := a.render.RenderStandalone(&buf, "download", "download", DownloadData{Site: a.site, Post: p, Now: a.now()}); err != nil {
			slog.Error("download html", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		downloadHeader(w, base+".html", "text/html; charset=utf-8")
		_, _ = buf.WriteTo(w)
	case "epub":
		data, err := buildEPUB(p, a.cfg, body)
		if err != nil {
			slog.Error("build epub", "error", err)
			http.Error(w, "could not build the EPUB", http.StatusInternalServerError)
			return
		}
		downloadHeader(w, base+".epub", "application/epub+zip")
		_, _ = w.Write(data)
	case "json":
		payload := map[string]any{
			"id":              p.ID,
			"blog":            p.Blog,
			"edition":         blogInfo(p.Blog).Name,
			"slug":            p.Slug,
			"title":           p.Title,
			"subtitle":        p.Subtitle,
			"excerpt":         p.Excerpt,
			"author":          p.Author,
			"published_at":    p.PublishedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"updated_at":      p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			"tags":            p.Tags,
			"cover_image":     p.CoverImage,
			"cover_alt":       p.CoverAlt,
			"canonical_url":   a.cfg.canonical(p.URL),
			"reading_minutes": p.ReadingMinutes,
			"word_count":      p.WordCount,
			"markdown":        p.BodyMD,
			"html":            body,
		}
		raw, _ := json.MarshalIndent(payload, "", "  ")
		downloadHeader(w, base+".json", "application/json; charset=utf-8")
		_, _ = w.Write(raw)
	default:
		http.NotFound(w, r)
	}
}

func downloadHeader(w http.ResponseWriter, filename, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mimeFormatAttachment(filename))
	w.Header().Set("Cache-Control", "private, max-age=0")
}

func mimeFormatAttachment(filename string) string {
	return fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(filename, `"`, ""))
}

var localAssetRe = regexp.MustCompile(`(src|href)="(/[^"]*)"`)

func absolutizeHTML(body, site string) string {
	return localAssetRe.ReplaceAllString(body, `$1="`+strings.TrimRight(site, "/")+`$2"`)
}

func jsonString(s string) string {
	raw, _ := json.Marshal(s)
	return string(raw)
}
