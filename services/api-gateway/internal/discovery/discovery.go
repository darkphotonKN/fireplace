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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/concepts"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/constants"
	"github.com/darkphotonKN/fireplace/services/api-gateway/internal/logger"
	"golang.org/x/net/html"
)

type Resource struct {
	Title       string                 `json:"title"`
	URL         string                 `json:"url"`
	Source      string                 `json:"source"`
	Type        constants.ResourceType `json:"type"`
	Description string                 `json:"description"`
}

// a discovery finder needs to be able to find relevant resources
// NOTE: right now this is only website urls
type Finder interface {
	FindResources(ctx context.Context, concepts []concepts.Concept) ([]Resource, error)
}

type YoutubeVideoFinder struct {
	crawler       *BasicWebCrawler
	baseSearchUrl string
	wg            *sync.WaitGroup
	mu            *sync.Mutex
}

const (
	youtubeSearchUrl = "https://www.youtube.com/results?search_query="
	youtubeUrl       = "https://www.youtube.com"
)

func NewYoutubeVideoFinder() (Finder, error) {
	crawler, err := NewBasicWebCrawler(youtubeSearchUrl)
	if err != nil {
		return nil, err
	}

	return &YoutubeVideoFinder{
		crawler:       crawler,
		baseSearchUrl: youtubeSearchUrl,
		wg:            &sync.WaitGroup{},
	}, nil
}

/**
* Starts a crawler to find relevant website links concurrently. Relevance is based on "concepts".
**/

type CrawlResult struct {
	index   int
	videoID string
	err     error
}

func (f *YoutubeVideoFinder) FindResources(ctx context.Context, concepts []concepts.Concept) ([]Resource, error) {
	var wg sync.WaitGroup
	conceptsLength := len(concepts)

	// channel for holding concurrent results and errors
	crawlResultCh := make(chan CrawlResult)

	if concepts == nil || conceptsLength == 0 {
		return nil, fmt.Errorf("Require concepts to start search to find relevant youtube videos.")
	}

	// start up crawlers and find at least 3 relevant videos
	// TODO: use description for now but later need to formulate entire concepts and spin up
	// as many goroutines to crawl the searches that match the length of concepts

	logger.Debug("Starting resource discovery", "concepts", concepts)

	wg.Add(conceptsLength)

	// loop and crawl all resources concurrently
	for index, concept := range concepts {
		go func() {
			defer wg.Done()

			// crawls youtube search results + decription
			resourceByte, err := f.crawler.CrawlPath(ctx, concept.Description)

			if err != nil {
				crawlResultCh <- CrawlResult{
					index: index,
					err:   err,
				}
			} else {
				// TODO: update to capture and parse via ai insights multiple different crawl results and make comparisons for the best

				// grab the first one
				extractedVideos := extractVideoIdsFromRawHtml(string(resourceByte))

				if len(extractedVideos) == 0 || extractedVideos == nil {
					logger.Error("Error extracting video IDs from HTML")
					crawlResultCh <- CrawlResult{
						err: fmt.Errorf("Error occured when extracting video ids from raw htmls, no result could be extracted."),
					}
					return
				}

				singleExtractedVideo := extractedVideos[0]

				crawlResultCh <- CrawlResult{
					index:   index,
					videoID: singleExtractedVideo,
				}
			}
		}()
	}

	crawledResults := make([]CrawlResult, conceptsLength)

	// to stop channel when done
	go func() {
		wg.Wait()
		close(crawlResultCh)
	}()

	// parse crawl results from channel
	for crawlResult := range crawlResultCh {
		logger.Debug("Received crawl result", "result", crawlResult)
		if crawlResult.err != nil {
			logger.Error("Error crawling for concept", "error", crawlResult.err)
			continue
		}

		crawledResults[crawlResult.index] = crawlResult
	}

	// @TEST: For debugging crawled content
	// debugPageContent(string(resourceByte))

	// NOTE: no recursive walk for now, as vidoes can't be found in the html DOM nodes
	// _, _ = parseHtml(resourceByte)

	// create 3 resources from them
	resources := make([]Resource, conceptsLength)

	errCount := 0
	for index, crawlResult := range crawledResults {
		if crawlResult.err != nil {
			resources[index] = Resource{
				Title:       "No relevant video found",
				Description: "Recommended Video for " + concepts[index].Description,
				URL:         "",
			}

			errCount++
		} else {
			resources[index] = Resource{
				Title:       "Video " + strconv.Itoa(index+1),
				Description: "Recommended Video for " + concepts[index].Description,
				URL:         youtubeUrl + crawlResult.videoID,
			}
		}
	}

	// return one general error if all errored
	allErrored := errCount == conceptsLength
	if allErrored {
		return nil, fmt.Errorf("No results could crawled.")
	}

	return resources, nil
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
	logger.Debug("Crawling URL", "url", resolvedURL)

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

	logger.Debug("HTML body parsed", "node", htmlNode)
	if err != nil {
		return nil, err
	}

	// walk through html tree
	logger.Debug("Starting HTML tree walk", "node", htmlNode)
	result := walkTree(htmlNode, make([]string, 0))

	logger.Debug("Final crawled links", "links", result)

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
		// fmt.Printf("\nmatch before: %+v\n\n", match)

		matchJson := fmt.Sprintf("{%s\"}", match[0])

		var videoId VideoId
		err := json.Unmarshal([]byte(matchJson), &videoId)

		if err != nil {
			logger.Error("Error unmarshaling video URL", "error", err)
			// skip over video
			continue
		}

		videoIds[index] = videoId.Url
	}

	return videoIds
}

// NOTE: only for debugging
func debugPageContent(htmlContent string) {
	// Look for title tag
	if strings.Contains(htmlContent, "<title>") {
		start := strings.Index(htmlContent, "<title>") + 7
		end := strings.Index(htmlContent[start:], "</title>")
		if end > 0 {
			title := htmlContent[start : start+end]
			logger.Debug("Page title found", "title", title)
		}
	}

	// extract video ids
	extractVideoIdsFromRawHtml(htmlContent)

	// look for video containers that might be empty
	hasVideoContainers := strings.Contains(htmlContent, "ytd-video-renderer") ||
		strings.Contains(htmlContent, "video-title") ||
		strings.Contains(htmlContent, "ytd-compact-video-renderer")
	logger.Debug("Video containers check", "hasContainers", hasVideoContainers)

	// count total links vs video links
	totalLinks := strings.Count(htmlContent, "href=")
	videoLinks := strings.Count(htmlContent, "watch?v=")
	logger.Debug("Link counts", "totalLinks", totalLinks, "videoLinks", videoLinks)

	// look for JavaScript indicators
	hasJavaScript := strings.Contains(htmlContent, "<script") ||
		strings.Contains(htmlContent, "application/json")
	logger.Debug("JavaScript check", "hasJavaScript", hasJavaScript)
}
