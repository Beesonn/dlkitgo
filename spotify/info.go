package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"errors"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type TrackInfo struct {
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	PreviewURL string `json:"preview_url"`
	URL        string `json:"url"`
}

type SpotifyData struct {
	Type       string      `json:"type"`
	Name       string      `json:"name"`
	Artist     string      `json:"artist"`
	SpotifyID  string      `json:"spotify_id"`
	URL        string      `json:"url"` 
	Image      string      `json:"image"`
	PreviewURL string      `json:"preview_url"`
	Tracks     []TrackInfo `json:"tracks"`
}

type LdJson struct {
	Context   string          `json:"@context,omitempty"`
	Type      string          `json:"@type"`
	Name      string          `json:"name"`
	Image     []string        `json:"image"`
	ByArtist  json.RawMessage `json:"byArtist"`
	Audio     struct {
		ContentURL string `json:"contentUrl"`
	} `json:"audio"`
	Track []struct {
		ItemListElement []struct { 
			Item struct {
				Name       string          `json:"name"`
				PreviewURL string          `json:"previewUrl"`
				URL        string          `json:"url"`
				ByArtist   json.RawMessage `json:"byArtist"`
				Audio      struct {
					ContentURL string `json:"contentUrl"`
				} `json:"audio"`
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

	// Extract image from meta og:image
	if imgSel := doc.Find(`meta[property="og:image"]`); imgSel.Length() > 0 {
		if img, exists := imgSel.Attr("content"); exists {
			data.Image = img
		}
	}

	// Also check for __NEXT_DATA__ on main page as fallback
	doc.Find("script#__NEXT_DATA__").Each(func(i int, s *goquery.Selection) {
		ParseNextData(s.Text(), &data)
	})

	doc.Find("script[type='application/ld+json']").Each(func(i int, s *goquery.Selection) {
		var ldjsons []json.RawMessage
		if err := json.Unmarshal([]byte(s.Text()), &ldjsons); err != nil {
			// If not array, try as single object
			var singleLd LdJson
			if json.Unmarshal([]byte(s.Text()), &singleLd) == nil {
				ProcessLdJson(&singleLd, &data)
			}
			return
		}

		for _, raw := range ldjsons {
			var ld LdJson
			if err := json.Unmarshal(raw, &ld); err == nil {
				ProcessLdJson(&ld, &data)
			}
		}
	})

	// Fallback for playlists/albums, and now also tracks if LD incomplete
	if (data.Type == "playlist" || data.Type == "album" || data.Type == "track") && len(data.Tracks) == 0 && data.Name == "" {
		FetchEmbedData(&data, s.Client)
	}

	return data, nil
}

func ProcessLdJson(ld *LdJson, data *SpotifyData) {
	if ld.Type == "MusicRecording" {
		data.Name = ld.Name
		if len(ld.Image) > 0 {
			data.Image = ld.Image[0]
		}
		data.PreviewURL = ld.Audio.ContentURL
		data.Artist = ParseArtistRaw(ld.ByArtist)

		// For track, add as single track
		if data.Type == "track" {
			data.Tracks = append(data.Tracks, TrackInfo{
				Name:       ld.Name,
				Artist:     data.Artist,
				PreviewURL: ld.Audio.ContentURL,
				URL:        data.URL,
			})
		}
	} else if ld.Type == "MusicAlbum" && data.Type == "album" {
		if len(ld.Track) > 0 {
			for _, t := range ld.Track {
				for _, elem := range t.ItemListElement {
					if elem.Item.Name != "" {
						data.Tracks = append(data.Tracks, TrackInfo{
							Name:       elem.Item.Name,
							Artist:     ParseArtistRaw(elem.Item.ByArtist),
							PreviewURL: elem.Item.Audio.ContentURL,
							URL:        elem.Item.URL,
						})
					}
				}
			}
		}
	}
}

func FetchEmbedData(data *SpotifyData, client *http.Client) {
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

	doc.Find("script#amino-initial-data").Each(func(i int, s *goquery.Selection) {
		ParseNextData(s.Text(), data)
	})
	doc.Find("script#__NEXT_DATA__").Each(func(i int, s *goquery.Selection) {
		ParseNextData(s.Text(), data)
	})
}

func ParseNextData(jsonStr string, data *SpotifyData) {
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
	
	// Populate main fields for single entities (track, etc.)
	if data.Name == "" {
		if title, ok := entity["name"].(string); ok {
			data.Name = title
		} else if title, ok := entity["title"].(string); ok {
			data.Name = title
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

	trackList, ok := entity["trackList"].([]interface{})
	if ok {
		for _, t := range trackList {
			trackMap, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := trackMap["title"].(string)
			artist, _ := trackMap["subtitle"].(string)
			uri, _ := trackMap["uri"].(string) 
            
			var previewUrl string
			if audioPreview, ok := trackMap["audioPreview"].(map[string]interface{}); ok {
				if urlI, ok := audioPreview["url"].(string); ok {
					previewUrl = urlI
				}
			}

			parts := strings.Split(uri, ":")
			if len(parts) > 1 {
				trackId := parts[len(parts)-1]
				trackLink := fmt.Sprintf("https://open.spotify.com/track/%s", trackId)
				data.Tracks = append(data.Tracks, TrackInfo{
					Name:       name,
					Artist:     artist,
					PreviewURL: previewUrl,
					URL:        trackLink,
				})
			}
		}
	} else {
		// for single track
		if data.Type == "track" && data.Name != "" {
			uri, _ := entity["uri"].(string)
			parts := strings.Split(uri, ":")
			if len(parts) > 1 {
				trackId := parts[len(parts)-1]
				trackLink := fmt.Sprintf("https://open.spotify.com/track/%s", trackId)
				data.Tracks = append(data.Tracks, TrackInfo{
					Name:       data.Name,
					Artist:     data.Artist,
					PreviewURL: data.PreviewURL,
					URL:        trackLink,
				})
			}
		}
	}

	if len(data.Tracks) > 0 && data.Type == "track" && data.Name == "" {
		data.Name = data.Tracks[0].Name
		data.Artist = data.Tracks[0].Artist
		data.PreviewURL = data.Tracks[0].PreviewURL
		data.Image = "" 
	}
}

// join name's like artist1, artist 2.
func ParseArtistRaw(raw json.RawMessage) string {
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