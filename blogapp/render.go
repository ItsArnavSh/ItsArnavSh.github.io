package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	htmltemplate "html/template"
	"io/fs"
	"log/slog"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	texttemplate "text/template"
	"time"
	"unicode"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

//go:embed templates/*.html templates/*.xml assets seed
var embeddedFS embed.FS

type mathItem struct {
	Token   string
	TeX     string
	Display bool
}

type Renderer struct {
	cfg   Config
	dev   bool
	mu    sync.RWMutex
	cache map[string]*htmltemplate.Template
	xml   map[string]*texttemplate.Template
	funcs htmltemplate.FuncMap
	md    goldmark.Markdown
}

func NewRenderer(cfg Config) *Renderer {
	rd := &Renderer{cfg: cfg, dev: cfg.Dev, cache: map[string]*htmltemplate.Template{}, xml: map[string]*texttemplate.Template{}}
	rd.funcs = rd.templateFuncs()
	rd.md = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			extension.Linkify,
			highlighting.NewHighlighting(
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
				highlighting.WithWrapperRenderer(func(w util.BufWriter, ctx highlighting.CodeBlockContext, entering bool) {
					lang := ""
					if l, ok := ctx.Language(); ok {
						lang = string(l)
					}
					if entering {
						w.WriteString(`<div class="code-block" data-lang="` + html.EscapeString(lang) + `"><pre class="chroma"><code>`)
					} else {
						w.WriteString(`</code></pre><button class="code-copy" type="button" aria-label="Copy code">Copy</button></div>`)
					}
				}),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(&figureRenderer{}, 100)),
		),
	)
	return rd
}

type figureRenderer struct{}

func (r *figureRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

func (r *figureRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.Image)
	alt := string(n.Text(source))
	title := string(n.Title)
	dest := string(n.Destination)
	_, _ = w.WriteString(`<span class="md-figure">`)
	_, _ = w.WriteString(`<img src="` + html.EscapeString(dest) + `" alt="` + html.EscapeString(alt) + `" loading="lazy" decoding="async"`)
	if title != "" {
		_, _ = w.WriteString(` title="` + html.EscapeString(title) + `"`)
	}
	_, _ = w.WriteString(`>`)
	if title != "" {
		_, _ = w.WriteString(`<span class="md-caption">` + html.EscapeString(title) + `</span>`)
	}
	_, _ = w.WriteString(`</span>`)
	return ast.WalkSkipChildren, nil
}

func (rd *Renderer) RenderMarkdown(md string) (string, []TOCItem) {
	protected, mathItems := extractMath(md)
	source := []byte(protected)
	doc := rd.md.Parser().Parse(text.NewReader(source))

	var toc []TOCItem
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok && h.Level >= 2 && h.Level <= 3 {
			id := ""
			if v, ok := h.AttributeString("id"); ok {
				switch t := v.(type) {
				case []byte:
					id = string(t)
				case string:
					id = t
				}
			}
			label := strings.TrimSpace(string(h.Text(source)))
			if id != "" && label != "" {
				toc = append(toc, TOCItem{Level: h.Level, ID: id, Text: label})
			}
		}
		return ast.WalkContinue, nil
	})

	var buf bytes.Buffer
	if err := rd.md.Renderer().Render(&buf, source, doc); err != nil {
		slog.Error("markdown render", "error", err)
		return "<p>Unable to render this dispatch.</p>", nil
	}
	out := buf.String()
	for _, m := range mathItems {
		class := "math math-inline"
		open, close := `\(`, `\)`
		if m.Display {
			class = "math math-display"
			open, close = `\[`, `\]`
		}
		repl := `<span class="` + class + `" data-tex="` + html.EscapeString(m.TeX) + `">` + html.EscapeString(open+m.TeX+close) + `</span>`
		out = strings.ReplaceAll(out, m.Token, repl)
	}
	return out, toc
}

func (rd *Renderer) RenderPreview(md string) (string, int, error) {
	body, _ := rd.RenderMarkdown(md)
	return body, readingMinutes(md), nil
}

