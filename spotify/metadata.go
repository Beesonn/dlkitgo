package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type TrackInfo struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	PreviewURL  string `json:"preview_url"`
	URL         string `json:"url"`
	Duration    int    `json:"duration"`
	ReleaseDate string `json:"release_date"`
	Image       string `json:"image"`
}

type SpotifyData struct {
	Type        string      `json:"type"`
	Name        string      `json:"name"`
	Artist      string      `json:"artist"`
	SpotifyID   string      `json:"spotify_id"`
	URL         string      `json:"url"`
	Image       string      `json:"image"`
	PreviewURL  string      `json:"preview_url"`
	Tracks      []TrackInfo `json:"tracks"`
	Duration    int         `json:"duration,omitempty"`
	ReleaseDate string      `json:"release_date,omitempty"`
}

type LdJson struct {
	Context     string          `json:"@context,omitempty"`
	Type        string          `json:"@type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Image       []string        `json:"image"`
	ByArtist    json.RawMessage `json:"byArtist"`
	Audio       struct {
		ContentURL string `json:"contentUrl"`
		Duration   string `json:"duration,omitempty"`
	} `json:"audio"`
	Duration  string `json:"duration,omitempty"`
	Date      string `json:"datePublished,omitempty"`
	ReleaseOf struct {
		DatePublished string `json:"datePublished,omitempty"`
	} `json:"releaseOf,omitempty"`
	Track []struct {
		ItemListElement []struct {
			Item struct {
				Name       string          `json:"name"`
				PreviewURL string          `json:"previewUrl"`
				URL        string          `json:"url"`
				ByArtist   json.RawMessage `json:"byArtist"`
				Audio      struct {
					ContentURL string `json:"contentUrl"`
					Duration   string `json:"duration,omitempty"`
				} `json:"audio"`
				Duration string `json:"duration,omitempty"`
			} `json:"item"`
		} `json:"itemListElement"`
	} `json:"track"`
}

type ArtistObj struct {
	Name string `json:"name"`
}

func (s *SpotifyService) GetInfo(url string) (SpotifyData, error) {
	data := SpotifyData{
		Type:   "unknown",
		Tracks: []TrackInfo{},
		URL:    url,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := s.Client.Do(req)
	if err != nil {
		return data, errors.New("Invalid URL or ID")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return data, errors.New("Invalid URL or ID")
	}

	data.URL = resp.Request.URL.String()
	re := regexp.MustCompile(`/(track|album|playlist)/([a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(data.URL)

	if len(matches) == 3 {
		data.Type = matches[1]
		data.SpotifyID = matches[2]
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return data, errors.New("Invalid URL or ID")
	}

	if imgSel := doc.Find(`meta[property="og:image"]`); imgSel.Length() > 0 {
		if img, exists := imgSel.Attr("content"); exists {
			data.Image = img
		}
	}

	doc.Find("script#__NEXT_DATA__").Each(func(i int, sel *goquery.Selection) {
		s.ParseNextData(sel.Text(), &data)
	})

	doc.Find("script[type='application/ld+json']").Each(func(i int, sel *goquery.Selection) {
		var ldjsons []json.RawMessage
		if err := json.Unmarshal([]byte(sel.Text()), &ldjsons); err != nil {
			var singleLd LdJson
			if json.Unmarshal([]byte(sel.Text()), &singleLd) == nil {
				s.ProcessLdJson(&singleLd, &data)
			}
			return
		}

		for _, raw := range ldjsons {
			var ld LdJson
			if err := json.Unmarshal(raw, &ld); err == nil {
				s.ProcessLdJson(&ld, &data)
			}
		}
	})

	if (data.Type == "playlist" || data.Type == "album" || data.Type == "track") && len(data.Tracks) == 0 && data.Name == "" {
		s.FetchEmbedData(&data, s.Client)
	}

	return data, nil
}

func (s *SpotifyService) ProcessLdJson(ld *LdJson, data *SpotifyData) {
	if ld.Type == "MusicRecording" {
		data.Name = ld.Name
		if len(ld.Image) > 0 {
			data.Image = ld.Image[0]
		}
		data.PreviewURL = ld.Audio.ContentURL
		data.Artist = s.ParseArtistRaw(ld.ByArtist)

		if ld.Audio.Duration != "" {
			if dur, err := s.ParseDurationToSeconds(ld.Audio.Duration); err == nil {
				data.Duration = dur
			}
		}

		if ld.Description != "" {
			data.ReleaseDate = s.ExtractReleaseDate(ld.Description)
		}

		if data.Type == "track" {
			track := TrackInfo{
				Name:        ld.Name,
				Artist:      data.Artist,
				PreviewURL:  ld.Audio.ContentURL,
				URL:         data.URL,
				Duration:    data.Duration,
				ReleaseDate: data.ReleaseDate,
				Image:       data.Image,
			}

			if ld.Audio.Duration != "" {
				if dur, err := s.ParseDurationToSeconds(ld.Audio.Duration); err == nil {
					track.Duration = dur
				}
			}

			data.Tracks = append(data.Tracks, track)
		}
	} else if ld.Type == "MusicAlbum" && data.Type == "album" {
		if ld.ReleaseOf.DatePublished != "" {
			data.ReleaseDate = ld.ReleaseOf.DatePublished
		} else if ld.Date != "" {
			data.ReleaseDate = ld.Date
		}

		if len(ld.Track) > 0 {
			for _, t := range ld.Track {
				for _, elem := range t.ItemListElement {
					if elem.Item.Name != "" {
						var duration int
						if elem.Item.Audio.Duration != "" {
							if dur, err := s.ParseDurationToSeconds(elem.Item.Audio.Duration); err == nil {
								duration = dur
							}
						} else if elem.Item.Duration != "" {
							if dur, err := s.ParseDurationToSeconds(elem.Item.Duration); err == nil {
								duration = dur
							}
						}

						data.Tracks = append(data.Tracks, TrackInfo{
							Name:        elem.Item.Name,
							Artist:      s.ParseArtistRaw(elem.Item.ByArtist),
							PreviewURL:  elem.Item.Audio.ContentURL,
							URL:         elem.Item.URL,
							Duration:    duration,
							ReleaseDate: data.ReleaseDate,
							Image:       data.Image,
						})
					}
				}
			}
		}
	} else if ld.Type == "MusicPlaylist" && data.Type == "playlist" {
		data.Name = ld.Name
		if len(ld.Image) > 0 {
			data.Image = ld.Image[0]
		}
	}
}

func (s *SpotifyService) FetchEmbedData(data *SpotifyData, client *http.Client) {
	embedUrl := strings.Replace(data.URL, "/playlist/", "/embed/playlist/", 1)
	embedUrl = strings.Replace(embedUrl, "/album/", "/embed/album/", 1)
	embedUrl = strings.Replace(embedUrl, "/track/", "/embed/track/", 1)

	req, _ := http.NewRequest("GET", embedUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return
	}

	doc.Find("script#amino-initial-data").Each(func(i int, sel *goquery.Selection) {
		s.ParseNextData(sel.Text(), data)
	})
	doc.Find("script#__NEXT_DATA__").Each(func(i int, sel *goquery.Selection) {
		s.ParseNextData(sel.Text(), data)
	})
}

func (s *SpotifyService) ParseNextData(jsonStr string, data *SpotifyData) {
	var generic map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &generic); err != nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
		}
	}()

	props, ok := generic["props"].(map[string]interface{})
	if !ok {
		return
	}
	pageProps, ok := props["pageProps"].(map[string]interface{})
	if !ok {
		return
	}
	state, ok := pageProps["state"].(map[string]interface{})
	if !ok {
		return
	}
	d, ok := state["data"].(map[string]interface{})
	if !ok {
		return
	}
	entity, ok := d["entity"].(map[string]interface{})
	if !ok {
		return
	}

	// Get playlist/album/track name
	if data.Name == "" {
		if title, ok := entity["name"].(string); ok {
			data.Name = title
		} else if title, ok := entity["title"].(string); ok {
			data.Name = title
		}
	}

	// Get main image
	if data.Image == "" {
		if images, ok := entity["images"].([]interface{}); ok && len(images) > 0 {
			if imageMap, ok := images[0].(map[string]interface{}); ok {
				if imgUrl, ok := imageMap["url"].(string); ok {
					data.Image = imgUrl
				}
			}
		}
	}

	// Process based on type
	if data.Type == "track" {
		s.parseSingleTrackData(entity, data)
	} else if data.Type == "playlist" || data.Type == "album" {
		s.parsePlaylistOrAlbumData(entity, data)
	}
}

