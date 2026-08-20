package insights

import (
	"context"

	"github.com/darkphotonKN/fireplace/services/insights-service/internal/discovery"
)

// resourceFinder is the slice of the discovery package this adapter consumes.
type resourceFinder interface {
	FindResources(ctx context.Context, searchTerms []string) ([]discovery.Resource, error)
}

// DiscoveryVideoFinder adapts the discovery package to the VideoFinder seam,
// keeping discovery's Resource shape out of the insights domain.
type DiscoveryVideoFinder struct {
	finder resourceFinder
}

func NewDiscoveryVideoFinder(finder resourceFinder) *DiscoveryVideoFinder {
	return &DiscoveryVideoFinder{finder: finder}
}

func (f *DiscoveryVideoFinder) FindVideos(ctx context.Context, searchTerms []string) ([]Video, error) {
	resources, err := f.finder.FindResources(ctx, searchTerms)
	if err != nil {
		return nil, err
	}

	videos := make([]Video, 0, len(resources))
	for _, r := range resources {
		videos = append(videos, Video{
			Title:       r.Title,
			URL:         r.URL,
			Source:      r.Source,
			Type:        string(r.Type),
			Description: r.Description,
		})
	}
	return videos, nil
}
