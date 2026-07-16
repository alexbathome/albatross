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
)

var resultPattern = regexp.MustCompile(`Hole #(\d+)</b> in <b>(\d+)</b>`)

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
	match := resultPattern.FindSubmatch(body)
	if match == nil {
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
