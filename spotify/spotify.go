package spotify

import (
	"errors"
	"net/http"
)

type TrackSource struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
	Image  string `json:"image"`
	URL    string `json:"url"` 
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
		return StreamResult{}, errors.New("URL or ID not found")
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

	for _, track := range tracks {
		streamURL := ""

		for _, provider := range s.Providers {
			u, err := provider.Stream(track.URL)
			if err == nil && u != "" {
				streamURL = u
				break 
			}
		}

		result.Source = append(result.Source, TrackSource{
			Title:  track.Name,
			Artist: track.Artist,
			Image:  info.Image,
			URL:    streamURL, 
		})
	}

	return result, nil
}