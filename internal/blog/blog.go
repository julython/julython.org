package blog

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

//go:embed content/*.md
var content embed.FS

type Post struct {
	Title      string    `yaml:"title"`
	Date       time.Time `yaml:"-"`
	RawDate    string    `yaml:"date"`
	Slug       string    `yaml:"slug"`
	Blurb      string    `yaml:"blurb"`
	Body       string
	HasMermaid bool
}

// mermaidRe matches ```mermaid...``` fenced blocks.
// We pre-process these before goldmark so they pass through as raw HTML
// and mermaid.js can render them client-side.
var mermaidRe = regexp.MustCompile("(?s)```mermaid\n(.*?)```")

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle("dracula"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(false), // inline styles, no extra CSS needed
			),
		),
	),
	goldmark.WithRendererOptions(
		html.WithUnsafe(), // needed to pass pre-processed mermaid HTML through
	),
)

// All returns all posts sorted newest first.
func All() ([]Post, error) {
	entries, err := fs.ReadDir(content, "content")
	if err != nil {
		return nil, err
	}
	posts := make([]Post, 0, len(entries))
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		p, err := parse("content/" + e.Name())
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, nil
}

// BySlug returns a single post by slug.
func BySlug(slug string) (Post, error) {
	return parse("content/" + slug + ".md")
}

func parse(path string) (Post, error) {
	data, err := content.ReadFile(path)
	if err != nil {
		return Post{}, err
	}

	var p Post
	body, err := frontmatter.Parse(strings.NewReader(string(data)), &p)
	if err != nil {
		return Post{}, err
	}
	p.Date, err = time.Parse("2006-01-02", p.RawDate)
	if err != nil {
		return Post{}, fmt.Errorf("parsing date in %s: %w", path, err)
	}

	// Pre-process mermaid blocks — replace with raw HTML that goldmark
	// passes through unchanged, and mermaid.js picks up on the client.
	src := mermaidRe.ReplaceAllStringFunc(string(body), func(match string) string {
		p.HasMermaid = true
		inner := mermaidRe.FindStringSubmatch(match)[1]
		return "<pre class=\"mermaid\">\n" + inner + "</pre>\n"
	})

	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return Post{}, err
	}
	p.Body = buf.String()
	return p, nil
}
