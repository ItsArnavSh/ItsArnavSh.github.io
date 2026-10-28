package main

import (
	"html/template"
	"time"
)

type BlogInfo struct {
	Key         string
	Name        string
	Masthead    string
	MastheadA   string
	MastheadB   string
	Tagline     string
	Description string
	Theme       string
	Path        string
	Short       string
}

var blogOrder = []string{"devlog", "journal"}

var blogCatalog = map[string]BlogInfo{
	"devlog": {
		Key:         "devlog",
		Name:        "Devlog",
		Masthead:    "DEVLOG",
		MastheadA:   "DEV",
		MastheadB:   "LOG",
		Tagline:     "Notes from the machine room",
		Description: "Build notes, systems, code and the occasional post-mortem. The technical edition of Arnav's desk.",
		Theme:       "dark",
		Path:        "/devlog",
		Short:       "Build notes",
	},
	"journal": {
		Key:         "journal",
		Name:        "Journal",
		Masthead:    "JOURNAL",
		MastheadA:   "JOUR",
		MastheadB:   "NAL",
		Tagline:     "A slower kind of internet",
		Description: "Field notes, photographs and small essays. The personal edition of Arnav's desk.",
		Theme:       "light",
		Path:        "/journal",
		Short:       "Field notes",
	},
}

func blogs() []BlogInfo {
	out := make([]BlogInfo, 0, len(blogOrder))
	for _, key := range blogOrder {
		out = append(out, blogCatalog[key])
	}
	return out
}

func blogInfo(key string) BlogInfo {
	if b, ok := blogCatalog[key]; ok {
		return b
	}
	return blogCatalog["devlog"]
}

type PostCard struct {
	ID             int64
	Blog           string
	Slug           string
	Title          string
	Subtitle       string
	Excerpt        string
	CoverImage     string
	CoverAlt       string
	Tags           []string
	PublishedAt    time.Time
	Views          int64
	ReadingMinutes int
	URL            string
	Featured       bool
}

type TOCItem struct {
	Level int
	ID    string
	Text  string
}

type Post struct {
	PostCard
	BodyMD         string
	BodyHTML       template.HTML
	Author         string
	Status         string
	HasPublishedAt bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	TOC            []TOCItem
	WordCount      int
}

type TagCount struct {
	Name  string
	Count int
}

type SearchResult struct {
	PostCard
	Rank        float64
	Snippet     string
	SnippetHTML template.HTML
}

type SiteInfo struct {
	URL       string
	MainURL   string
	Title     string
	AdminUser string
	Year      int
}

type PageData struct {
	Site          SiteInfo
	Blog          BlogInfo
	Blogs         []BlogInfo
	Path          string
	Title         string
	Description   string
	Canonical     string
	OGImage       string
	OGType        string
	Now           time.Time
	Issue         int
	Posts         []PostCard
	Lead          *PostCard
	Secondary     []PostCard
	More          []PostCard
	Trending      []PostCard
	Tags          []TagCount
	Post          *Post
	Prev          *PostCard
	Next          *PostCard
	Related       []PostCard
	SearchQuery   string
	SearchBlog    string
	SearchResults []SearchResult
	SearchCount   int
	TotalPosts    int
	Flashes       []string
	DevMode       bool
	PrintMode     bool
	NoIndex       bool
	JSONLD        template.JS
}

type AdminBlogStat struct {
	Key       string
	Name      string
	Posts     int
	Published int
	Drafts    int
	Views     int64
}

type DayViews struct {
	Day   time.Time
	Label string
	Count int
	Pct   int
}

type AdminStats struct {
	Blogs      []AdminBlogStat
	TotalPosts int
	Published  int
	Drafts     int
	TotalViews int64
	Days       []DayViews
	MaxDay     int
}

type AdminPostRow struct {
	ID          int64
	Blog        string
	BlogName    string
	Slug        string
	Title       string
	Status      string
	Views       int64
	UpdatedAt   time.Time
	PublishedAt *time.Time
	URL         string
}

type MediaItem struct {
	Name string    `json:"name"`
	URL  string    `json:"url"`
	Size int64     `json:"size"`
	Mod  time.Time `json:"mod"`
}

type AdminPageData struct {
	Site       SiteInfo
	Title      string
	Path       string
	Admin      string
	SignedIn   bool
	Editing    bool
	CSRF       string
	CSRFToken  string
	Stats      AdminStats
	Posts      []AdminPostRow
	Post       *Post
	Media      []MediaItem
	Flash      string
	Error      string
	DevLogin   bool
	DevMode    bool
	Query      string
	FilterBlog string
	Device     *DeviceLoginView
	Now        time.Time
}

type FeedItem struct {
	PostCard
	BodyHTML template.HTML
}

type FeedData struct {
	Site        SiteInfo
	Title       string
	Description string
	Path        string
	FeedPath    string
	Now         time.Time
	Posts       []FeedItem
}

type SitemapData struct {
	Posts []PostCard
	Now   time.Time
}

type DownloadData struct {
	Site SiteInfo
	Post *Post
	Now  time.Time
}

type PrintData struct {
	PageData
}

type DeviceLoginView struct {
	Code            string
	VerificationURI string
	CompleteURI     string
	ExpiresIn       int
	Interval        int
	Status          string
	Message         string
}
