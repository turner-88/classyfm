package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/classyfm/classyfm/internal/imgsave"
)

// listingItem is one article as scraped from a Hot Release listing page.
type listingItem struct {
	Title       string
	URL         string
	Excerpt     string
	PublishedAt time.Time
}

// detailPage is the parsed content of one article's detail page.
type detailPage struct {
	Body     string // paragraph texts joined with blank lines
	ImageSrc string // absolute image URL, or "" if none found
}

// listingDateLayout matches the old site's listing date format, e.g.
// "25-06-2026 11:06:15". The detail page's own date element is not usable -
// it renders the current date at request time on every post, not the actual
// publish date - so published_at is always taken from the listing instead.
const listingDateLayout = "02-01-2006 15:04:05"

// fetchListingPage fetches and parses one page of the Hot Release category
// listing. A nil/empty result means there's nothing on that page - the old
// site doesn't bound-check the page number, it just falls back to rendering
// unrelated homepage widgets past the last real page - so the caller treats
// an empty page as "no more pages".
func (imp *importer) fetchListingPage(ctx context.Context, page int) ([]listingItem, error) {
	u := fmt.Sprintf("%s/category/hot-release?page=%d", oldSiteBase, page)
	doc, err := imp.getDocument(ctx, u)
	if err != nil {
		return nil, err
	}

	var items []listingItem
	doc.Find("ul.article-list > li").Each(func(_ int, li *goquery.Selection) {
		info := li.Find(".info")
		href, _ := info.Find("a[href]").First().Attr("href")
		title := strings.TrimSpace(info.Find("h4.title").First().Text())
		excerpt := strings.TrimSpace(info.Find(".summary").First().Text())
		dateText := strings.TrimSpace(info.Find("span.date").First().Text())
		if href == "" || title == "" {
			return
		}
		publishedAt, err := time.ParseInLocation(listingDateLayout, dateText, time.Local)
		if err != nil {
			log.Printf("  warn: unparseable date %q for %q, skipping", dateText, title)
			return
		}
		items = append(items, listingItem{
			Title:       title,
			URL:         href,
			Excerpt:     excerpt,
			PublishedAt: publishedAt,
		})
	})
	return items, nil
}

// fetchDetail fetches an article's detail page and extracts its body
// paragraphs and featured image. ".page-body > p" only matches the article's
// own paragraph elements - the ads container and featured-image wrapper are
// divs, not bare <p> siblings, so they're excluded automatically.
func (imp *importer) fetchDetail(ctx context.Context, articleURL string) (detailPage, error) {
	doc, err := imp.getDocument(ctx, articleURL)
	if err != nil {
		return detailPage{}, err
	}

	var paras []string
	doc.Find(".page-body > p").Each(func(_ int, p *goquery.Selection) {
		text := strings.TrimSpace(p.Text())
		if text != "" {
			paras = append(paras, text)
		}
	})

	imageSrc, _ := doc.Find(`meta[property="og:image"]`).First().Attr("content")

	return detailPage{
		Body:     strings.Join(paras, "\n\n"),
		ImageSrc: strings.TrimSpace(imageSrc),
	}, nil
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	s := strings.ToLower(title)
	s = slugNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// uniqueSlug generates a URL slug for title that doesn't collide with any
// slug already seen this run or already in the database. The uq_slug unique
// index is the backstop either way.
func (imp *importer) uniqueSlug(title string) string {
	base := slugify(title)
	if base == "" {
		base = "hot-release"
	}
	slug := base
	for n := 2; imp.seenSlugs[slug]; n++ {
		slug = fmt.Sprintf("%s-%d", base, n)
	}
	return slug
}

// downloadImage fetches src, validates it's an allowed image type, saves it
// under uploadDir/hot-release/, and returns the public path to store as
// image_url - e.g. "/uploads/hot-release/xxxx.jpg", matching the exact
// convention the admin upload form uses.
func (imp *importer) downloadImage(ctx context.Context, src string) (string, error) {
	body, err := imp.get(ctx, src)
	if err != nil {
		return "", err
	}
	defer body.Close()

	buf, err := io.ReadAll(io.LimitReader(body, 5<<20))
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}

	head := buf
	if len(head) > 512 {
		head = head[:512]
	}
	ext, err := imgsave.ExtForHeader(head)
	if err != nil {
		return "", err
	}

	name, err := imgsave.RandomName(ext)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(imp.uploadDir, "hot-release")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create upload dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), buf, 0o644); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	return "/uploads/hot-release/" + name, nil
}

// get issues a GET request with a conventional User-Agent and returns the
// response body for the caller to read/close, erroring on non-200 status.
func (imp *importer) get(ctx context.Context, u string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ClassyFM-Import/1.0")
	resp, err := imp.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, u)
	}
	return resp.Body, nil
}

func (imp *importer) getDocument(ctx context.Context, u string) (*goquery.Document, error) {
	body, err := imp.get(ctx, u)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return goquery.NewDocumentFromReader(body)
}