// extractMath protects math spans from Goldmark and replaces them with tokens.
// It deliberately skips fenced code, inline code and escaped dollars.
func extractMath(input string) (string, []mathItem) {
	var out strings.Builder
	var items []mathItem
	n := len(input)
	i := 0
	lineStart := true
	fenceChar := byte(0)
	fenceLen := 0

	makeToken := func(display bool) string {
		idx := len(items)
		kind := "I"
		if display {
			kind = "B"
		}
		token := fmt.Sprintf("ZKMATH%s%06dZ", kind, idx)
		return token
	}

	for i < n {
		if lineStart {
			// Detect fenced code blocks.
			j := i
			for j < n && j-i < 3 && input[j] == ' ' {
				j++
			}
			if j < n && (input[j] == '`' || input[j] == '~') {
				c := input[j]
				k := j
				for k < n && input[k] == c {
					k++
				}
				run := k - j
				if fenceChar == 0 && run >= 3 {
					fenceChar, fenceLen = c, run
					out.WriteString(input[i:k])
					i = k
					lineStart = false
					continue
				}
				if fenceChar == c && run >= fenceLen {
					fenceChar, fenceLen = 0, 0
					out.WriteString(input[i:k])
					i = k
					lineStart = false
					continue
				}
			}
			lineStart = false
		}

		ch := input[i]
		if ch == '\n' {
			out.WriteByte(ch)
			i++
			lineStart = true
			continue
		}
		if fenceChar != 0 {
			out.WriteByte(ch)
			i++
			continue
		}

		// Escaped character.
		if ch == '\\' && i+1 < n {
			out.WriteString(input[i : i+2])
			i += 2
			continue
		}

		// Inline code span: copy verbatim.
		if ch == '`' {
			j := i
			for j < n && input[j] == '`' {
				j++
			}
			run := j - i
			closeAt := -1
			for k := j; k < n; k++ {
				if input[k] != '`' {
					continue
				}
				e := k
				for e < n && input[e] == '`' {
					e++
				}
				if e-k == run {
					closeAt = e
					break
				}
				k = e - 1
			}
			if closeAt != -1 {
				out.WriteString(input[i:closeAt])
				i = closeAt
				continue
			}
			out.WriteString(input[i:j])
			i = j
			continue
		}

		// \( ... \)
		if ch == '\\' && i+1 < n && input[i+1] == '(' {
			if end := findUnescaped(input, `\)`, i+2); end != -1 {
				tex := input[i+2 : end]
				token := makeToken(false)
				items = append(items, mathItem{Token: token, TeX: tex, Display: false})
				out.WriteString(token)
				i = end + 2
				continue
			}
		}
		// \[ ... \]
		if ch == '\\' && i+1 < n && input[i+1] == '[' {
			if end := findUnescaped(input, `\]`, i+2); end != -1 {
				tex := input[i+2 : end]
				token := makeToken(true)
				items = append(items, mathItem{Token: token, TeX: tex, Display: true})
				out.WriteString(token)
				i = end + 2
				continue
			}
		}

		if ch == '$' {
			if i+1 < n && input[i+1] == '$' {
				if end := findUnescaped(input, "$$", i+2); end != -1 {
					tex := input[i+2 : end]
					token := makeToken(true)
					items = append(items, mathItem{Token: token, TeX: tex, Display: true})
					out.WriteString(token)
					i = end + 2
					continue
				}
			} else if i+1 < n && !unicode.IsSpace(rune(input[i+1])) {
				if end := findInlineDollarClose(input, i+1); end != -1 {
					tex := input[i+1 : end]
					token := makeToken(false)
					items = append(items, mathItem{Token: token, TeX: tex, Display: false})
					out.WriteString(token)
					i = end + 1
					continue
				}
			}
		}

		out.WriteByte(ch)
		i++
	}
	return out.String(), items
}

func findUnescaped(s, needle string, from int) int {
	for i := from; i <= len(s)-len(needle); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if strings.HasPrefix(s[i:], needle) {
			return i
		}
	}
	return -1
}

