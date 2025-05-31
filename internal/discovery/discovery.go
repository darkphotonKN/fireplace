package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/darkphotonKN/fireplace/internal/concepts"
	"github.com/darkphotonKN/fireplace/internal/constants"
	"golang.org/x/net/html"
)

type Resource struct {
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Source      string                 `json:"source"`
	Type        constants.ResourceType `json:"type"`
	Description string                 `json:"description"`
}

// a discovery finder need to be able to find relevant resources (NOTE: right now only website urls)
type Finder interface {
	FindResources(ctx context.Context, concepts []concepts.Concept) ([]Resource, error)
}

type YoutubeVideoFinder struct {
	crawler       *BasicWebCrawler
	baseSearchUrl string
}

const (
	youtubeSearchUrl = "https://www.youtube.com/results?search_query="
)

func NewYoutubeVideoFinder() (Finder, error) {
	crawler, err := NewBasicWebCrawler(youtubeSearchUrl)
	if err != nil {
		return nil, err
	}

	return &YoutubeVideoFinder{
		crawler:       crawler,
		baseSearchUrl: youtubeSearchUrl,
	}, nil
}

/**
* Starts a crawler to find relevant website links concurrently. Relevance is based on "concepts".
**/
func (f *YoutubeVideoFinder) FindResources(ctx context.Context, concepts []concepts.Concept) ([]Resource, error) {
	if concepts == nil || len(concepts) == 0 {
		return nil, fmt.Errorf("Require concepts to start search to find relevant youtube videos.")
	}

	// start up crawlers and find at least 5 relevant videos
	// TODO: use description for now but later need to formulate entire concepts and spin up
	// as many goroutines to crawl the searches that match the length of concepts

	fmt.Printf("\nconcepts: %+v\n\n", concepts)

	fmt.Println("Crawl Path at concepts description:", concepts[0].Description)
	resourceByte, err := f.crawler.CrawlPath(ctx, concepts[0].Description)

	if err != nil {
		fmt.Println("Error when trying to crawl url", err)
		return nil, err
	}

	// @TEST: For debugging crawled content
	debugPageContent(string(resourceByte))

	_, _ = parseHtml(resourceByte)

	return nil, nil
}

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

// Crawl fetches a webpage and returns its content
func (c *BasicWebCrawler) CrawlPath(ctx context.Context, path string) ([]byte, error) {

	// Resolve the URL properly
	resolvedURL, err := c.ResolvePath(path)
	if err != nil {
		return nil, err
	}

	// TODO: upgrade to resolved url
	fmt.Println("Crawling url at:", resolvedURL)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolvedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// set up appropriate headers to avoid being blocked by youtube
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// make request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %w", resolvedURL, err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code: %d for URL %s", resp.StatusCode, resolvedURL)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

// recursive dfs crawler
func parseHtml(htmlBinary []byte) (links []string, err error) {
	htmlNode, err := html.Parse(bytes.NewReader(htmlBinary))

	fmt.Printf("\n\nBody parsed from html: %+v\n\n\n", htmlNode)
	if err != nil {
		return nil, err
	}

	// walk through html tree
	fmt.Printf("\nStarting Html Node %+v\n\n", htmlNode)
	result := walkTree(htmlNode, make([]string, 0))

	fmt.Printf("\nFinal Crawled Links: %+v\n\n", result)

	return result, nil
}

// recursive walk function for parseHtml
func walkTree(node *html.Node, links []string) []string {
	// base case - end if nil
	if node == nil {
		return links
	}

	// using pre-order traversal, so "visit" node first
	// check if its an element tag

	if node.Type == html.ElementNode {
		// check all attributes for video IDs
		for _, attr := range node.Attr {
			if strings.Contains(attr.Val, "watch?v=") {
				links = append(links, attr.Val)
			}
		}
	}

	// traverse through all exisitng nested children
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		links = walkTree(child, links)
	}

	return links
}

/**
* Finds all the matching video ids from raw html string, parse them into json, then collect them into a slice.
**/
type VideoId struct {
	Url string `json:"url"`
}

func extractVideoIdsFromRawHtml(htmlContent string) []string {
	// check for the regex pattern: "url":"/watch?v=VIDEO_ID"

	watchPattern := `"url":"/watch\?v=([^"&]+)`
	regex := regexp.MustCompile(watchPattern)

	matches := regex.FindAllStringSubmatch(htmlContent, -1)
	videoIds := make([]string, len(matches))

	for index, match := range matches {
		fmt.Printf("\nmatch before: %+v\n\n", match)

		matchJson := fmt.Sprintf("{%s\"}", match[0])
		var videoId VideoId
		err := json.Unmarshal([]byte(matchJson), &videoId)
		videoIds[index] = videoId.Url

		if err != nil {
			fmt.Printf("unmarshal err: %s\n", err.Error())
		}
	}
	fmt.Printf("Parsed Video Ids: %+v", videoIds)

	return videoIds
}

// for debugging
func debugPageContent(htmlContent string) {
	// Look for title tag
	if strings.Contains(htmlContent, "<title>") {
		start := strings.Index(htmlContent, "<title>") + 7
		end := strings.Index(htmlContent[start:], "</title>")
		if end > 0 {
			title := htmlContent[start : start+end]
			fmt.Printf("Page title: %s\n", title)
		}
	}

	// extract video ids
	extractVideoIdsFromRawHtml(htmlContent)

	// look for video containers that might be empty
	hasVideoContainers := strings.Contains(htmlContent, "ytd-video-renderer") ||
		strings.Contains(htmlContent, "video-title") ||
		strings.Contains(htmlContent, "ytd-compact-video-renderer")
	fmt.Printf("Contains video containers: %t\n", hasVideoContainers)

	// count total links vs video links
	totalLinks := strings.Count(htmlContent, "href=")
	videoLinks := strings.Count(htmlContent, "watch?v=")
	fmt.Printf("Total links: %d, Video links: %d\n", totalLinks, videoLinks)

	// look for JavaScript indicators
	hasJavaScript := strings.Contains(htmlContent, "<script") ||
		strings.Contains(htmlContent, "application/json")
	fmt.Printf("Contains JavaScript: %t\n", hasJavaScript)
}
