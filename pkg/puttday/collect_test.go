package puttday

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestValidateShareUrl(t *testing.T) {
	testCases := []struct {
		desc  string
		link  string
		valid bool
	}{
		{
			desc:  "valid share link",
			link:  "https://putt.day/s/HRTRtJo8DO53",
			valid: true,
		},
		{
			desc:  "http scheme rejected",
			link:  "http://putt.day/s/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "wrong host rejected",
			link:  "https://evil.com/s/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "host with putt.day as suffix rejected",
			link:  "https://notputt.day/s/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "missing /s/ path prefix rejected",
			link:  "https://putt.day/x/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "explicit port changes host and is rejected",
			link:  "https://putt.day:443/s/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "case-sensitive host is rejected",
			link:  "https://PUTT.DAY/s/HRTRtJo8DO53",
			valid: false,
		},
		{
			desc:  "bare /s/ prefix with no id is accepted (known gap: no id-format validation)",
			link:  "https://putt.day/s/",
			valid: true,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			u, err := url.Parse(tC.link)
			if err != nil {
				t.Fatalf("parsing: %v", err)
			}
			if got := validateShareUrl(u); got != tC.valid {
				t.Fatalf("validateShareUrl(%q) = %t, want %t", tC.link, got, tC.valid)
			}
		})
	}
}

func TestParseAndValidateShareLink(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		u, err := parseAndValidateShareLink("https://putt.day/s/HRTRtJo8DO53")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := u.String(); got != "https://putt.day/s/HRTRtJo8DO53" {
			t.Errorf("parseAndValidateShareLink() = %q, want %q", got, "https://putt.day/s/HRTRtJo8DO53")
		}
	})

	t.Run("wrong host wraps ErrInvalidShareLink", func(t *testing.T) {
		_, err := parseAndValidateShareLink("https://evil.com/s/HRTRtJo8DO53")
		if !errors.Is(err, ErrInvalidShareLink) {
			t.Fatalf("error = %v, want wrapping ErrInvalidShareLink", err)
		}
	})

	t.Run("malformed url does not report ErrInvalidShareLink", func(t *testing.T) {
		_, err := parseAndValidateShareLink("://bad-url")
		if err == nil {
			t.Fatal("expected an error for a malformed URL")
		}
		if errors.Is(err, ErrInvalidShareLink) {
			t.Fatalf("error = %v, want a URL parse error, not the ErrInvalidShareLink sentinel", err)
		}
	})
}

// rewriteHostTransport redirects every request to target while leaving the
// path/query untouched, so tests can point a *collector at an httptest
// server even though validateShareUrl requires the host to be putt.day.
type rewriteHostTransport struct {
	target *url.URL
}

func (rt *rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	out.URL.Scheme = rt.target.Scheme
	out.URL.Host = rt.target.Host
	out.Host = rt.target.Host
	return http.DefaultTransport.RoundTrip(out)
}

func newTestCollector(t *testing.T, handler http.HandlerFunc) *collector {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}
	return NewCollector(WithHttpClient(&http.Client{Transport: &rewriteHostTransport{target: target}}))
}

func TestCollect(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		statusCode  int
		wantHole    int
		wantStrokes int
		wantErr     error
	}{
		{
			name:        "success",
			body:        `Someone finished <b>Hole #65</b> in <b>6</b> strokes. Can you beat it?`,
			statusCode:  http.StatusOK,
			wantHole:    65,
			wantStrokes: 6,
		},
		{
			name:        "success with multi-digit hole and score",
			body:        `finished <b>Hole #100</b> in <b>12</b> strokes.`,
			statusCode:  http.StatusOK,
			wantHole:    100,
			wantStrokes: 12,
		},
		{
			name:       "result not found in page",
			body:       `<html><body>this page has no recorded result</body></html>`,
			statusCode: http.StatusOK,
			wantErr:    ErrResultNotFound,
		},
		{
			name:       "empty body",
			body:       ``,
			statusCode: http.StatusOK,
			wantErr:    ErrResultNotFound,
		},
		{
			name:        "non-200 status with matching body still parses (current behavior: status code is not checked)",
			body:        `finished <b>Hole #7</b> in <b>3</b> strokes.`,
			statusCode:  http.StatusNotFound,
			wantHole:    7,
			wantStrokes: 3,
		},
		{
			name:       "custom map with curly quotes is dropped, not treated as not-found",
			body:       `Someone finished <b>“In the Clouds”</b> in <b>2</b> strokes. Can you beat it?`,
			statusCode: http.StatusOK,
			wantErr:    ErrCustomMap,
		},
		{
			name:       "custom map with straight quotes is dropped, not treated as not-found",
			body:       `finished <b>"Windy Ridge"</b> in <b>9</b> strokes.`,
			statusCode: http.StatusOK,
			wantErr:    ErrCustomMap,
		},
	}

	const shareLink = "https://putt.day/s/HRTRtJo8DO53"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			})

			score, err := c.Collect(context.Background(), shareLink)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Collect() error = %v, want wrapping %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Collect() unexpected error: %v", err)
			}
			if score.Hole != tt.wantHole || score.Strokes != tt.wantStrokes {
				t.Errorf("Collect() = %+v, want Hole=%d Strokes=%d", score, tt.wantHole, tt.wantStrokes)
			}
			if score.Link != shareLink {
				t.Errorf("Collect().Link = %q, want %q", score.Link, shareLink)
			}
		})
	}
}

func TestCollect_InvalidShareLinkMakesNoHTTPCall(t *testing.T) {
	c := NewCollector(WithHttpClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected HTTP call for an invalid share link")
			return nil, nil
		}),
	}))

	_, err := c.Collect(context.Background(), "https://evil.com/s/HRTRtJo8DO53")
	if !errors.Is(err, ErrInvalidShareLink) {
		t.Fatalf("Collect() error = %v, want wrapping ErrInvalidShareLink", err)
	}
}

func TestCollect_TransportError(t *testing.T) {
	wantErr := errors.New("network exploded")
	c := NewCollector(WithHttpClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, wantErr
		}),
	}))

	_, err := c.Collect(context.Background(), "https://putt.day/s/HRTRtJo8DO53")
	if err == nil || !strings.Contains(err.Error(), "retrieving score") {
		t.Fatalf("Collect() error = %v, want error containing %q", err, "retrieving score")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("Collect() error = %v, want wrapping %v", err, wantErr)
	}
}

func TestCollect_BodyReadError(t *testing.T) {
	c := NewCollector(WithHttpClient(&http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       errReadCloser{},
				Header:     make(http.Header),
			}, nil
		}),
	}))

	_, err := c.Collect(context.Background(), "https://putt.day/s/HRTRtJo8DO53")
	if err == nil || !strings.Contains(err.Error(), "reading response body") {
		t.Fatalf("Collect() error = %v, want error containing %q", err, "reading response body")
	}
}

func TestCollect_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewCollector()
	_, err := c.Collect(ctx, "https://putt.day/s/HRTRtJo8DO53")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Collect() error = %v, want wrapping context.Canceled", err)
	}
}

// roundTripFunc lets a plain function satisfy http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// errReadCloser is an io.ReadCloser that always fails to read, used to
// exercise Collect's response-body-read error path.
type errReadCloser struct{}

func (errReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("simulated read failure")
}

func (errReadCloser) Close() error {
	return nil
}
