package main

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	htmltemplate "html/template"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	pool  *pgxpool.Pool
	redis *redis.Client
	cfg   Config
}

var dbNameRe = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func openStore(ctx context.Context, cfg Config) (*Store, error) {
	if err := ensureDatabase(ctx, cfg.DatabaseURL); err != nil {
		return nil, err
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	poolCfg.MaxConns = 10
	poolCfg.MinConns = 1
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	s := &Store{pool: pool, cfg: cfg}
	if cfg.RedisURL != "" {
		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			slog.Warn("redis url is invalid; continuing without redis", "error", err)
		} else {
			s.redis = redis.NewClient(opt)
			if err := s.redis.Ping(ctx).Err(); err != nil {
				slog.Warn("redis is unavailable; continuing without it", "error", err)
				_ = s.redis.Close()
				s.redis = nil
			}
		}
	}
	if err := s.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func ensureDatabase(ctx context.Context, raw string) error {
	name, err := databaseName(raw)
	if err != nil {
		return err
	}
	if name == "" || name == "postgres" {
		return nil
	}
	if !dbNameRe.MatchString(name) {
		return fmt.Errorf("invalid database name %q", name)
	}

	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	admin := *u
	admin.Path = "/postgres"

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		return fmt.Errorf("connect postgres bootstrap database: %w", err)
	}
	defer conn.Close(context.Background())

	var exists bool
	if err := conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists); err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if exists {
		return nil
	}
	slog.Info("creating blog database", "name", name)
	_, err = conn.Exec(ctx, `CREATE DATABASE "`+name+`"`)
	if err != nil {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}

