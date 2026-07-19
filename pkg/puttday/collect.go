// Package puttday scrapes putt.day share links for a player's recorded hole
// and stroke count.
package puttday

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrInvalidShareLink is returned when a link is not a putt.day share URL.
	ErrInvalidShareLink = errors.New("invalid share link")
	// ErrResultNotFound is returned when a share page doesn't contain a
	// recognizable hole/stroke result.
	ErrResultNotFound = errors.New("result not found in page")
	// ErrCustomMap is returned when a share link is for a custom
	// (player-made) map rather than a numbered daily hole. Custom maps have
	// no hole number, so there's nothing to record; callers should treat
	// this as "nothing to do" rather than a failure.
	ErrCustomMap = errors.New("share link is for a custom map, not a numbered hole")
)

var (
	// numberedResultPattern matches a share page for a numbered daily hole,
	// e.g. `Hole #65</b> in <b>6</b>`.
	numberedResultPattern = regexp.MustCompile(`Hole #(\d+)</b> in <b>(\d+)</b>`)
	// customMapPattern matches a share page for a custom (player-made) map,
	// which names the map in quotes instead of a hole number, e.g.
	// `“In the Clouds”</b> in <b>2</b>`. putt.day uses curly quotes, but
	// straight quotes are accepted too in case that ever changes.
	customMapPattern = regexp.MustCompile(`<b>[“"][^”"<]+[”"]</b> in <b>\d+</b>`)
	// mapIDPattern matches the "Play this hole" button that share pages for
	// custom (player-made) maps render, e.g.
	// `href="/h/mqftqdwy7z3e5j" ...>Play this hole<`. That /h/<id> path is
	// the map's stable, persistent identifier, unlike the page's sequential
	// display number. Confirmed (via putt.day's own client bundle) that
	// numbered daily holes never render this button — they get a generic
	// "Play today's hole" link to /play instead, since a daily hole has no
	// persistent /h/ page of its own. So this pattern only ever matches on
	// custom-map pages.
	mapIDPattern = regexp.MustCompile(`href="/h/([^"]+)"[^>]*>Play this hole<`)
)

type collector struct {
	client *http.Client
}

// CollectorOption configures a Collector constructed by NewCollector.
type CollectorOption func(*collector)

// WithHttpClient overrides the HTTP client a Collector uses to fetch share
// pages.
func WithHttpClient(client *http.Client) CollectorOption {
	return func(c *collector) {
		c.client = client
	}
}

// NewCollector constructs a share-link scraper, using a default
// 10-second-timeout HTTP client unless overridden by opts.
func NewCollector(opts ...CollectorOption) *collector {
	c := &collector{
		client: &http.Client{
			Transport: http.DefaultTransport,
			Timeout:   10 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// SharedScore is a single hole result scraped from a putt.day share link.
type SharedScore struct {
	Hole    int
	Strokes int
	Link    string
	// MapID is the persistent id from the share page's "Play this hole"
	// button (`/h/<id>`), which identifies the underlying map itself as
	// opposed to Hole's sequential display number. It is only ever
	// populated for custom (player-made) maps: numbered daily holes have no
	// such button on their share page (see mapIDPattern), and custom-map
	// share pages never reach this struct at all today, since Collect
	// short-circuits them with ErrCustomMap before building a SharedScore.
	// In practice MapID is therefore always empty; the field and its
	// extraction are wired up ready for whenever a caller needs it.
	MapID string
}

// Collect takes a shareLink and returns a ([SharedScore], error)
func (c *collector) Collect(ctx context.Context, shareLink string) (*SharedScore, error) {
	var sharedScore = &SharedScore{}
	parsedShareLink, err := parseAndValidateShareLink(shareLink)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedShareLink.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("crafting request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("retrieving score: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}
	match := numberedResultPattern.FindSubmatch(body)
	if match == nil {
		if customMapPattern.Match(body) {
			return nil, fmt.Errorf("%w: %s", ErrCustomMap, shareLink)
		}
		return nil, fmt.Errorf("%w: %s", ErrResultNotFound, shareLink)
	}

	sharedScore.Hole, err = strconv.Atoi(string(match[1]))
	if err != nil {
		return nil, fmt.Errorf("parsing hole number: %w", err)
	}
	sharedScore.Strokes, err = strconv.Atoi(string(match[2]))
	if err != nil {
		return nil, fmt.Errorf("parsing score: %w", err)
	}
	sharedScore.Link = parsedShareLink.String()

	// The "Play this hole" button (and thus MapID) is only ever present on
	// custom-map share pages, not numbered-hole ones like this success
	// branch. Its absence here is expected, not an error: unlike Hole and
	// Strokes, MapID is supplementary data, so a missing button just leaves
	// it as the zero value rather than failing the whole collect.
	if mapIDMatch := mapIDPattern.FindSubmatch(body); mapIDMatch != nil {
		sharedScore.MapID = string(mapIDMatch[1])
	}

	return sharedScore, nil
}

func parseAndValidateShareLink(shareLink string) (*url.URL, error) {
	parsedShareLink, err := url.Parse(shareLink)
	if err != nil {
		return nil, fmt.Errorf("parsing share link: %w", err)
	}
	if !validateShareUrl(parsedShareLink) {
		return nil, fmt.Errorf("%w: %s", ErrInvalidShareLink, shareLink)
	}
	return parsedShareLink, nil
}

func validateShareUrl(u *url.URL) bool {
	return u.Host == "putt.day" && u.Scheme == "https" && strings.HasPrefix(u.Path, "/s/")
}
