package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"html"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func markdownExport(p *Post, site string) string {
	site = strings.TrimRight(site, "/")
	body := p.BodyMD
	if site != "" {
		body = strings.ReplaceAll(body, "](/", "]("+site+"/")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", p.Title)
	if p.Subtitle != "" {
		fmt.Fprintf(&b, "*%s*\n\n", p.Subtitle)
	}
	fmt.Fprintf(&b, "By %s · %s · %d min read\n\n", p.Author, p.PublishedAt.Format("2 January 2006"), p.ReadingMinutes)
	if len(p.Tags) > 0 {
		fmt.Fprintf(&b, "Filed under: %s\n\n", strings.Join(p.Tags, ", "))
	}
	if site != "" {
		fmt.Fprintf(&b, "Canonical: %s%s\n\n", site, p.URL)
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteString("\n\n---\n\n")
	fmt.Fprintf(&b, "*Downloaded from ARNΛV — %s. Standard Markdown, no lock-in.*\n", blogInfo(p.Blog).Name)
	return b.String()
}

func textExport(p *Post) string {
	var b strings.Builder
	b.WriteString(p.Title)
	b.WriteString("\n")
	if p.Subtitle != "" {
		b.WriteString(p.Subtitle)
		b.WriteString("\n")
	}
	b.WriteString(strings.Repeat("=", minInt(len([]rune(p.Title)), 72)))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "By %s | %s | %d min read\n", p.Author, p.PublishedAt.Format("2 January 2006"), p.ReadingMinutes)
	if len(p.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(p.Tags, ", "))
	}
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(stripMarkdown(p.BodyMD)))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "— ARNΛV, %s edition. Downloaded %s.\n", blogInfo(p.Blog).Name, time.Now().Format("2 January 2006"))
	return b.String()
}

var (
	epubImgSrcRe  = regexp.MustCompile(`src="/media/([^"]+)"`)
	epubVoidImgRe = regexp.MustCompile(`<img([^>]*?)\s*/?>`)
	epubVoidInRe  = regexp.MustCompile(`<input([^>]*?)\s*/?>`)
	epubHrRe      = regexp.MustCompile(`<hr\s*/?>`)
)

type epubAsset struct {
	Name string
	Data []byte
	ID   string
}

func buildEPUB(p *Post, cfg Config, body string) ([]byte, error) {
	assets := map[string]*epubAsset{}
	order := []string{}
	addAsset := func(rawName string) string {
		rawName = filepath.Base(rawName)
		if rawName == "" || rawName == "." || strings.Contains(rawName, "..") {
			return ""
		}
		if existing, ok := assets[rawName]; ok {
			return existing.Name
		}
		data, err := readMediaFile(cfg, rawName)
		if err != nil {
			return ""
		}
		a := &epubAsset{Name: rawName, Data: data, ID: fmt.Sprintf("img%d", len(order)+1)}
		assets[rawName] = a
		order = append(order, rawName)
		return a.Name
	}

	chapter := epubImgSrcRe.ReplaceAllStringFunc(body, func(match string) string {
		sub := epubImgSrcRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		if name := addAsset(sub[1]); name != "" {
			return `src="images/` + html.EscapeString(name) + `"`
		}
		return match
	})
	if p.CoverImage != "" {
		if strings.HasPrefix(p.CoverImage, "/media/") {
			addAsset(strings.TrimPrefix(p.CoverImage, "/media/"))
		}
	}
	chapter = absolutizeHTML(chapter, cfg.SiteURL)
	chapter = normalizeXHTML(chapter)

	title := html.EscapeString(p.Title)
	subtitle := ""
	if p.Subtitle != "" {
		subtitle = `<p class="deck">` + html.EscapeString(p.Subtitle) + `</p>`
	}
	tags := ""
	if len(p.Tags) > 0 {
		tags = `<p class="tags">` + html.EscapeString(strings.Join(p.Tags, " · ")) + `</p>`
	}
	author := html.EscapeString(p.Author)
	byline := html.EscapeString(fmt.Sprintf("%s · %s · %d min read", p.Author, p.PublishedAt.Format("2 January 2006"), p.ReadingMinutes))

	chapterDoc := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="en">
<head>
<meta charset="utf-8"/>
<title>%s</title>
<link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
<section epub:type="chapter">
<p class="kicker">%s · Dispatch %03d</p>
<h1>%s</h1>
%s
<p class="byline">%s</p>
<div class="body">
%s
</div>
%s
<p class="colophon">ARNΛV · Two editions, one desk. %s</p>
</section>
</body>
</html>`, title, html.EscapeString(blogInfo(p.Blog).Name), p.ID, title, subtitle, byline, chapter, tags, html.EscapeString(cfg.canonical(p.URL)))

	navDoc := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="en">
<head><meta charset="utf-8"/><title>%s</title></head>
<body>
<nav epub:type="toc" id="toc">
<h1>%s</h1>
<ol><li><a href="chapter.xhtml">%s</a></li></ol>
</nav>
</body>
</html>`, title, title, title)

	const styleCSS = `@charset "utf-8";
body { font-family: Georgia, "Times New Roman", serif; color: #241a12; background: #fbf6ed; line-height: 1.65; margin: 0; }
section { max-width: 38em; margin: 0 auto; padding: 1.2em; }
.kicker, .tags, .colophon { font-family: ui-monospace, Menlo, monospace; font-size: 0.72em; letter-spacing: 0.08em; text-transform: uppercase; color: #7d6957; }
h1 { font-size: 2.1em; line-height: 1.05; margin: 0.35em 0 0.3em; letter-spacing: -0.025em; }
h2 { font-size: 1.35em; margin: 1.7em 0 0.4em; line-height: 1.2; }
h3 { font-size: 1.12em; margin: 1.4em 0 0.3em; }
p { margin: 0 0 0.9em; }
.deck { font-style: italic; color: #6c5744; font-size: 1.1em; }
.byline { border-top: 1px solid #d7c7b2; border-bottom: 1px solid #d7c7b2; padding: 0.5em 0; color: #7d6957; font-family: ui-monospace, Menlo, monospace; font-size: 0.78em; }
blockquote { margin: 1.2em 0; padding-left: 1em; border-left: 3px solid #82492a; font-style: italic; }
pre { background: #241a12; color: #f2e7d5; padding: 0.8em; overflow-x: auto; font-size: 0.8em; }
code { font-family: ui-monospace, Menlo, monospace; font-size: 0.88em; }
img { max-width: 100%; height: auto; }
a { color: #82492a; }
hr { border: 0; border-top: 1px solid #d7c7b2; margin: 1.6em 0; }
table { border-collapse: collapse; width: 100%; }
th, td { border-bottom: 1px solid #d7c7b2; padding: 0.35em; text-align: left; }
`

	id := epubUUID(p)
	modified := p.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z")
	manifestImages := strings.Builder{}
	for _, name := range order {
		a := assets[name]
		mediaType := "image/jpeg"
		switch strings.ToLower(filepath.Ext(name)) {
		case ".png":
			mediaType = "image/png"
		case ".webp":
			mediaType = "image/webp"
		case ".gif":
			mediaType = "image/gif"
		case ".svg":
			mediaType = "image/svg+xml"
		}
		fmt.Fprintf(&manifestImages, `<item id="%s" href="images/%s" media-type="%s"/>
`, a.ID, html.EscapeString(name), mediaType)
	}

	opf := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="pub-id" xml:lang="en">