func (s *Store) Close() {
	if s.redis != nil {
		_ = s.redis.Close()
	}
	s.pool.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS pg_trgm`); err != nil {
		slog.Warn("pg_trgm extension unavailable; trigram fallback disabled", "error", err)
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS posts (
			id BIGSERIAL PRIMARY KEY,
			blog TEXT NOT NULL CHECK (blog IN ('devlog','journal')),
			slug TEXT NOT NULL,
			title TEXT NOT NULL,
			subtitle TEXT NOT NULL DEFAULT '',
			excerpt TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			cover_alt TEXT NOT NULL DEFAULT '',
			body_md TEXT NOT NULL DEFAULT '',
			author TEXT NOT NULL DEFAULT 'Arnav Sharma',
			tags TEXT[] NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
			featured BOOLEAN NOT NULL DEFAULT FALSE,
			published_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			views BIGINT NOT NULL DEFAULT 0,
			reading_minutes INT NOT NULL DEFAULT 1,
			search TSVECTOR,
			UNIQUE (blog, slug)
		)`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS cover_alt TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS subtitle TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS reading_minutes INT NOT NULL DEFAULT 1`,
		`ALTER TABLE posts ADD COLUMN IF NOT EXISTS search TSVECTOR`,
		`CREATE TABLE IF NOT EXISTS post_views (
			post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			visitor_hash TEXT NOT NULL,
			view_date DATE NOT NULL DEFAULT CURRENT_DATE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (post_id, visitor_hash, view_date)
		)`,
		`CREATE INDEX IF NOT EXISTS post_views_date_idx ON post_views (view_date)`,
		`CREATE INDEX IF NOT EXISTS posts_blog_published_idx ON posts (blog, status, published_at DESC NULLS LAST, id DESC)`,
		`CREATE INDEX IF NOT EXISTS posts_status_published_idx ON posts (status, published_at DESC NULLS LAST)`,
		`CREATE OR REPLACE FUNCTION blog_posts_search_refresh() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			NEW.search :=
				setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
				setweight(to_tsvector('english', coalesce(NEW.subtitle, '') || ' ' || coalesce(NEW.excerpt, '') || ' ' || coalesce(array_to_string(NEW.tags, ' '), '')), 'B') ||
				setweight(to_tsvector('english', coalesce(NEW.body_md, '')), 'C');
			RETURN NEW;
		END
		$$`,
		`DROP TRIGGER IF EXISTS blog_posts_search_refresh ON posts`,
		`CREATE TRIGGER blog_posts_search_refresh
			BEFORE INSERT OR UPDATE OF title, subtitle, excerpt, tags, body_md
			ON posts FOR EACH ROW EXECUTE FUNCTION blog_posts_search_refresh()`,
		`CREATE INDEX IF NOT EXISTS posts_search_gin ON posts USING GIN (search)`,
		`UPDATE posts SET search =
			setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
			setweight(to_tsvector('english', coalesce(subtitle, '') || ' ' || coalesce(excerpt, '') || ' ' || coalesce(array_to_string(tags, ' '), '')), 'B') ||
			setweight(to_tsvector('english', coalesce(body_md, '')), 'C')
			WHERE search IS NULL`,
	}
	for i, stmt := range stmts {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	if _, err := s.pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS posts_title_trgm ON posts USING GIN (title gin_trgm_ops)`); err != nil {
		slog.Warn("trigram title index unavailable", "error", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

const cardCols = `id, blog, slug, title, subtitle, excerpt, cover_image, cover_alt, tags,
	COALESCE(published_at, created_at) AS published_at, views, reading_minutes, featured`

func scanCard(row scanner) (PostCard, error) {
	var c PostCard
	err := row.Scan(&c.ID, &c.Blog, &c.Slug, &c.Title, &c.Subtitle, &c.Excerpt,
		&c.CoverImage, &c.CoverAlt, &c.Tags, &c.PublishedAt, &c.Views, &c.ReadingMinutes, &c.Featured)
	if err != nil {
		return c, err
	}
	c.URL = blogInfo(c.Blog).Path + "/" + c.Slug
	return c, nil
}

const postCols = `id, blog, slug, title, subtitle, excerpt, cover_image, cover_alt, tags,
	COALESCE(published_at, created_at) AS published_at, views, reading_minutes, featured,
	body_md, author, status, (published_at IS NOT NULL) AS has_published_at, created_at, updated_at`

func scanPost(row scanner) (Post, error) {
	var p Post
	err := row.Scan(&p.ID, &p.Blog, &p.Slug, &p.Title, &p.Subtitle, &p.Excerpt,
		&p.CoverImage, &p.CoverAlt, &p.Tags, &p.PublishedAt, &p.Views, &p.ReadingMinutes,
		&p.Featured, &p.BodyMD, &p.Author, &p.Status, &p.HasPublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return p, err
	}
	p.URL = blogInfo(p.Blog).Path + "/" + p.Slug
	return p, nil
}

func (s *Store) ListPublished(ctx context.Context, blog string, limit int) ([]PostCard, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := s.pool.Query(ctx, `SELECT `+cardCols+`
		FROM posts
		WHERE status = 'published' AND ($1 = '' OR blog = $1)
		ORDER BY COALESCE(published_at, created_at) DESC, id DESC
		LIMIT $2`, blog, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetPost(ctx context.Context, blog, slug string) (*Post, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+postCols+`
		FROM posts WHERE blog=$1 AND slug=$2 AND status='published'`, blog, slug)
	p, err := scanPost(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetPostByID(ctx context.Context, id int64) (*Post, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+postCols+` FROM posts WHERE id=$1`, id)
	p, err := scanPost(row)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetCardsByIDs(ctx context.Context, ids []int64) ([]PostCard, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT `+cardCols+` FROM posts WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Neighbors(ctx context.Context, p *Post) (*PostCard, *PostCard, error) {
	prevRow := s.pool.QueryRow(ctx, `SELECT `+cardCols+` FROM posts
		WHERE blog=$1 AND status='published' AND id<>$2
		AND COALESCE(published_at, created_at) < $3
		ORDER BY COALESCE(published_at, created_at) DESC, id DESC LIMIT 1`, p.Blog, p.ID, p.PublishedAt)
	prev, err := scanCard(prevRow)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	var prevPtr *PostCard
	if err == nil {
		prevPtr = &prev
	}

	nextRow := s.pool.QueryRow(ctx, `SELECT `+cardCols+` FROM posts
		WHERE blog=$1 AND status='published' AND id<>$2
		AND COALESCE(published_at, created_at) > $3
		ORDER BY COALESCE(published_at, created_at) ASC, id ASC LIMIT 1`, p.Blog, p.ID, p.PublishedAt)
	next, err := scanCard(nextRow)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, err
	}
	var nextPtr *PostCard
	if err == nil {
		nextPtr = &next
	}
	return prevPtr, nextPtr, nil
}

func (s *Store) Related(ctx context.Context, p *Post, limit int) ([]PostCard, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+cardCols+` FROM posts
		WHERE blog=$1 AND status='published' AND id<>$2 AND tags && $3::text[]
		ORDER BY COALESCE(published_at, created_at) DESC LIMIT $4`, p.Blog, p.ID, p.Tags, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return s.ListPublished(ctx, p.Blog, limit+1)
	}
	return out, rows.Err()
}

func (s *Store) Trending(ctx context.Context, blog string, limit int) ([]PostCard, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+cardCols+` FROM posts
		WHERE status='published' AND ($1='' OR blog=$1)
		ORDER BY views DESC, COALESCE(published_at, created_at) DESC LIMIT $2`, blog, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PostCard
	for rows.Next() {
		c, err := scanCard(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) Tags(ctx context.Context, blog string) ([]TagCount, error) {
	rows, err := s.pool.Query(ctx, `SELECT tag, count(*) FROM (
		SELECT unnest(tags) AS tag FROM posts WHERE status='published' AND ($1='' OR blog=$1)
	) t GROUP BY tag ORDER BY count(*) DESC, tag ASC LIMIT 14`, blog)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TagCount
	for rows.Next() {
		var t TagCount
		if err := rows.Scan(&t.Name, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CountPublished(ctx context.Context, blog string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM posts WHERE status='published' AND ($1='' OR blog=$1)`, blog).Scan(&n)
	return n, err
}

func (s *Store) AllPublished(ctx context.Context, blog string) ([]PostCard, error) {
	return s.ListPublished(ctx, blog, 500)
}

func (s *Store) Search(ctx context.Context, q, blog string, limit int) ([]SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 40
	}

	cacheKey := s.searchCacheKey(q, blog, limit)
	if cacheKey != "" {
		if raw, err := s.redis.Get(ctx, cacheKey).Result(); err == nil {
			var cached []SearchResult
			if json.Unmarshal([]byte(raw), &cached) == nil {
				for i := range cached {
					cached[i].SnippetHTML = highlightSnippet(cached[i].Snippet)
				}
				return cached, nil
			}
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+cardCols+`,
			ts_rank_cd(search, websearch_to_tsquery('english', $1)) AS rank,
			ts_headline('english', COALESCE(NULLIF(body_md,''), excerpt),
				websearch_to_tsquery('english', $1),
				'StartSel=[[[, StopSel=]]], MaxWords=38, MinWords=18, ShortWord=3, MaxFragments=2, FragmentDelimiter= … ') AS snippet
		FROM posts
		WHERE status='published' AND ($2='' OR blog=$2)
			AND search @@ websearch_to_tsquery('english', $1)
		ORDER BY rank DESC, COALESCE(published_at, created_at) DESC
		LIMIT $3`, q, blog, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SearchResult
	for rows.Next() {
		var r SearchResult
		err := rows.Scan(&r.ID, &r.Blog, &r.Slug, &r.Title, &r.Subtitle, &r.Excerpt,
			&r.CoverImage, &r.CoverAlt, &r.Tags, &r.PublishedAt, &r.Views, &r.ReadingMinutes,
			&r.Featured, &r.Rank, &r.Snippet)
		if err != nil {
			return nil, err
		}
		r.URL = blogInfo(r.Blog).Path + "/" + r.Slug
		r.SnippetHTML = highlightSnippet(r.Snippet)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(out) == 0 {
		like := "%" + q + "%"
		fRows, err := s.pool.Query(ctx, `SELECT `+cardCols+`, 0::float8 AS rank,
			COALESCE(NULLIF(excerpt,''), left(body_md, 260)) AS snippet
			FROM posts WHERE status='published' AND ($2='' OR blog=$2)
			AND (title ILIKE $1 OR subtitle ILIKE $1 OR excerpt ILIKE $1 OR body_md ILIKE $1 OR array_to_string(tags,' ') ILIKE $1)
			ORDER BY COALESCE(published_at, created_at) DESC LIMIT $3`, like, blog, limit)
		if err != nil {
			return nil, err
		}
		defer fRows.Close()
		for fRows.Next() {
			var r SearchResult
			err := fRows.Scan(&r.ID, &r.Blog, &r.Slug, &r.Title, &r.Subtitle, &r.Excerpt,
				&r.CoverImage, &r.CoverAlt, &r.Tags, &r.PublishedAt, &r.Views, &r.ReadingMinutes,
				&r.Featured, &r.Rank, &r.Snippet)
			if err != nil {
				return nil, err
			}
			r.URL = blogInfo(r.Blog).Path + "/" + r.Slug
			r.SnippetHTML = highlightSnippet(r.Snippet)
			out = append(out, r)
		}
		if err := fRows.Err(); err != nil {
			return nil, err
		}
	}

	if cacheKey != "" && len(out) > 0 {
		if raw, err := json.Marshal(out); err == nil {
			_ = s.redis.Set(ctx, cacheKey, raw, 90*time.Second).Err()
		}
	}
	return out, nil
}

func (s *Store) searchCacheKey(q, blog string, limit int) string {
	if s.redis == nil {
		return ""
	}
	sum := sha1.Sum([]byte(strings.ToLower(strings.TrimSpace(q)) + "\x00" + blog))
	return "blog:search:" + hex.EncodeToString(sum[:]) + fmt.Sprintf(":%d", limit)
}

func highlightSnippet(s string) htmltemplate.HTML {
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "[[[", "<mark>")
	s = strings.ReplaceAll(s, "]]]", "</mark>")
	return htmltemplate.HTML(s)
}

func (s *Store) ClearSearchCache(ctx context.Context) {
	if s.redis == nil {
		return
	}
	iter := s.redis.Scan(ctx, 0, "blog:search:*", 100).Iterator()
	for iter.Next(ctx) {
		_ = s.redis.Del(ctx, iter.Val()).Err()
	}
}

func (s *Store) IncrementView(ctx context.Context, postID int64, visitor string, day time.Time) (int64, bool, error) {
	dayKey := day.Format("2006-01-02")

	// Redis is only a fast short-circuit. The database remains the source of truth.
	if s.redis != nil {
		key := fmt.Sprintf("blog:seen:%d:%s:%s", postID, visitor, dayKey)
		ok, err := s.redis.SetNX(ctx, key, "1", 26*time.Hour).Result()
		if err == nil && !ok {
			var views int64
			_ = s.pool.QueryRow(ctx, `SELECT views FROM posts WHERE id=$1`, postID).Scan(&views)
			return views, false, nil
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback(context.Background())

	ct, err := tx.Exec(ctx, `INSERT INTO post_views (post_id, visitor_hash, view_date)
		VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, postID, visitor, dayKey)
	if err != nil {
		return 0, false, err
	}
	counted := ct.RowsAffected() > 0
	if counted {
		if _, err := tx.Exec(ctx, `UPDATE posts SET views = views + 1 WHERE id=$1`, postID); err != nil {
			return 0, false, err
		}
	}
	var views int64
	if err := tx.QueryRow(ctx, `SELECT views FROM posts WHERE id=$1`, postID).Scan(&views); err != nil {
		return 0, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, false, err
	}
	return views, counted, nil
}

func (s *Store) AdminStats(ctx context.Context) (AdminStats, error) {
	var st AdminStats
	rows, err := s.pool.Query(ctx, `SELECT blog, count(*),
		count(*) FILTER (WHERE status='published'),
		count(*) FILTER (WHERE status='draft'),
		COALESCE(sum(views),0)
		FROM posts GROUP BY blog`)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	byBlog := map[string]AdminBlogStat{}
	for rows.Next() {
		var b AdminBlogStat
		if err := rows.Scan(&b.Key, &b.Posts, &b.Published, &b.Drafts, &b.Views); err != nil {
			return st, err
		}
		b.Name = blogInfo(b.Key).Name
		byBlog[b.Key] = b
		st.TotalPosts += b.Posts
		st.Published += b.Published
		st.Drafts += b.Drafts
		st.TotalViews += b.Views
	}
	for _, key := range blogOrder {
		b, ok := byBlog[key]
		if !ok {
			b = AdminBlogStat{Key: key, Name: blogInfo(key).Name}
		}
		st.Blogs = append(st.Blogs, b)
	}

	counts := map[string]int{}
	dRows, err := s.pool.Query(ctx, `SELECT view_date, count(*) FROM post_views
		WHERE view_date >= CURRENT_DATE - 13 GROUP BY view_date ORDER BY view_date`)
	if err != nil {
		return st, err
	}
	defer dRows.Close()
	for dRows.Next() {
		var d time.Time
		var n int
		if err := dRows.Scan(&d, &n); err != nil {
			return st, err
		}
		counts[d.Format("2006-01-02")] = n
	}
	today := time.Now().In(s.cfg.Location)
	for i := 13; i >= 0; i-- {
		day := today.AddDate(0, 0, -i)
		n := counts[day.Format("2006-01-02")]
		if n > st.MaxDay {
			st.MaxDay = n
		}
		st.Days = append(st.Days, DayViews{Day: day, Label: day.Format("02 Jan"), Count: n})
	}
	for i := range st.Days {
		if st.MaxDay > 0 {
			st.Days[i].Pct = int(float64(st.Days[i].Count)/float64(st.MaxDay)*100 + 0.5)
		}
		if st.Days[i].Pct < 3 && st.Days[i].Count > 0 {
			st.Days[i].Pct = 3
		}
	}
	return st, nil
}

func (s *Store) AdminPosts(ctx context.Context, q, blog string) ([]AdminPostRow, error) {
	q = strings.TrimSpace(q)
	rows, err := s.pool.Query(ctx, `SELECT id, blog, slug, title, status, views, updated_at, published_at
		FROM posts
		WHERE ($1='' OR blog=$1)
		AND ($2='' OR title ILIKE '%'||$2||'%' OR body_md ILIKE '%'||$2||'%' OR array_to_string(tags,' ') ILIKE '%'||$2||'%')
		ORDER BY updated_at DESC LIMIT 100`, blog, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminPostRow
	for rows.Next() {
		var r AdminPostRow
		var pub *time.Time
		if err := rows.Scan(&r.ID, &r.Blog, &r.Slug, &r.Title, &r.Status, &r.Views, &r.UpdatedAt, &pub); err != nil {
			return nil, err
		}
		r.BlogName = blogInfo(r.Blog).Name
		r.PublishedAt = pub
		r.URL = blogInfo(r.Blog).Path + "/" + r.Slug
		out = append(out, r)
	}
	return out, rows.Err()
}

type PostInput struct {
	Blog       string
	Slug       string
	Title      string
	Subtitle   string
	Excerpt    string
	CoverImage string
	CoverAlt   string
	BodyMD     string
	Author     string
	Tags       []string
	Status     string
	Featured   bool
	Published  *time.Time
}

func (s *Store) CreatePost(ctx context.Context, in PostInput, author string) (int64, error) {
	if in.Author != "" {
		author = in.Author
	}
	if author == "" {
		author = "Arnav Sharma"
	}
	var id int64
	err := s.pool.QueryRow(ctx, `INSERT INTO posts
		(blog, slug, title, subtitle, excerpt, cover_image, cover_alt, body_md, author, tags, status, featured, published_at, reading_minutes, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW())
		RETURNING id`,
		in.Blog, in.Slug, in.Title, in.Subtitle, in.Excerpt, in.CoverImage, in.CoverAlt,
		in.BodyMD, author, in.Tags, in.Status, in.Featured, in.Published, readingMinutes(in.BodyMD)).Scan(&id)
	return id, err
}

func (s *Store) UpdatePost(ctx context.Context, id int64, in PostInput) error {
	author := in.Author
	if author == "" {
		author = "Arnav Sharma"
	}
	_, err := s.pool.Exec(ctx, `UPDATE posts SET
		blog=$2, slug=$3, title=$4, subtitle=$5, excerpt=$6, cover_image=$7, cover_alt=$8,
		body_md=$9, author=$10, tags=$11, status=$12, featured=$13, published_at=$14,
		reading_minutes=$15, updated_at=NOW()
		WHERE id=$1`,
		id, in.Blog, in.Slug, in.Title, in.Subtitle, in.Excerpt, in.CoverImage, in.CoverAlt,
		in.BodyMD, author, in.Tags, in.Status, in.Featured, in.Published, readingMinutes(in.BodyMD))
	return err
}

func (s *Store) DeletePost(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM posts WHERE id=$1`, id)
	return err
}

func (s *Store) SlugTaken(ctx context.Context, blog, slug string, exceptID int64) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM posts WHERE blog=$1 AND slug=$2 AND id<>$3`, blog, slug, exceptID).Scan(&n)
	return n > 0, err
}

func (s *Store) CountAll(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM posts`).Scan(&n)
	return n, err
}