func findInlineDollarClose(s string, from int) int {
	for i := from; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] != '$' {
			continue
		}
		if i == from {
			continue
		}
		if unicode.IsSpace(rune(s[i-1])) {
			continue
		}
		if i+1 < len(s) && unicode.IsDigit(rune(s[i+1])) {
			continue
		}
		return i
	}
	return -1
}

func (rd *Renderer) templateFS() fs.FS {
	if rd.dev {
		return os.DirFS(".")
	}
	return embeddedFS
}

func (rd *Renderer) page(name string) (*htmltemplate.Template, error) {
	key := "page:" + name
	if !rd.dev {
		rd.mu.RLock()
		t, ok := rd.cache[key]
		rd.mu.RUnlock()
		if ok {
			return t, nil
		}
	}
	files := []string{"templates/public_layout.html", "templates/partials.html", "templates/" + name + ".html"}
	t := htmltemplate.New("layout").Funcs(rd.funcs)
	var err error
	if rd.dev {
		for _, f := range files {
			b, e := os.ReadFile(f)
			if e != nil {
				return nil, e
			}
			if _, e = t.New(filepath.Base(f)).Parse(string(b)); e != nil {
				return nil, e
			}
		}
	} else {
		t, err = t.ParseFS(rd.templateFS(), files...)
		if err != nil {
			return nil, err
		}
	}
	if !rd.dev {
		rd.mu.Lock()
		rd.cache[key] = t
		rd.mu.Unlock()
	}
	return t, nil
}

func (rd *Renderer) adminPage(name string) (*htmltemplate.Template, error) {
	key := "admin:" + name
	if !rd.dev {
		rd.mu.RLock()
		t, ok := rd.cache[key]
		rd.mu.RUnlock()
		if ok {
			return t, nil
		}
	}
	files := []string{"templates/admin_layout.html", "templates/partials.html", "templates/" + name + ".html"}
	t := htmltemplate.New("admin_layout").Funcs(rd.funcs)
	var err error
	if rd.dev {
		for _, f := range files {
			b, e := os.ReadFile(f)
			if e != nil {
				return nil, e
			}
			if _, e = t.New(filepath.Base(f)).Parse(string(b)); e != nil {
				return nil, e
			}
		}
	} else {
		t, err = t.ParseFS(rd.templateFS(), files...)
		if err != nil {
			return nil, err
		}
	}
	if !rd.dev {
		rd.mu.Lock()
		rd.cache[key] = t
		rd.mu.Unlock()
	}
	return t, nil
}

func (rd *Renderer) RenderPage(w *bytes.Buffer, name string, data any) error {
	t, err := rd.page(name)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, "layout", data)
}

func (rd *Renderer) RenderAdmin(w *bytes.Buffer, name string, data any) error {
	t, err := rd.adminPage(name)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, "admin_layout", data)
}

func (rd *Renderer) RenderStandalone(w *bytes.Buffer, name, templateName string, data any) error {
	key := "standalone:" + name
	if !rd.dev {
		rd.mu.RLock()
		t, ok := rd.cache[key]
		rd.mu.RUnlock()
		if ok {
			return t.ExecuteTemplate(w, templateName, data)
		}
	}
	files := []string{"templates/" + name + ".html"}
	t := htmltemplate.New(name).Funcs(rd.funcs)
	var err error
	if rd.dev {
		b, e := os.ReadFile(files[0])
		if e != nil {
			return e
		}
		t, err = t.Parse(string(b))
		if err != nil {
			return err
		}
	} else {
		t, err = t.ParseFS(rd.templateFS(), files...)
		if err != nil {
			return err
		}
	}
	if !rd.dev {
		rd.mu.Lock()
		rd.cache[key] = t
		rd.mu.Unlock()
	}
	return t.ExecuteTemplate(w, templateName, data)
}

