package discovery

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BasicWebCrawler fetches a search-results page for a given path.
type BasicWebCrawler struct {
	client  *http.Client
	baseURL *url.URL
}

// NewBasicWebCrawler creates a new web crawler instance
func NewBasicWebCrawler(baseURLStr string) (*BasicWebCrawler, error) {
	// Parse the base URL once at initialization
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &BasicWebCrawler{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		baseURL: baseURL,
	}, nil
}

// ResolvePath properly resolves a URL string against the base URL
func (c *BasicWebCrawler) ResolvePath(path string) (string, error) {
	// constructing proper path incase of spaces
	pathSlice := strings.Split(path, " ")
	pathNoSpaces := strings.Join(pathSlice, "%20")
	joinedUrl := c.baseURL.String() + pathNoSpaces

	pathURL, err := url.Parse(joinedUrl)
	if err != nil {
		return "", err
	}

	// resolve against original base URL
	resolvedURL := c.baseURL.ResolveReference(pathURL)
	return resolvedURL.String(), nil
}

// CrawlPath fetches a webpage and returns its content
func (c *BasicWebCrawler) CrawlPath(ctx context.Context, path string) ([]byte, error) {
	resolvedURL, err := c.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "crawling URL", "url", resolvedURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// set up appropriate headers to avoid being blocked by youtube
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %w", resolvedURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d for URL %s", resp.StatusCode, resolvedURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}