func (s *SpotifyService) parseSingleTrackData(entity map[string]interface{}, data *SpotifyData) {
	// Get artist for single track
	if data.Artist == "" {
		if artists, ok := entity["artists"].([]interface{}); ok {
			var artistNames []string
			for _, a := range artists {
				if artistMap, ok := a.(map[string]interface{}); ok {
					if name, ok := artistMap["name"].(string); ok {
						artistNames = append(artistNames, name)
					}
				}
			}
			if len(artistNames) > 0 {
				data.Artist = strings.Join(artistNames, ", ")
			}
		}
	}

	if data.Artist == "" {
		if subtitle, ok := entity["subtitle"].(string); ok {
			data.Artist = subtitle
		}
	}

	if data.PreviewURL == "" {
		if audioPreview, ok := entity["audioPreview"].(map[string]interface{}); ok {
			if urlI, ok := audioPreview["url"].(string); ok {
				data.PreviewURL = urlI
			}
		}
	}

	if durationMs, ok := entity["duration"].(float64); ok {
		data.Duration = int(durationMs) / 1000
	}

	if releaseDate, ok := entity["releaseDate"].(map[string]interface{}); ok {
		if dateStr, ok := releaseDate["isoString"].(string); ok {
			data.ReleaseDate = strings.Split(dateStr, "T")[0]
		}
	} else if dateStr, ok := entity["releaseDate"].(string); ok {
		data.ReleaseDate = dateStr
	}

	// Create single track entry
	if data.Name != "" {
		uri, _ := entity["uri"].(string)
		parts := strings.Split(uri, ":")
		if len(parts) > 1 {
			trackId := parts[len(parts)-1]
			trackLink := fmt.Sprintf("https://open.spotify.com/track/%s", trackId)
			data.Tracks = append(data.Tracks, TrackInfo{
				Name:        data.Name,
				Artist:      data.Artist,
				PreviewURL:  data.PreviewURL,
				URL:         trackLink,
				Duration:    data.Duration,
				ReleaseDate: data.ReleaseDate,
				Image:       data.Image,
			})
		}
	}
}