<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
<dc:identifier id="pub-id">urn:uuid:%s</dc:identifier>
<dc:title>%s</dc:title>
<dc:language>en</dc:language>
<dc:creator>%s</dc:creator>
<dc:date>%s</dc:date>
<meta property="dcterms:modified">%s</meta>
<meta property="schema:accessibilityFeature">structuralNavigation</meta>
<meta property="schema:accessibilityFeature">alternativeText</meta>
<meta name="generator" content="ARNΛV"/>
</metadata>
<manifest>
<item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
<item id="chapter" href="chapter.xhtml" media-type="application/xhtml+xml"/>
<item id="css" href="style.css" media-type="text/css"/>
%s</manifest>
<spine><itemref idref="chapter"/></spine>
</package>`, id, title, author, p.PublishedAt.UTC().Format("2006-01-02"), modified, manifestImages.String())

	container := `<?xml version="1.0" encoding="utf-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
<rootfiles><rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	writeStored := func(name string, data []byte) error {
		h := &zip.FileHeader{Name: name, Method: zip.Store}
		h.SetModTime(time.Now())
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	writeDeflated := func(name string, data []byte) error {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetModTime(time.Now())
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	if err := writeStored("mimetype", []byte("application/epub+zip")); err != nil {
		return nil, err
	}
	files := []struct {
		name string
		data []byte
	}{
		{"META-INF/container.xml", []byte(container)},
		{"OEBPS/content.opf", []byte(opf)},
		{"OEBPS/nav.xhtml", []byte(navDoc)},
		{"OEBPS/chapter.xhtml", []byte(chapterDoc)},
		{"OEBPS/style.css", []byte(styleCSS)},
	}
	for _, f := range files {
		if err := writeDeflated(f.name, f.data); err != nil {
			return nil, err
		}
	}
	for _, name := range order {
		a := assets[name]
		if err := writeDeflated("OEBPS/images/"+a.Name, a.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func readMediaFile(cfg Config, name string) ([]byte, error) {
	name = filepath.Base(name)
	if name == "" || name == "." || name == ".." {
		return nil, fs.ErrNotExist
	}
	if data, err := os.ReadFile(filepath.Join(cfg.MediaDir, name)); err == nil {
		return data, nil
	}
	return fs.ReadFile(embeddedFS, "seed/"+name)
}

func normalizeXHTML(s string) string {
	s = strings.ReplaceAll(s, "&nbsp;", "&#160;")
	s = epubVoidImgRe.ReplaceAllString(s, `<img$1/>`)
	s = epubVoidInRe.ReplaceAllString(s, `<input$1/>`)
	s = epubHrRe.ReplaceAllString(s, `<hr/>`)
	s = strings.ReplaceAll(s, "<br>", "<br/>")
	return s
}

func epubUUID(p *Post) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("arnav-blog-epub\x00%d\x00%s\x00%s", p.ID, p.Slug, p.UpdatedAt.UTC().Format(time.RFC3339Nano))))
	sum[6] = (sum[6] & 0x0f) | 0x50
	sum[8] = (sum[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
