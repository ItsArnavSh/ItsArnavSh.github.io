package main

import (
	"context"
	"log/slog"
	"time"
)

type seedPost struct {
	Blog     string
	Slug     string
	Title    string
	Subtitle string
	Cover    string
	CoverAlt string
	Tags     []string
	DaysAgo  int
	Hour     int
	Featured bool
	Body     string
}

func seedIfEmpty(ctx context.Context, a *App) error {
	n, err := a.store.CountAll(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	now := a.now()
	for _, sp := range seedPosts {
		hour := sp.Hour
		if hour == 0 {
			hour = 9
		}
		published := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, a.cfg.Location).AddDate(0, 0, -sp.DaysAgo)
		in := PostInput{
			Blog:       sp.Blog,
			Slug:       sp.Slug,
			Title:      sp.Title,
			Subtitle:   sp.Subtitle,
			Excerpt:    excerptFrom(sp.Body, 230),
			CoverImage: sp.Cover,
			CoverAlt:   sp.CoverAlt,
			BodyMD:     sp.Body,
			Author:     "Arnav Sharma",
			Tags:       sp.Tags,
			Status:     "published",
			Featured:   sp.Featured,
			Published:  &published,
		}
		if _, err := a.store.CreatePost(ctx, in, "Arnav Sharma"); err != nil {
			return err
		}
	}
	slog.Info("the first edition has been set", "dispatches", len(seedPosts))
	return nil
}

