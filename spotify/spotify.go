package spotify

import (
	"errors"
	"net/http"
	"sync"
)

type TrackSource struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Image       string `json:"image"`
	URL         string `json:"url"`
	Duration    int    `json:"duration"`
	ReleaseDate string `json:"release_date"`
}

type StreamResult struct {
	URL    string        `json:"url"`
	ID     string        `json:"id"`
	Type   string        `json:"type"`
	Source []TrackSource `json:"source"`
}

type SpotifyService struct {
	Client    *http.Client
	Providers []Provider
}

func NewSpotify(client *http.Client) *SpotifyService {
	return &SpotifyService{
		Client:    client,
		Providers: DefaultProviders(client),
	}
}

func (s *SpotifyService) Stream(url string) (StreamResult, error) {
	if url == "" {
		return StreamResult{}, errors.New("url or id cannot be empty")
	}

	info, err := s.GetInfo(url)
	if err != nil {
		return StreamResult{}, err
	}

	result := StreamResult{
		URL:  info.URL,
		ID:   info.SpotifyID,
		Type: info.Type,
	}

	var tracks []TrackInfo
	if info.Type == "track" && len(info.Tracks) > 0 {
		tracks = info.Tracks
	} else {
		tracks = info.Tracks
	}

	if len(tracks) == 0 {
		tracks = []TrackInfo{{Name: info.Name, Artist: info.Artist}}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sources := make([]TrackSource, 0, len(tracks))

	for _, track := range tracks {
		wg.Add(1)
		go func(t TrackInfo) {
			defer wg.Done()

			var streamURL string
			for _, provider := range s.Providers {
				u, err := provider.Stream(t.URL)
				if err == nil && u != "" {
					streamURL = u
					break
				}
			}

			if streamURL != "" {
				mu.Lock()
				sources = append(sources, TrackSource{
					Title:       t.Name,
					Artist:      t.Artist,
					Image:       t.Image,
					URL:         streamURL,
					ReleaseDate: t.ReleaseDate,
					Duration:    t.Duration,
				})
				mu.Unlock()
			}
		}(track)
	}

	wg.Wait()
	result.Source = sources

	return result, nil
}
