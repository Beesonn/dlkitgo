package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

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
				Duration      string `json:"duration,omitempty"`
				DatePublished string `json:"datePublished,omitempty"`
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
		s.ProcessLdJsonScript(sel.Text(), &data)
	})

	if (data.Type == "playlist" || data.Type == "album" || data.Type == "track") && len(data.Tracks) == 0 && data.Name == "" {
		s.FetchEmbedData(&data)
	}

	if data.Type == "playlist" || data.Type == "album" {
		s.EnhanceTrackData(&data)
	}

	return data, nil
}

func (s *SpotifyService) EnhanceTrackData(data *SpotifyData) {
	if len(data.Tracks) == 0 {
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range data.Tracks {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			trackData, err := s.GetInfo(data.Tracks[idx].URL)
			if err != nil {
				return
			}

			mu.Lock()
			if trackData.Name != "" {
				data.Tracks[idx].Name = trackData.Name
			}
			if trackData.Artist != "" {
				data.Tracks[idx].Artist = trackData.Artist
			}
			if trackData.ReleaseDate != "" {
				data.Tracks[idx].ReleaseDate = trackData.ReleaseDate
			}
			if trackData.Image != "" {
				data.Tracks[idx].Image = trackData.Image
			}
			if trackData.PreviewURL != "" {
				data.Tracks[idx].PreviewURL = trackData.PreviewURL
			}
			if trackData.Duration > 0 {
				data.Tracks[idx].Duration = trackData.Duration
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
}

func (s *SpotifyService) ProcessLdJsonScript(scriptText string, data *SpotifyData) {
	var ldjsons []json.RawMessage
	if err := json.Unmarshal([]byte(scriptText), &ldjsons); err != nil {
		var singleLd LdJson
		if json.Unmarshal([]byte(scriptText), &singleLd) == nil {
			s.ProcessLdJson(&singleLd, data)
		}
		return
	}

	for _, raw := range ldjsons {
		var ld LdJson
		if err := json.Unmarshal(raw, &ld); err == nil {
			s.ProcessLdJson(&ld, data)
		}
	}
}

func (s *SpotifyService) ProcessLdJson(ld *LdJson, data *SpotifyData) {
	switch ld.Type {
	case "MusicRecording":
		s.ProcessMusicRecording(ld, data)
	case "MusicAlbum":
		if data.Type == "album" {
			s.ProcessMusicAlbum(ld, data)
		}
	case "MusicPlaylist":
		if data.Type == "playlist" {
			s.ProcessMusicPlaylist(ld, data)
		}
	}
}

func (s *SpotifyService) ProcessMusicRecording(ld *LdJson, data *SpotifyData) {
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
		s.AddTrackFromLdJson(ld, data)
	}
}

func (s *SpotifyService) AddTrackFromLdJson(ld *LdJson, data *SpotifyData) {
	duration := data.Duration
	if ld.Audio.Duration != "" {
		if dur, err := s.ParseDurationToSeconds(ld.Audio.Duration); err == nil {
			duration = dur
		}
	}

	track := TrackInfo{
		Name:        ld.Name,
		Artist:      data.Artist,
		PreviewURL:  ld.Audio.ContentURL,
		URL:         data.URL,
		Duration:    duration,
		ReleaseDate: data.ReleaseDate,
		Image:       data.Image,
	}

	data.Tracks = append(data.Tracks, track)
}

func (s *SpotifyService) ProcessMusicAlbum(ld *LdJson, data *SpotifyData) {
	if ld.ReleaseOf.DatePublished != "" {
		data.ReleaseDate = ld.ReleaseOf.DatePublished
	} else if ld.Date != "" {
		data.ReleaseDate = ld.Date
	}

	if len(ld.Track) > 0 {
		for _, t := range ld.Track {
			for _, elem := range t.ItemListElement {
				if elem.Item.Name != "" {
					s.AddAlbumTrack(elem.Item, data)
				}
			}
		}
	}
}

func (s *SpotifyService) AddAlbumTrack(item struct {
	Name       string          `json:"name"`
	PreviewURL string          `json:"previewUrl"`
	URL        string          `json:"url"`
	ByArtist   json.RawMessage `json:"byArtist"`
	Audio      struct {
		ContentURL string `json:"contentUrl"`
		Duration   string `json:"duration,omitempty"`
	} `json:"audio"`
	Duration      string `json:"duration,omitempty"`
	DatePublished string `json:"datePublished,omitempty"`
}, data *SpotifyData) {
	var duration int
	if item.Audio.Duration != "" {
		if dur, err := s.ParseDurationToSeconds(item.Audio.Duration); err == nil {
			duration = dur
		}
	} else if item.Duration != "" {
		if dur, err := s.ParseDurationToSeconds(item.Duration); err == nil {
			duration = dur
		}
	}

	releaseDate := data.ReleaseDate
	if item.DatePublished != "" {
		releaseDate = item.DatePublished
	}

	data.Tracks = append(data.Tracks, TrackInfo{
		Name:        item.Name,
		Artist:      s.ParseArtistRaw(item.ByArtist),
		PreviewURL:  item.Audio.ContentURL,
		URL:         item.URL,
		Duration:    duration,
		ReleaseDate: releaseDate,
		Image:       data.Image,
	})
}

func (s *SpotifyService) ProcessMusicPlaylist(ld *LdJson, data *SpotifyData) {
	data.Name = ld.Name
	if len(ld.Image) > 0 {
		data.Image = ld.Image[0]
	}
}

func (s *SpotifyService) FetchEmbedData(data *SpotifyData) {
	embedUrl := strings.Replace(data.URL, "/playlist/", "/embed/playlist/", 1)
	embedUrl = strings.Replace(embedUrl, "/album/", "/embed/album/", 1)
	embedUrl = strings.Replace(embedUrl, "/track/", "/embed/track/", 1)

	req, _ := http.NewRequest("GET", embedUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	resp, err := s.Client.Do(req)
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

	entity := s.ExtractEntity(generic)
	if entity == nil {
		return
	}

	s.ExtractBasicData(entity, data)

	if data.Type == "track" {
		s.ParseSingleTrackData(entity, data)
	} else if data.Type == "playlist" || data.Type == "album" {
		s.ParsePlaylistOrAlbumData(entity, data)
	}
}

func (s *SpotifyService) ExtractEntity(generic map[string]interface{}) map[string]interface{} {
	props, ok := generic["props"].(map[string]interface{})
	if !ok {
		return s.ExtractEntityAlternative(generic)
	}
	pageProps, ok := props["pageProps"].(map[string]interface{})
	if !ok {
		return s.ExtractEntityAlternative(generic)
	}
	state, ok := pageProps["state"].(map[string]interface{})
	if !ok {
		return s.ExtractEntityAlternative(generic)
	}
	d, ok := state["data"].(map[string]interface{})
	if !ok {
		return s.ExtractEntityAlternative(generic)
	}
	entity, ok := d["entity"].(map[string]interface{})
	if !ok {
		return s.ExtractEntityAlternative(generic)
	}

	return entity
}

func (s *SpotifyService) ExtractEntityAlternative(generic map[string]interface{}) map[string]interface{} {
	if data, ok := generic["data"].(map[string]interface{}); ok {
		if entity, ok := data["entity"].(map[string]interface{}); ok {
			return entity
		}
	}

	if props, ok := generic["props"].(map[string]interface{}); ok {
		if data, ok := props["data"].(map[string]interface{}); ok {
			if entity, ok := data["entity"].(map[string]interface{}); ok {
				return entity
			}
		}
	}

	return nil
}

func (s *SpotifyService) ExtractBasicData(entity map[string]interface{}, data *SpotifyData) {
	if data.Name == "" {
		if title, ok := entity["name"].(string); ok {
			data.Name = title
		} else if title, ok := entity["title"].(string); ok {
			data.Name = title
		}
	}

	if data.Image == "" {
		if images, ok := entity["images"].([]interface{}); ok && len(images) > 0 {
			if imageMap, ok := images[0].(map[string]interface{}); ok {
				if imgUrl, ok := imageMap["url"].(string); ok {
					data.Image = imgUrl
				}
			}
		}
	}
}

func (s *SpotifyService) ParseSingleTrackData(entity map[string]interface{}, data *SpotifyData) {
	s.ExtractTrackArtist(entity, data)
	s.ExtractTrackPreviewURL(entity, data)
	s.ExtractTrackDuration(entity, data)
	s.ExtractTrackReleaseDate(entity, data)

	if data.Image == "" {
		if album, ok := entity["album"].(map[string]interface{}); ok {
			if images, ok := album["images"].([]interface{}); ok && len(images) > 0 {
				if imageMap, ok := images[0].(map[string]interface{}); ok {
					if imgUrl, ok := imageMap["url"].(string); ok {
						data.Image = imgUrl
					}
				}
			}
		}
	}

	s.CreateSingleTrackEntry(entity, data)
}

func (s *SpotifyService) ExtractTrackArtist(entity map[string]interface{}, data *SpotifyData) {
	if data.Artist == "" {
		if artists, ok := entity["artists"].([]interface{}); ok {
			artistNames := s.ExtractArtistNames(artists)
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
}

func (s *SpotifyService) ExtractArtistNames(artists []interface{}) []string {
	var names []string
	for _, a := range artists {
		if artistMap, ok := a.(map[string]interface{}); ok {
			if name, ok := artistMap["name"].(string); ok {
				names = append(names, name)
			}
		}
	}
	return names
}

func (s *SpotifyService) ExtractTrackPreviewURL(entity map[string]interface{}, data *SpotifyData) {
	if data.PreviewURL == "" {
		if audioPreview, ok := entity["audioPreview"].(map[string]interface{}); ok {
			if urlI, ok := audioPreview["url"].(string); ok {
				data.PreviewURL = urlI
			}
		}
	}
}

func (s *SpotifyService) ExtractTrackDuration(entity map[string]interface{}, data *SpotifyData) {
	if durationMs, ok := entity["duration"].(float64); ok {
		data.Duration = int(durationMs) / 1000
	}
}

func (s *SpotifyService) ExtractTrackReleaseDate(entity map[string]interface{}, data *SpotifyData) {
	if releaseDate, ok := entity["releaseDate"].(map[string]interface{}); ok {
		if dateStr, ok := releaseDate["isoString"].(string); ok {
			data.ReleaseDate = strings.Split(dateStr, "T")[0]
		}
	} else if dateStr, ok := entity["releaseDate"].(string); ok {
		data.ReleaseDate = dateStr
	}
}

func (s *SpotifyService) CreateSingleTrackEntry(entity map[string]interface{}, data *SpotifyData) {
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

func (s *SpotifyService) ParsePlaylistOrAlbumData(entity map[string]interface{}, data *SpotifyData) {
	s.ExtractPlaylistOrAlbumArtist(entity, data)
	s.ExtractPlaylistOrAlbumReleaseDate(entity, data)
	s.ProcessTrackList(entity, data)
}

func (s *SpotifyService) ExtractPlaylistOrAlbumArtist(entity map[string]interface{}, data *SpotifyData) {
	if data.Artist == "" {
		if subtitle, ok := entity["subtitle"].(string); ok {
			data.Artist = subtitle
		}
	}
}

func (s *SpotifyService) ExtractPlaylistOrAlbumReleaseDate(entity map[string]interface{}, data *SpotifyData) {
	if releaseDate, ok := entity["releaseDate"].(map[string]interface{}); ok {
		if dateStr, ok := releaseDate["isoString"].(string); ok {
			data.ReleaseDate = strings.Split(dateStr, "T")[0]
		}
	} else if dateStr, ok := entity["releaseDate"].(string); ok {
		data.ReleaseDate = dateStr
	}
}

func (s *SpotifyService) ProcessTrackList(entity map[string]interface{}, data *SpotifyData) {
	if trackList, ok := entity["trackList"].([]interface{}); ok {
		for _, t := range trackList {
			trackMap, ok := t.(map[string]interface{})
			if !ok {
				continue
			}

			s.ProcessSingleTrack(trackMap, data)
		}
	}
}

func (s *SpotifyService) ProcessSingleTrack(trackMap map[string]interface{}, data *SpotifyData) {
	name := s.ExtractTrackName(trackMap)
	if name == "" {
		return
	}

	artist := s.ExtractTrackArtistFromMap(trackMap)
	uri, _ := trackMap["uri"].(string)
	duration := s.ExtractTrackDurationFromMap(trackMap)
	previewUrl := s.ExtractTrackPreviewURLFromMap(trackMap)

	parts := strings.Split(uri, ":")
	if len(parts) > 1 {
		trackId := parts[len(parts)-1]
		trackLink := fmt.Sprintf("https://open.spotify.com/track/%s", trackId)
		data.Tracks = append(data.Tracks, TrackInfo{
			Name:        name,
			Artist:      artist,
			PreviewURL:  previewUrl,
			URL:         trackLink,
			Duration:    duration,
			ReleaseDate: data.ReleaseDate,
			Image:       data.Image,
		})
	}
}

func (s *SpotifyService) ExtractTrackName(trackMap map[string]interface{}) string {
	name, _ := trackMap["title"].(string)
	if name == "" {
		name, _ = trackMap["name"].(string)
	}
	return name
}

func (s *SpotifyService) ExtractTrackArtistFromMap(trackMap map[string]interface{}) string {
	artist, _ := trackMap["subtitle"].(string)
	if artist == "" {
		if artists, ok := trackMap["artists"].([]interface{}); ok {
			artistNames := s.ExtractArtistNames(artists)
			if len(artistNames) > 0 {
				artist = strings.Join(artistNames, ", ")
			}
		}
	}
	return artist
}

func (s *SpotifyService) ExtractTrackDurationFromMap(trackMap map[string]interface{}) int {
	var duration int
	if dur, ok := trackMap["duration"].(float64); ok {
		duration = int(dur) / 1000
	}
	return duration
}

func (s *SpotifyService) ExtractTrackPreviewURLFromMap(trackMap map[string]interface{}) string {
	var previewUrl string
	if audioPreview, ok := trackMap["audioPreview"].(map[string]interface{}); ok {
		if urlI, ok := audioPreview["url"].(string); ok {
			previewUrl = urlI
		}
	}
	return previewUrl
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
