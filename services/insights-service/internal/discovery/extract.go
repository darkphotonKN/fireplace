package discovery

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
)

// VideoId is the shape the watch-url regex match is reassembled into.
type VideoId struct {
	Url string `json:"url"`
}

// watchPattern matches the `"url":"/watch?v=VIDEO_ID` fragments YouTube embeds
// in the JSON blob on its search-results page.
var watchRegex = regexp.MustCompile(`"url":"/watch\?v=([^"&]+)`)

// extractVideoIdsFromRawHtml finds all matching watch paths in the raw HTML,
// parses each back through JSON, and collects them in page order.
func extractVideoIdsFromRawHtml(htmlContent string) []string {
	matches := watchRegex.FindAllStringSubmatch(htmlContent, -1)

	videoIds := make([]string, 0, len(matches))
	for _, match := range matches {
		matchJson := fmt.Sprintf("{%s\"}", match[0])

		var videoId VideoId
		if err := json.Unmarshal([]byte(matchJson), &videoId); err != nil {
			slog.Error("error unmarshaling video URL", "error", err)
			// skip over video
			continue
		}

		videoIds = append(videoIds, videoId.Url)
	}

	return videoIds
}