func (s *SpotifyService) parsePlaylistOrAlbumData(entity map[string]interface{}, data *SpotifyData) {
	// Get artist for playlist/album (usually the owner/creator)
	if data.Artist == "" {
		if subtitle, ok := entity["subtitle"].(string); ok {
			data.Artist = subtitle
		}
	}

	if releaseDate, ok := entity["releaseDate"].(map[string]interface{}); ok {
		if dateStr, ok := releaseDate["isoString"].(string); ok {
			data.ReleaseDate = strings.Split(dateStr, "T")[0]
		}
	} else if dateStr, ok := entity["releaseDate"].(string); ok {
		data.ReleaseDate = dateStr
	}

	// Process track list for playlists and albums
	if trackList, ok := entity["trackList"].([]interface{}); ok {
		for _, t := range trackList {
			trackMap, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			// Get track name - THIS WAS THE MAIN BUG
			name, _ := trackMap["title"].(string)
			if name == "" {
				// Fallback to different field name
				name, _ = trackMap["name"].(string)
			}

			// Get artist for this specific track
			artist, _ := trackMap["subtitle"].(string)
			if artist == "" {
				// Try to get artists from nested structure
				if artists, ok := trackMap["artists"].([]interface{}); ok {
					var artistNames []string
					for _, a := range artists {
						if artistMap, ok := a.(map[string]interface{}); ok {
							if artistName, ok := artistMap["name"].(string); ok {
								artistNames = append(artistNames, artistName)
							}
						}
					}
					if len(artistNames) > 0 {
						artist = strings.Join(artistNames, ", ")
					}
				}
			}

			uri, _ := trackMap["uri"].(string)

			var duration int
			if dur, ok := trackMap["duration"].(float64); ok {
				duration = int(dur) / 1000
			}

			var previewUrl string
			if audioPreview, ok := trackMap["audioPreview"].(map[string]interface{}); ok {
				if urlI, ok := audioPreview["url"].(string); ok {
					previewUrl = urlI
				}
			}

			// Get track image (album art for playlist tracks)
			var trackImage string
			if trackImages, ok := trackMap["images"].([]interface{}); ok && len(trackImages) > 0 {
				if imageMap, ok := trackImages[0].(map[string]interface{}); ok {
					if imgUrl, ok := imageMap["url"].(string); ok {
						trackImage = imgUrl
					}
				}
			}
			// Fallback to main image if track-specific image not found
			if trackImage == "" {
				trackImage = data.Image
			}

			parts := strings.Split(uri, ":")
			if len(parts) > 1 && name != "" {
				trackId := parts[len(parts)-1]
				trackLink := fmt.Sprintf("https://open.spotify.com/track/%s", trackId)
				data.Tracks = append(data.Tracks, TrackInfo{
					Name:        name, // Now correctly using the track's own name
					Artist:      artist,
					PreviewURL:  previewUrl,
					URL:         trackLink,
					Duration:    duration,
					ReleaseDate: data.ReleaseDate,
					Image:       trackImage,
				})
			}
		}
	}
}

