// Package discovery turns AI-generated search terms into concrete learning
// resources by crawling YouTube's search results page and extracting video ids
// from the raw HTML.
//
// Ported from the api-gateway's internal/discovery during the strangler move of
// the insights domain. Crawl and extraction behaviour is unchanged; what was
// dropped is code the gateway never reached — the recursive DOM walk
// (parseHtml/walkTree, superseded by the regex extractor and only ever called
// from a commented-out line) and debugPageContent. The concept struct is gone
// too: only its Description was ever read, so this takes plain search terms.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
)

// ResourceType classifies a discovered resource.
type ResourceType string

const (
	TypeVideo   ResourceType = "video"
	TypeWebsite ResourceType = "website"
)

// Resource is a discovered learning resource.
type Resource struct {
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	Source      string       `json:"source"`
	Type        ResourceType `json:"type"`
	Description string       `json:"description"`
}

// Finder finds relevant resources for a set of search terms.
type Finder interface {
	FindResources(ctx context.Context, searchTerms []string) ([]Resource, error)
}

const (
	youtubeSearchUrl = "https://www.youtube.com/results?search_query="
	youtubeUrl       = "https://www.youtube.com"
)

type YoutubeVideoFinder struct {
	crawler       *BasicWebCrawler
	baseSearchUrl string
}

func NewYoutubeVideoFinder() (*YoutubeVideoFinder, error) {
	crawler, err := NewBasicWebCrawler(youtubeSearchUrl)
	if err != nil {
		return nil, err
	}

	return &YoutubeVideoFinder{
		crawler:       crawler,
		baseSearchUrl: youtubeSearchUrl,
	}, nil
}

type CrawlResult struct {
	index   int
	videoID string
	err     error
}

// FindResources crawls one YouTube search per term concurrently and takes the
// first video id off each result page.
func (f *YoutubeVideoFinder) FindResources(ctx context.Context, searchTerms []string) ([]Resource, error) {
	termCount := len(searchTerms)
	if termCount == 0 {
		return nil, fmt.Errorf("discovery: require search terms to find relevant youtube videos")
	}

	slog.DebugContext(ctx, "starting resource discovery", "search_terms", searchTerms)

	var wg sync.WaitGroup
	crawlResultCh := make(chan CrawlResult)

	wg.Add(termCount)

	for index, term := range searchTerms {
		go func(index int, term string) {
			defer wg.Done()

			// crawls the youtube search results page for this term
			resourceByte, err := f.crawler.CrawlPath(ctx, term)
			if err != nil {
				crawlResultCh <- CrawlResult{index: index, err: err}
				return
			}

			// TODO: update to capture and parse via ai insights multiple different crawl results and make comparisons for the best
			extractedVideos := extractVideoIdsFromRawHtml(string(resourceByte))
			if len(extractedVideos) == 0 {
				slog.ErrorContext(ctx, "error extracting video IDs from HTML", "search_term", term)
				crawlResultCh <- CrawlResult{
					index: index,
					err:   fmt.Errorf("discovery: no video ids could be extracted for %q", term),
				}
				return
			}

			// grab the first one
			crawlResultCh <- CrawlResult{index: index, videoID: extractedVideos[0]}
		}(index, term)
	}

	go func() {
		wg.Wait()
		close(crawlResultCh)
	}()

	crawledResults := make([]CrawlResult, termCount)
	for crawlResult := range crawlResultCh {
		crawledResults[crawlResult.index] = crawlResult
	}

	resources := make([]Resource, termCount)
	errCount := 0
	for index, crawlResult := range crawledResults {
		if crawlResult.err != nil {
			slog.ErrorContext(ctx, "error crawling for search term",
				"search_term", searchTerms[index], "error", crawlResult.err)

			resources[index] = Resource{
				Title:       "No relevant video found",
				Description: "Recommended Video for " + searchTerms[index],
				URL:         "",
			}
			errCount++
			continue
		}

		resources[index] = Resource{
			Title:       "Video " + strconv.Itoa(index+1),
			Description: "Recommended Video for " + searchTerms[index],
			URL:         youtubeUrl + crawlResult.videoID,
		}
	}

	// return one general error if all errored
	if errCount == termCount {
		return nil, fmt.Errorf("discovery: no results could be crawled")
	}

	return resources, nil
}