var seedPosts = []seedPost{
	{
		Blog:     "devlog",
		Slug:     "a-newspaper-that-thinks-its-a-database",
		Title:    "A newspaper that thinks it's a database",
		Subtitle: "Building a two-edition blog on Postgres, Go and a stubborn refusal to add a framework",
		Cover:    "/media/seed-laptop.jpg",
		CoverAlt: "A laptop open on a bright desk, a monitor and papers behind it.",
		Tags:     []string{"Go", "Postgres", "Search"},
		DaysAgo:  1,
		Hour:     9,
		Featured: true,
		Body: `There is a particular kind of quiet that arrives when a side project stops being an idea and starts being a URL. This paper is the third attempt at that quiet. The first two attempts were static sites: beautiful, fast, and completely frozen the moment I wanted to publish from a train.

So this one is a small Go process. It renders HTML on the way out, keeps every dispatch as Markdown in Postgres, and pretends to be a newspaper while doing it. There is no CMS, no JavaScript framework, and no build step between me and the publish button.

## The index is a trigger

Search was the first thing I refused to bolt on later. Postgres already has a full-text engine; the trick is making it feel like part of the writing, not a separate service.

Each row carries a generated weighting:

` + "```sql" + `
NEW.search :=
  setweight(to_tsvector('english', coalesce(NEW.title, '')), 'A') ||
  setweight(to_tsvector('english', coalesce(NEW.subtitle, '') || ' ' || coalesce(NEW.excerpt, '')), 'B') ||
  setweight(to_tsvector('english', coalesce(NEW.body_md, '')), 'C');
` + "```" + `

Headlines matter most, decks and excerpts next, the body last. A trigger keeps the vector honest so that every save from the editor is immediately searchable, even before the request has finished writing its own redirect.

![The desk where most of this was written](/media/seed-flatlay.jpg "A flat lay of notebook, pen and coffee on a warm wooden desk.")

## Boring is a feature

The server is the standard library. ` + "`net/http`" + `, ` + "`html/template`" + `, a handful of handlers, and one middleware that sets security headers. The only dependencies that do real work are Goldmark for Markdown, pgx for the database, and Chroma for code colour. That is four moving parts, which is roughly three fewer than I can reliably keep in my head at 2 a.m.

The design brief was simple: two editions with opposite moods. The Devlog is dark espresso and burnt orange; the Journal is warm paper and brown. Same architecture, different light. The front page is a front page, not an infinite feed: one lead story, a couple of side stories, a list of what people actually read.

## What comes next

The view counter is deliberately modest. No cookies, no fingerprinting, no third-party script. A reader is a hash of address, browser and day, stored just long enough to avoid counting the same person twice before midnight. The number is the only thing that survives.

There is still a list of things I want to add: a proper archive page, a small reading list, and a way to subscribe without handing my address book to a company. But the presses are warm now, and that feels like enough for the first edition.`,
	},
	{
		Blog:     "devlog",
		Slug:     "latex-but-make-it-the-browsers-problem",
		Title:    "LaTeX, but make it the browser's problem",
		Subtitle: "A math pipeline that survives Markdown, code blocks and aggressive typography",
		Cover:    "/media/seed-notebook.jpg",
		CoverAlt: "A fountain pen writing across a page of notes.",
		Tags:     []string{"LaTeX", "KaTeX", "Markdown"},
		DaysAgo:  4,
		Hour:     10,
		Body: `Writing about systems without being able to write a formula is like describing a city without a map. The first draft of this blog had no math support, which meant every other paragraph contained an apology. This is the note about fixing that.

## Do not trust the parser

Goldmark is a very good Markdown parser. It is also not a LaTeX parser, and the overlap between the two grammars is a small diplomatic incident involving dollar signs, backslashes and code fences.

The approach here is to protect math before Markdown sees it. A small scanner walks the source, skips fenced code and inline code, and replaces each ` + "`$…$`" + ` or ` + "`$$…$$`" + ` span with an opaque token. Goldmark parses the rest. After rendering, the tokens are swapped for ` + "`<span class=\"math\" data-tex=\"…\">`" + ` elements and KaTeX takes over in the browser.

## Reading time, approximately

Even the reading estimate gets a formula, because it is funnier that way:

$$
\mathrm{minutes} = \left\lceil \frac{\mathrm{words}}{220} \right\rceil
$$

The number is not a promise. It is a small act of respect for the reader's attention, printed next to every headline.

![A notebook page with a half-finished proof](/media/seed-typing2.jpg "Some proofs are better left as photographs.")

## The fallback is the point

KaTeX is a client-side dependency, so the page has to make sense without it. The token replacement leaves the original TeX in the element, which means a failed render still shows ` + "`\\(E = mc^2\\)`" + ` rather than an empty shape. Math is part of the text, not an image of the text.

The same rule applies to code. Chroma emits classes, the stylesheet does the colouring, and if the CSS fails the code is still readable. Every enhancement is allowed to fail loudly and leave the paper intact.

## What I would change

There is a known edge case around currency: a sentence with two dollar amounts can look like a display equation to a very eager parser. The scanner requires a non-space character after the opening dollar and does not close on a digit, which catches most of them. If it ever misses one, the fix belongs in the scanner, not in a rule that tells writers to avoid money.

LaTeX in Markdown is a negotiation, not a translation. The job is to keep the negotiation invisible.`,
	},
	{
		Blog:     "devlog",
		Slug:     "a-small-go-server-deliberately-boring",
		Title:    "A small Go server, deliberately boring",
		Subtitle: "Templates, handlers and the strange pleasure of having no framework to blame",
		Cover:    "/media/seed-laptop2.jpg",
		CoverAlt: "A closed laptop on a pale wooden table.",
		Tags:     []string{"Go", "Architecture", "Templates"},
		DaysAgo:  8,
		Hour:     8,
		Body: `Every framework is a bet that the next six months will look like the last six. For a personal blog, that is a bad bet. The interesting parts of this project are the writing, the typography and the search index; the HTTP router is not one of them.

So the server is a ` + "`http.ServeMux`" + ` and a set of methods that look like this:

` + "```go" + `
func (a *App) handlePost(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")
    post, err := a.store.GetPost(r.Context(), "devlog", slug)
    if err != nil {
        a.notFound(w, r)
        return
    }
    body, toc := a.render.RenderMarkdown(post.BodyMD)
    post.BodyHTML = template.HTML(body)
    post.TOC = toc
    a.renderPage(w, "post", postPage(post))
}
` + "```" + `

There is no clever abstraction over "resources". There is a function for each URL, and if I want to know what a URL does I open the function with the same name.

## Templates in two worlds

The public pages and the editor share a design language but not a layout. They are parsed separately, cached separately, and reload from disk when ` + "`BLOG_DEV=1`" + ` is set. In production the templates are embedded in the binary, which makes deployment a single file plus a media directory.

The only shared piece is the function map: dates, view formatting, share URLs, the little ` + "`truncate`" + ` helper. Everything that reads like a design decision lives in the template or the CSS. Go is there to fetch rows and write bytes.

![A quiet corner of the machine room](/media/seed-setup.jpg "The server is small enough to fit in the corner of a notebook.")

## The cost of boring

The trade-off is obvious: I write more code than a framework would require for the same feature. Pagination, form validation and CSRF all have to be built, and built carefully. The reward is that nothing happens by magic. When a request is slow, I can point at the query or the render call. When a cookie is wrong, there is exactly one place that sets it.

For a project this size, boring is not a compromise. It is the feature. The paper should feel considered; the machinery under it should feel like a table. You only notice a table when it wobbles.

The next addition is probably a small archive page and a better index for the Journal. Neither needs a new dependency.`,
	},
	{
		Blog:     "devlog",
		Slug:     "counting-readers-without-surveillance",
		Title:    "Counting readers without becoming a surveillance company",
		Subtitle: "Views, Redis and the shortest possible memory",
		Cover:    "/media/seed-phone.jpg",
		CoverAlt: "A phone held in one hand against a blue-lit background.",
		Tags:     []string{"Privacy", "Redis", "Postgres"},
		DaysAgo:  12,
		Hour:     11,
		Body: `A view counter is a small promise: I want to know whether anyone is reading, and I do not want to know who. Most analytics products break the second half of that sentence. This one is built to keep it.

## A reader is a hash

On every article load, the browser posts the edition and the slug. The server combines the address, the user agent and the day into a hash, salted with a server secret, and stores that hash in ` + "`post_views`" + `. The primary key is ` + "`(post_id, visitor_hash, view_date)`" + `. The same reader can count once per dispatch per day. The hash cannot be reversed into an address, and tomorrow's salt makes it useless for linking today's visit to tomorrow's.

` + "```sql" + `
INSERT INTO post_views (post_id, visitor_hash, view_date)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;
` + "```" + `

If the insert touched a row, the view count moves by one. If it did not, nothing happens. There is no event stream, no session reconstruction, no dashboard with a map.

## Redis is a short memory

The database is the source of truth. Redis is a bouncer at the door: a ` + "`SETNX`" + ` key that expires after 26 hours, so a refresh does not even reach Postgres. If Redis is down, the database constraint still does the right thing. If both are down, the page still loads and the counter is simply a little stale.

This is the only place the blog uses Redis, apart from a short-lived cache for search results. The cache is invalidated whenever a dispatch changes, because a search index that lies is worse than no cache at all.

![A telephone that is not listening](/media/seed-cafe2.jpg "Three cups, a shared table, and no notifications.")

## What the numbers are for

The view count appears on the front page, on each dispatch, and in the editor's dashboard. It is a rough signal of what people find useful, not a ranking algorithm. There is no "recommended for you", because the paper does not know enough about you to recommend anything honestly.

The counter also has a job in the editor: it tells me whether a piece found its audience. A post with three views is not a failure; it might be a note to myself. But a post with three views and a lot of downloads is interesting, and that is exactly the kind of pattern a small counter can reveal.

The rule for future features is simple: if it would require remembering a reader across days, it does not belong here. The archive should remember the words, not the reader.`,
	},
	{
		Blog:     "devlog",
		Slug:     "the-cache-that-lied-at-2am",
		Title:    "The 2 a.m. post-mortem: a cache that lied",
		Subtitle: "What a stale search result taught me about invalidation and trust",
		Cover:    "/media/seed-typing.jpg",
		CoverAlt: "Hands typing on a laptop in a dim room.",
		Tags:     []string{"Postmortem", "Redis", "Search"},
		DaysAgo:  18,
		Hour:     2,
		Body: `The bug report was one sentence long: "I edited a post and search still shows the old title." It arrived at 1:47 a.m., which is the hour when every cache is guilty until proven innocent.

## The shape of the bug

Search results were cached in Redis for ninety seconds, keyed by the lowercased query and the edition. When a post was saved, the handler cleared the cache. That should have been enough. It was not, because the editor had two tabs open.

The first tab saved the post and cleared the cache. The second tab, still holding an old draft, saved it again a few seconds later. The second save re-populated the cache before the transaction had committed its new search vector. For the next ninety seconds the archive confidently served a title that no longer existed.

` + "```go" + `
if err := a.store.UpdatePost(ctx, id, in); err != nil {
    return err
}
a.store.ClearSearchCache(ctx)
` + "```" + `

The code looks correct. The ordering was the problem, and the ordering was invisible because the cache lived outside the transaction.

## The fix, and the non-fix

The first fix was to clear the cache after the commit and again after a short delay. That is a band-aid with a timer, and timers are how caches become folklore. The real fix was simpler: never cache a result that was computed from a write still in flight. The store now returns only after the transaction commits, and the cache is cleared by the handler after that return. The duplicate-tab race is still possible, but it now resolves to the newer draft instead of an older ghost.

I also added a cache key version. When the schema or the ranking changes, the version moves and every old entry becomes unreachable without a scan. It is a small thing that makes deployments less exciting.

![The scene of the crime, approximately](/media/seed-cafe.jpg "The café where the bug was eventually reproduced, badly.")

## What I keep from it

Caches do not lie; they repeat what they were told. The lesson is not "avoid caching". Search is one of the few places where a ninety-second cache is genuinely worth it, and the archive is read far more than it is written. The lesson is to make the write path boring and the read path honest about how old it can be.

The post-mortem is now part of the devlog, which feels appropriate. A newspaper that never prints corrections is not a newspaper; it is a brochure. The archive should keep its mistakes visible, at least the ones that taught it something.`,
	},
	{
		Blog:     "devlog",
		Slug:     "epub-is-a-zip-file",
		Title:    "EPUB is a zip file and that is the whole trick",
		Subtitle: "Shipping a dispatch to an e-reader without a publishing pipeline",
		Cover:    "/media/seed-book.jpg",
		CoverAlt: "A stack of books on a desk in warm light.",
		Tags:     []string{"EPUB", "Go", "Markdown"},
		DaysAgo:  26,
		Hour:     14,
		Body: `The download desk has six buttons, and the EPUB is the one that makes people raise an eyebrow. "You built an e-reader format?" Not really. I built a zip file with three XML documents and a stylesheet, which is what an EPUB is once you stop being afraid of it.

## The minimum viable book

An EPUB 3 file needs:

- a ` + "`mimetype`" + ` entry containing exactly ` + "`application/epub+zip`" + `, stored first and uncompressed;
- ` + "`META-INF/container.xml`" + ` pointing at the package document;
- a package document with metadata, a manifest and a spine;
- a navigation document;
- at least one XHTML content document.

That is the entire contract. Everything else — fonts, cover images, accessibility metadata — is an improvement, not a requirement.

` + "```go" + `
zw := zip.NewWriter(&buf)
h := &zip.FileHeader{Name: "mimetype", Method: zip.Store}
w, _ := zw.CreateHeader(h)
w.Write([]byte("application/epub+zip"))
` + "```" + `

The only real care is XHTML. Browsers forgive unclosed tags; e-readers do not. Images and horizontal rules have to be self-closing, entities have to be numeric, and the whole thing has to parse as XML. A few regular expressions handle the Goldmark output, and a test reader catches the rest.

## Images are the hard part

A dispatch can have photographs. If the EPUB links to ` + "`/media/…`" + `, the reader will show a broken image once the file leaves the server. So the builder scans the HTML for local image sources, reads each file from the media directory, copies it into ` + "`OEBPS/images/`" + `, and rewrites the source to a relative path. Remote images are left alone; they are someone else's problem, and probably someone else's tracking pixel.

The result is a self-contained book. The cover is embedded, the captions survive, and the typography is a deliberate cousin of the web version rather than a screenshot of it.

![A shelf of things worth keeping](/media/seed-window.jpg "The best format is the one you can open in ten years.")

## Why bother

Because a blog is a public notebook, and a notebook should be portable. The Markdown download is the source; the EPUB is the reading copy; the print view is the version you can hand to someone. None of them require an account, an app, or a permission slip from a platform.

The next version will embed a proper cover image and a colophon page. For now, the zip opens, the chapters turn, and the words are the same words. That is enough.`,
	},
	{
		Blog:     "journal",
		Slug:     "a-desk-by-the-window",
		Title:    "A desk by the window",
		Subtitle: "On light, small rooms and the luxury of a place to put a notebook",
		Cover:    "/media/seed-window.jpg",
		CoverAlt: "A bright window with plants and a laptop on the sill.",
		Tags:     []string{"Notes", "Delhi", "Rooms"},
		DaysAgo:  2,
		Hour:     7,
		Featured: true,
		Body: `The desk is not really a desk. It is a table that was once used for dinner, pushed against the window because the window is the only part of the room that does not feel rented. In the morning the light comes in sideways and lands on the wall opposite, and for about twenty minutes the room looks like a photograph of somewhere else.

I have moved this table four times in two years. Each time I promise it will be the last. Each time I discover a new problem: the chair is too low, the window faces the wrong way, the cable is two feet short. The current arrangement is the first one where I can sit down without moving anything first. That is a small thing, but it changes the entire day.

## The economics of a corner

There is a version of working from home that looks like a stock photo: a large monitor, a plant, a mechanical keyboard arranged at a perfect angle. This is not that. This is one laptop, one notebook, a glass of water, and a stack of books that I keep meaning to shelve. The corner is small enough that everything has to earn its place.

The notebook stays because it is where the day starts. The book stays because it is the one I am actually reading. The second book stays because it looks good, which is not a real reason, but it is an honest one.

![Morning light on the table](/media/seed-flatlay.jpg "The table before the day starts. It never stays this tidy.")

## What the window does

A window is a low-bandwidth feed. There is a tree outside, a wall, a sliver of sky, and a man who walks past every morning at roughly the same time with a bag that looks too heavy. I do not know his name. I have never spoken to him. But his routine has become part of mine, a small external clock that does not require a notification.

The internet is designed to be more interesting than a window. It usually is. But the window has one advantage: it never asks me to engage. It is simply there, doing its slow thing, and after a while the slow thing becomes the point.

## A room of one's own, approximately

Virginia Woolf asked for a room and money. I have a corner and a table, which is not the same, but it is close enough to feel like a beginning. The room is shared with other people's schedules and other people's noise. The corner is mine for as long as the light lasts.

In the evening the window turns into a mirror. You can see the desk, the lamp, the shape of the person sitting at it. It is not a flattering mirror. It is, however, an accurate one.`,
	},
	{
		Blog:     "journal",
		Slug:     "the-cafe-that-keeps-the-internet-at-arm's-length",
		Title:    "The café that keeps the internet at arm's length",
		Subtitle: "A table, a notebook, and the small discipline of leaving the phone face down",
		Cover:    "/media/seed-cafe.jpg",
		CoverAlt: "A café interior with pendant lights and a long counter.",
		Tags:     []string{"Cafés", "Writing", "Attention"},
		DaysAgo:  6,
		Hour:     16,
		Body: `The café is not the best café nearby. The coffee is fine, the chairs are uncomfortable, and the music is usually louder than it needs to be. It has one advantage: the wifi is bad enough that I stop trying to use it.

I come here with a notebook and a pen. That is the whole setup. No laptop, no headphones, no second screen. The first fifteen minutes are always a small negotiation with boredom. Then the notebook opens, and the negotiation is over.

## Writing by hand is slow on purpose

Typing is faster than writing, which is exactly the problem. When I type, the sentence arrives almost as fast as I think it, and I end up with paragraphs that are technically complete and emotionally approximate. When I write by hand, the sentence has to wait for my wrist. The delay is not a bug. It is the editing pass.

The pages are messy. There are arrows, crossed-out words, and a drawing of a cup that looks like a small building. None of it is publishable. Some of it will become a dispatch; most of it will become compost. The notebook does not care.

![The table by the window](/media/seed-cafe2.jpg "A second cup, a second draft, the same bad chairs.")

## The phone goes face down

This is not a productivity tip. It is a small ritual that makes the table feel like a place instead of a waiting room. Face down, the phone is a rectangle. Face up, it is a small window into everyone else's expectations. I do not need those at 4 p.m. on a Tuesday.

Occasionally I cheat. I pick it up to check the time and emerge eight minutes later having read three messages and one article about a man who builds tiny bridges. The notebook is patient. It waits.

## The bill as a boundary

The other thing the café gives me is an ending. At home, work can continue indefinitely; there is always one more thing to fix. At the café, eventually the cup is empty and the table needs to be returned to the next person. The session has a shape. That shape is worth the price of a coffee.

I leave with maybe 400 usable words, which is not a lot. But they are words that did not exist before, and they were written in a place that asked nothing of me except that I buy a drink and not stay past closing. That is a good trade.`,
	},
	{
		Blog:     "journal",
		Slug:     "what-i-keep-in-the-notebook",
		Title:    "What I keep in the notebook",
		Subtitle: "A pocket-sized inventory of half-finished ideas",
		Cover:    "/media/seed-notebook.jpg",
		CoverAlt: "A fountain pen moving across a page of handwritten notes.",
		Tags:     []string{"Notebooks", "Process"},
		DaysAgo:  14,
		Hour:     21,
		Body: `The notebook is small enough to fit in a jacket pocket, which is the only feature that matters. A notebook that lives in a bag is a notebook that stays in the bag. This one is always within reach, and as a result it is full of things that would otherwise be lost.

The contents are not impressive. There are shopping lists, a diagram of a database schema, the name of a song I heard in a shop, and a note that says "the problem with the first draft is that it is a first draft". Every few weeks I go through it and cross things out. The crossing out is the most satisfying part.

## The anatomy of a page

A typical page has three kinds of writing. The top is the thing I do not want to forget: a title, a URL, a line of dialogue. The middle is the thing I am trying to think through, usually a mess of arrows. The bottom is the thing I will do later, which is almost always "write this properly".

I have tried elaborate systems: index cards, colour codes, a bullet journal with a key on the first page. They all lasted about two weeks. The pocket notebook survives because it has no system. It is a place to put a thought, not a productivity apparatus.

![A flat lay of paper and pens](/media/seed-flatlay.jpg "Everything in the notebook starts as a small, unreadable note.")

## The return of the lost thought

The best thing about keeping a notebook is not the notes themselves. It is the moment, months later, when a current problem turns out to be the same problem you wrote down in a different season. The page is there, in a slightly embarrassing handwriting, saying: you have been here before. Here is what you learned.

That is a kind of memory the internet does not provide. Search gives you everything at once; a notebook gives you things in the order you needed them. The order is the meaning.

## What does not go in

I do not keep a diary in it. I tried, and the result was a series of entries that started with the weather and ended with an apology for not writing more. A diary is a different muscle. The notebook is for work, fragments, and the occasional drawing of a cup that looks like a building.

The rule is simple: if it fits on a page and it will be useful later, it goes in. If it needs more than a page, it is a draft, and drafts go in the laptop. The notebook is a pocket, not a library. That is what makes it portable.`,
	},
	{
		Blog:     "journal",
		Slug:     "a-walk-without-a-podcast",
		Title:    "A walk without a podcast",
		Subtitle: "Twenty minutes of not being informed about anything",
		Cover:    "/media/seed-forest.jpg",
		CoverAlt: "Dense green trees reflected in a calm river.",
		Tags:     []string{"Walking", "Attention", "Silence"},
		DaysAgo:  22,
		Hour:     17,
		Body: `The first ten minutes are the worst. Without a podcast, the mind behaves like a browser with too many tabs. There is a work problem, then a conversation from three days ago, then a song, then a sudden desire to check whether the library book is overdue. The silence does not feel peaceful. It feels like being left alone with an unpaid bill.

I keep walking. That is the only technique. The road turns, the trees get taller, and the tabs begin to close on their own. By the fifteen-minute mark there is usually one thought left, and it is often not the one I started with.

## The path

The path is not scenic in the postcard sense. It runs behind a row of houses, crosses a small stream, and ends at a stretch of trees that were planted by someone with more patience than the current residents. There are dogs, bicycles, and a man who waters his plants at exactly 5:30. It is a route, not a destination.

That is the point. A walk with a destination is travel; a walk without one is maintenance. I do not need to arrive anywhere. I need to move for long enough that the body stops asking and the mind starts listening.

![Trees doing their slow work](/media/seed-forest.jpg "The trees behind the houses, just before the light goes.")

## What comes back

Without a podcast, I notice things I would otherwise edit out: the sound of a gate, the smell of someone cooking, the way the light changes under a particular tree. None of this is useful. All of it is material. A sentence I have been stuck on often resolves itself somewhere around the stream, not because the walk is magical, but because the work has been left alone.

There is a lot of advice about "being present" that sounds like a product description. This is simpler and less impressive: take a walk, leave the headphones at home, come back. The first ten minutes will be bad. The rest is the reason.

## The return

I come back with no steps to report to anyone and no article to cite. The phone has been in my pocket the whole time, counting nothing. That feels like a small act of resistance, even if it is only resistance against my own habits.

The house is still there. The work is still there. But the tabs are closed, and for an hour the evening belongs to nobody in particular. That is enough.`,
	},
	{
		Blog:     "journal",
		Slug:     "the-photograph-you-didnt-take",
		Title:    "The photograph you didn't take",
		Subtitle: "On leaving the camera in the bag and letting the evening be enough",
		Cover:    "/media/seed-phone.jpg",
		CoverAlt: "A phone held in one hand, glowing blue.",
		Tags:     []string{"Photographs", "Attention"},
		DaysAgo:  38,
		Hour:     18,
		Body: `The light was perfect for about ninety seconds. It came in low through the café window, caught the edge of the cup, and turned the whole table into something worth photographing. I had my phone in my hand. I did not take the picture.

I am not sure why. There was no decision, exactly. The moment was already complete, and lifting the phone felt like it would turn the moment into a task. So I sat there and watched the light move across the table, and then it was gone.

## The archive of almosts

Photographers talk about "the decisive moment", which is a useful phrase until it becomes a reason to be anxious all the time. Most moments are not decisive. They are small, ordinary, and gone before the camera is out of the bag. The photograph you did not take is not a failure unless you decide it is.

I used to keep a mental list of missed photographs: a man asleep on a station bench, a dog in a yellow raincoat, the reflection of a train in a puddle. The list was a way of holding on. Now it feels more like a collection of places I was lucky enough to see.

## Looking is the practice

A camera is a good excuse to look closely. Without one, looking has to be its own excuse. That is harder. The phone offers a shortcut: press the button, and the looking is stored somewhere else, to be done later. Leaving it in the pocket means the looking has to happen now.

The café kept its light for a few more minutes, then the clouds came over and the room went grey. I finished my coffee. The evening was not recorded. It was, however, attended to.

## What remains

I remember the cup, the window, the sound of the espresso machine, and the particular shade of the table. The memory is not a photograph; it is softer, more partial, and easier to lose. But it belongs to me in a way that a picture in a camera roll never quite does.

The next time the light is perfect, I may take the photograph. This time, I am glad I did not.`,
	},
	{
		Blog:     "journal",
		Slug:     "the-controller-on-the-shelf",
		Title:    "The controller on the shelf",
		Subtitle: "On games, time, and the things we keep for the person we used to be",
		Cover:    "/media/seed-controller.jpg",
		CoverAlt: "A game controller glowing in a dark room.",
		Tags:     []string{"Games", "Memory", "Time"},
		DaysAgo:  32,
		Hour:     20,
		Body: `The controller has been on the shelf for two years. It is not broken. It is not obsolete. It is simply waiting for a version of me that has four uninterrupted hours and no better idea of how to spend them.

I bought it during a week when I was convinced that playing games was the missing piece of a balanced life. I played two games, loved one, and then the semester started. The controller migrated from the desk to the shelf, then from the shelf to the top of the bookshelf, where it now keeps company with a broken hard drive and a camera I never learned to use.

## The person who owned it

There is a specific kind of nostalgia for the person you were when you bought a thing. The controller is not just a controller; it is a small monument to the belief that leisure was something I could schedule. The belief was not wrong. The schedule was.

I still think about the game I did not finish. Not the plot, which I have forgotten, but the feeling of the first hour: the menus, the music, the sense that the world was large and the rules were discoverable. That feeling is the reason people play games. It is also the reason they keep controllers they no longer use.

![A childhood shelf, rearranged](/media/seed-book.jpg "Books on the shelf, waiting for the controller.")

## Keeping things for later

There is a practical argument for getting rid of it. It takes up space. It will be worth less next year. If I want to play again, I can buy another one. The practical argument is correct, which is why it is not the whole argument.

The controller is a bookmark. It says: you are allowed to want this again. You are allowed to have an evening that produces nothing, that does not go into a notebook, that cannot be downloaded in five formats. The shelf is not a graveyard. It is a place for things that are still waiting.

## The evening

Last week I took it down, plugged it in, and played for forty minutes. The controls came back faster than I expected. The game was smaller than I remembered, and the evening was better than I expected. Then I put it back on the shelf, next to the camera, and went to bed.

I do not know when I will play again. That is fine. The controller is not a debt. It is an invitation, and it can stay open.`,
	},
}