func (s *SpotifyService) ParseArtistRaw(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var artists []ArtistObj
	if err := json.Unmarshal(raw, &artists); err == nil {
		names := []string{}
		for _, a := range artists {
			names = append(names, a.Name)
		}
		return strings.Join(names, ", ")
	}

	var artist ArtistObj
	if err := json.Unmarshal(raw, &artist); err == nil {
		return artist.Name
	}

	return ""
}

func (s *SpotifyService) ParseDurationToSeconds(durationStr string) (int, error) {
	if !strings.HasPrefix(durationStr, "PT") {
		return 0, errors.New("invalid duration format")
	}

	durationStr = durationStr[2:]

	var totalSeconds float64

	if idx := strings.Index(durationStr, "H"); idx != -1 {
		var hours float64
		fmt.Sscanf(durationStr[:idx], "%f", &hours)
		totalSeconds += hours * 3600
		durationStr = durationStr[idx+1:]
	}

	if idx := strings.Index(durationStr, "M"); idx != -1 {
		var minutes float64
		fmt.Sscanf(durationStr[:idx], "%f", &minutes)
		totalSeconds += minutes * 60
		durationStr = durationStr[idx+1:]
	}

	if idx := strings.Index(durationStr, "S"); idx != -1 {
		var seconds float64
		fmt.Sscanf(durationStr[:idx], "%f", &seconds)
		totalSeconds += seconds
	}

	return int(totalSeconds), nil
}

func (s *SpotifyService) ExtractReleaseDate(description string) string {
	patterns := []string{
		`Released (\d{4}-\d{2}-\d{2})`,
		`(\d{4}-\d{2}-\d{2})`,
		`(\d{4})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if match := re.FindStringSubmatch(description); match != nil {
			return match[1]
		}
	}
	return ""
}