func (rd *Renderer) RenderXML(w *bytes.Buffer, name string, data any) error {
	base := strings.TrimSuffix(name, ".xml")
	templateName := base + ".xml"
	key := "xml:" + templateName
	if !rd.dev {
		rd.mu.RLock()
		t, ok := rd.xml[key]
		rd.mu.RUnlock()
		if ok {
			return t.ExecuteTemplate(w, templateName, data)
		}
	}
	files := []string{"templates/" + base + ".xml"}
	t := texttemplate.New(templateName).Funcs(rd.xmlFuncs())
	var err error
	if rd.dev {
		b, e := os.ReadFile(files[0])
		if e != nil {
			return e
		}
		t, err = t.Parse(string(b))
		if err != nil {
			return err
		}
	} else {
		t, err = t.ParseFS(rd.templateFS(), files...)
		if err != nil {
			return err
		}
	}
	if !rd.dev {
		rd.mu.Lock()
		rd.xml[key] = t
		rd.mu.Unlock()
	}
	return t.ExecuteTemplate(w, templateName, data)
}

func (rd *Renderer) templateFuncs() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"date":      func(t time.Time) string { return t.In(rd.cfg.Location).Format("02 Jan 2006") },
		"dateShort": func(t time.Time) string { return t.In(rd.cfg.Location).Format("2 Jan") },
		"dateLong":  func(t time.Time) string { return t.In(rd.cfg.Location).Format("Monday, 2 January 2006") },
		"dateFull":  func(t time.Time) string { return t.In(rd.cfg.Location).Format("2 January 2006") },
		"dateISO":   func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
		"dateInput": func(t time.Time) string { return t.In(rd.cfg.Location).Format("2006-01-02T15:04") },
		"datePtr":   datePtr(rd.cfg.Location),
		"year":      func(t time.Time) int { return t.In(rd.cfg.Location).Year() },
		"nowHour":   func(t time.Time) int { return t.In(rd.cfg.Location).Hour() },
		"views":     formatViews,
		"viewWord": func(n any) string {
			switch v := n.(type) {
			case int:
				if v == 1 {
					return "view"
				}
			case int64:
				if v == 1 {
					return "view"
				}
			}
			return "views"
		},
		"reading": func(n int) string {
			if n <= 1 {
				return "1 min read"
			}
			return fmt.Sprintf("%d min read", n)
		},
		"safeHTML":  func(s string) htmltemplate.HTML { return htmltemplate.HTML(s) },
		"safeAttr":  func(s string) htmltemplate.HTMLAttr { return htmltemplate.HTMLAttr(s) },
		"abs":       rd.cfg.absolute,
		"canonical": rd.cfg.canonical,
		"json":      func(v any) htmltemplate.JS { b, _ := json.Marshal(v); return htmltemplate.JS(b) },
		"add":       func(a, b int) int { return a + b },
		"sub":       func(a, b int) int { return a - b },
		"mul":       func(a, b int) int { return a * b },
		"pct": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return int(math.Round(float64(a) / float64(b) * 100))
		},
		"truncate": truncate,
		"initials": initials,
		"lower":    strings.ToLower,
		"tagURL":   func(tag string) string { return "/search?q=" + urlQuery(tag) },
		"blog":     blogInfo,
		"blogs":    blogs,
		"join":     strings.Join,
		"hasTag": func(tags []string, tag string) bool {
			for _, t := range tags {
				if t == tag {
					return true
				}
			}
			return false
		},
		"dict": dict,
		"default": func(v, fallback string) string {
			if strings.TrimSpace(v) == "" {
				return fallback
			}
			return v
		},
		"shareX": func(title, url string) string {
			return "https://twitter.com/intent/tweet?text=" + urlQuery(title) + "&url=" + urlQuery(url)
		},
		"shareLinkedIn": func(url string) string {
			return "https://www.linkedin.com/sharing/share-offsite/?url=" + urlQuery(url)
		},
		"shareWhatsApp": func(title, url string) string {
			return "https://wa.me/?text=" + urlQuery(title+" "+url)
		},
		"shareReddit": func(title, url string) string {
			return "https://www.reddit.com/submit?url=" + urlQuery(url) + "&title=" + urlQuery(title)
		},
		"shareTelegram": func(title, url string) string {
			return "https://t.me/share/url?url=" + urlQuery(url) + "&text=" + urlQuery(title)
		},
		"shareHN": func(title, url string) string {
			return "https://news.ycombinator.com/submitlink?u=" + urlQuery(url) + "&t=" + urlQuery(title)
		},
		"shareMail": func(title, url string) string {
			return "mailto:?subject=" + urlQuery(title) + "&body=" + urlQuery(url)
		},
	}
}

func (rd *Renderer) xmlFuncs() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"dateRFC822": func(t time.Time) string { return t.Format(time.RFC1123Z) },
		"dateISO":    func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
		"abs":        rd.cfg.absolute,
		"safeXML":    func(s string) string { return s },
		"xmlEscape":  func(s string) string { return html.EscapeString(s) },
		"views":      formatViews,
	}
}

func datePtr(loc *time.Location) func(any) string {
	return func(v any) string {
		switch t := v.(type) {
		case time.Time:
			if t.IsZero() {
				return "—"
			}
			return t.In(loc).Format("02 Jan 2006")
		case *time.Time:
			if t == nil || t.IsZero() {
				return "—"
			}
			return t.In(loc).Format("02 Jan 2006")
		default:
			return "—"
		}
	}
}

func urlQuery(s string) string {
	return url.QueryEscape(s)
}

func formatViews(n int64) string {
	if n < 1000 {
		return strconv.FormatInt(n, 10)
	}
	if n < 1_000_000 {
		v := float64(n) / 1000
		if v < 10 {
			return strconv.FormatFloat(v, 'f', 1, 64) + "k"
		}
		return strconv.FormatFloat(v, 'f', 0, 64) + "k"
	}
	v := float64(n) / 1_000_000
	return strconv.FormatFloat(v, 'f', 1, 64) + "m"
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	cut := string(r[:n])
	if idx := strings.LastIndex(cut, " "); idx > n/2 {
		cut = cut[:idx]
	}
	return strings.TrimRight(cut, " ,.;:") + "…"
}

func initials(name string) string {
	var b strings.Builder
	for _, f := range strings.Fields(name) {
		if len(f) > 0 {
			b.WriteByte(f[0])
		}
	}
	return strings.ToUpper(b.String())
}

func dict(values ...any) map[string]any {
	m := map[string]any{}
	for i := 0; i < len(values)-1; i += 2 {
		if k, ok := values[i].(string); ok {
			m[k] = values[i+1]
		}
	}
	return m
}

func readingMinutes(md string) int {
	words := len(strings.Fields(stripMarkdown(md)))
	if words == 0 {
		return 1
	}
	return int(math.Ceil(float64(words) / 220.0))
}

var (
	reCodeFence = regexp.MustCompile("(?s)```.*?```")
	reImage     = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	reLink      = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	reHeading   = regexp.MustCompile(`(?m)^#{1,6}\s*`)
	reTags      = regexp.MustCompile(`<[^>]+>`)
	reMarks     = regexp.MustCompile(`[*_~` + "`" + `>#]+`)
)

func stripMarkdown(s string) string {
	s = reCodeFence.ReplaceAllString(s, " ")
	s = reImage.ReplaceAllString(s, "$1")
	s = reLink.ReplaceAllString(s, "$1")
	s = reHeading.ReplaceAllString(s, "")
	s = reTags.ReplaceAllString(s, " ")
	s = reMarks.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	return strings.Join(strings.Fields(s), " ")
}

func excerptFrom(md string, n int) string {
	return truncate(stripMarkdown(md), n)
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == ' ' || r == '-' || r == '_' || r == '/' || r == '.':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		default:
			if r > unicode.MaxASCII {
				// Keep unicode letters readable in slugs.
				if unicode.IsLetter(r) || unicode.IsDigit(r) {
					b.WriteRune(unicode.ToLower(r))
					lastDash = false
				}
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 90 {
		out = strings.Trim(out[:90], "-")
	}
	if out == "" {
		out = "dispatch"
	}
	return out
}

func parseTags(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == '\n' })
	var out []string
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "#")
		if p == "" || seen[strings.ToLower(p)] {
			continue
		}
		seen[strings.ToLower(p)] = true
		out = append(out, p)
		if len(out) >= 8 {
			break
		}
	}
	return out
}
