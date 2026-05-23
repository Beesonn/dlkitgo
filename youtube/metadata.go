package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type YouTubePlaylistInfo struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	Duration    int    `json:"duration"`
	ChannelName string `json:"channel_name"`
}

type YouTubeVideoInfo struct {
	Name        string `json:"name"`
	ChannelName string `json:"channel_name"`
	ChannelURL  string `json:"channel_url"`
	URL         string `json:"url"`
	Duration    int    `json:"duration"`
	ReleaseDate string `json:"release_date"`
	Image       string `json:"image"`
}

type YouTubeData struct {
	Type           string                `json:"type"`
	ID             string                `json:"id"`
	Name           string                `json:"name"`
	URL            string                `json:"url"`
	Image          string                `json:"image"`
	TotalPlaylists int                   `json:"total_playlists,omitempty"`
	TotalVideos    int                   `json:"total_videos,omitempty"`
	Playlist       []YouTubePlaylistInfo `json:"playlist,omitempty"`
	Videos         []YouTubeVideoInfo    `json:"videos,omitempty"`
}

func (t *TubeService) GetInfo(url string) (YouTubeData, error) {
	if url == "" {
		return YouTubeData{}, errors.New("URL cannot be empty")
	}

	contentType, id := detectYouTubeType(url)
	if contentType == "" {
		return YouTubeData{}, errors.New("unsupported YouTube URL (only video, playlist, or shorts)")
	}

	if strings.Contains(url, "&si=") {
		url = strings.Split(url, "&si=")[0]
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return YouTubeData{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.Client.Do(req)
	if err != nil {
		return YouTubeData{}, errors.New("failed to fetch YouTube page")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return YouTubeData{}, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return YouTubeData{}, errors.New("failed to read response body")
	}
	html := string(body)

	result := YouTubeData{
		Type: contentType,
		ID:   id,
		URL:  url,
	}

	switch contentType {
	case "video", "shorts":
		err = t.parseVideoPage(html, &result)
	case "playlist":
		err = t.parsePlaylistPage(html, &result)
	}

	if err != nil {
		return result, err
	}

	return result, nil
}

func detectYouTubeType(url string) (typ string, id string) {
	videoPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`youtube\.com/shorts/([a-zA-Z0-9_-]{11})`),
	}
	for _, re := range videoPatterns {
		if matches := re.FindStringSubmatch(url); len(matches) > 1 {
			if strings.Contains(url, "/shorts/") {
				return "shorts", matches[1]
			}
			return "video", matches[1]
		}
	}
	playlistRe := regexp.MustCompile(`[&?]list=([a-zA-Z0-9_-]+)`)
	if matches := playlistRe.FindStringSubmatch(url); len(matches) > 1 {
		return "playlist", matches[1]
	}
	return "", ""
}

func extractYtInitialPlayerResponse(html string) map[string]interface{} {
	patterns := []string{
		`var ytInitialPlayerResponse\s*=\s*({.*?});</script>`,
		`var ytInitialPlayerResponse\s*=\s*({.*?});\s*var`,
		`ytInitialPlayerResponse\s*=\s*({.*?});`,
		`"ytInitialPlayerResponse"\s*:\s*({.*?})\s*,\s*"`,
		`ytInitialPlayerResponse"\s*:\s*({.*?})\s*}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) >= 2 {
			var ytData map[string]interface{}
			if err := json.Unmarshal([]byte(matches[1]), &ytData); err == nil {
				return ytData
			}
		}
	}
	return nil
}

func (t *TubeService) parseVideoPage(html string, result *YouTubeData) error {
	playerResponse := extractYtInitialPlayerResponse(html)
	if playerResponse == nil {
		return errors.New("could not extract video data")
	}

	videoDetails, ok := playerResponse["videoDetails"].(map[string]interface{})
	if !ok {
		return errors.New("invalid video page structure")
	}

	if title, ok := videoDetails["title"].(string); ok {
		result.Name = title
	}

	result.Image = "https://img.youtube.com/vi/" + result.ID + "/maxresdefault.jpg"

	videoInfo := YouTubeVideoInfo{
		Name:  result.Name,
		URL:   result.URL,
		Image: result.Image,
	}

	if author, ok := videoDetails["author"].(string); ok {
		videoInfo.ChannelName = author
	}
	if channelId, ok := videoDetails["channelId"].(string); ok {
		videoInfo.ChannelURL = "https://www.youtube.com/channel/" + channelId
	}
	if lengthSeconds, ok := videoDetails["lengthSeconds"].(string); ok {
		videoInfo.Duration, _ = strconv.Atoi(lengthSeconds)
	}

	result.Videos = []YouTubeVideoInfo{videoInfo}
	result.TotalVideos = 1
	result.TotalPlaylists = 0

	return nil
}

func (t *TubeService) parsePlaylistPage(html string, result *YouTubeData) error {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		if title, exists := doc.Find(`meta[name="title"]`).Attr("content"); exists && title != "" {
			result.Name = title
		} else {
			doc.Find("title").Each(func(i int, s *goquery.Selection) {
				title := s.Text()
				if strings.Contains(title, " - YouTube") {
					title = strings.Replace(title, " - YouTube", "", 1)
					result.Name = title
				}
			})
		}

		if image, exists := doc.Find(`meta[property="og:image"]`).Attr("content"); exists {
			result.Image = image
		}
	}

	ytData := extractYtInitialData(html)
	if ytData != nil {
		if !t.extractPlaylistFromYtData(ytData, result) {
			t.extractPlaylistFromAlternativeJSON(html, result)
		}
	} else {
		t.extractPlaylistFromAlternativeJSON(html, result)
	}

	if len(result.Playlist) == 0 {
		t.extractPlaylistFromHTML(html, result)
	}

	return nil
}

func extractYtInitialData(html string) map[string]interface{} {
	patterns := []string{
		`var ytInitialData = ({.*?});</script>`,
		`var ytInitialData = ({.*?});\s*</script>`,
		`"ytInitialData"\s*:\s*({.*?})\s*,\s*"`,
		`ytInitialData"\s*:\s*({.*?})\s*}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) >= 2 {
			var ytData map[string]interface{}
			if err := json.Unmarshal([]byte(matches[1]), &ytData); err == nil {
				return ytData
			}
		}
	}
	return nil
}

func (t *TubeService) extractPlaylistFromYtData(ytData map[string]interface{}, result *YouTubeData) bool {
	contents, ok := ytData["contents"].(map[string]interface{})
	if !ok {
		return false
	}
	twoColumn, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return false
	}
	tabs, ok := twoColumn["tabs"].([]interface{})
	if len(tabs) == 0 {
		return false
	}
	firstTab, ok := tabs[0].(map[string]interface{})
	if !ok {
		return false
	}
	tabRenderer, ok := firstTab["tabRenderer"].(map[string]interface{})
	if !ok {
		return false
	}
	content, ok := tabRenderer["content"].(map[string]interface{})
	if !ok {
		return false
	}
	sectionList, ok := content["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return false
	}
	contentsList, ok := sectionList["contents"].([]interface{})
	if len(contentsList) == 0 {
		return false
	}

	for _, item := range contentsList {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		var playlistRenderer map[string]interface{}

		if itemSection, ok := itemMap["itemSectionRenderer"].(map[string]interface{}); ok {
			if sectionContents, ok := itemSection["contents"].([]interface{}); ok && len(sectionContents) > 0 {
				if secItemMap, ok := sectionContents[0].(map[string]interface{}); ok {
					playlistRenderer, _ = secItemMap["playlistVideoListRenderer"].(map[string]interface{})
				}
			}
		}

		if playlistRenderer == nil {
			playlistRenderer, _ = itemMap["playlistVideoListRenderer"].(map[string]interface{})
		}

		if playlistRenderer == nil {
			continue
		}

		if total, ok := playlistRenderer["totalVideos"].(float64); ok {
			result.TotalVideos = int(total)
		} else if totalStr, ok := playlistRenderer["totalVideos"].(string); ok {
			result.TotalVideos, _ = strconv.Atoi(totalStr)
		}

		result.TotalPlaylists = 1

		playlistItems, ok := playlistRenderer["contents"].([]interface{})
		if !ok {
			continue
		}

		for _, videoItem := range playlistItems {
			videoItemMap, ok := videoItem.(map[string]interface{})
			if !ok {
				continue
			}
			videoRenderer, ok := videoItemMap["playlistVideoRenderer"].(map[string]interface{})
			if !ok {
				continue
			}

			video := t.extractVideoDetails(videoRenderer)
			if video.Name != "" {
				result.Playlist = append(result.Playlist, YouTubePlaylistInfo{
					URL:         video.URL,
					Name:        video.Name,
					Duration:    video.Duration,
					ChannelName: video.ChannelName,
				})

				result.Videos = append(result.Videos, video)
			}
		}
		return true
	}
	return false
}

func (t *TubeService) extractVideoDetails(renderer map[string]interface{}) YouTubeVideoInfo {
	video := YouTubeVideoInfo{}

	videoID, _ := renderer["videoId"].(string)
	if videoID != "" {
		video.URL = "https://www.youtube.com/watch?v=" + videoID
	}

	if title, ok := renderer["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				video.Name, _ = run["text"].(string)
			}
		} else if simpleText, ok := title["simpleText"].(string); ok {
			video.Name = simpleText
		}
	}

	if length, ok := renderer["lengthSeconds"].(string); ok {
		video.Duration, _ = strconv.Atoi(length)
	} else if lengthFloat, ok := renderer["lengthSeconds"].(float64); ok {
		video.Duration = int(lengthFloat)
	} else if duration, ok := renderer["duration"].(map[string]interface{}); ok {
		if simpleText, ok := duration["simpleText"].(string); ok {
			video.Duration = parseDurationString(simpleText)
		}
	} else if lengthText, ok := renderer["lengthText"].(map[string]interface{}); ok {
		if simpleText, ok := lengthText["simpleText"].(string); ok {
			video.Duration = parseDurationString(simpleText)
		} else if runs, ok := lengthText["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					video.Duration = parseDurationString(text)
				}
			}
		}
	}

	if shortByline, ok := renderer["shortBylineText"].(map[string]interface{}); ok {
		if runs, ok := shortByline["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				video.ChannelName, _ = run["text"].(string)
				if nav, ok := run["navigationEndpoint"].(map[string]interface{}); ok {
					if browse, ok := nav["browseEndpoint"].(map[string]interface{}); ok {
						if cid, ok := browse["browseId"].(string); ok {
							video.ChannelURL = "https://www.youtube.com/channel/" + cid
						}
					}
				}
			}
		}
	}

	if video.ChannelName == "" {
		if longByline, ok := renderer["longBylineText"].(map[string]interface{}); ok {
			if runs, ok := longByline["runs"].([]interface{}); ok && len(runs) > 0 {
				if run, ok := runs[0].(map[string]interface{}); ok {
					video.ChannelName, _ = run["text"].(string)
				}
			}
		}
	}

	if thumbnails, ok := renderer["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnails["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
			if last, ok := thumbs[len(thumbs)-1].(map[string]interface{}); ok {
				video.Image, _ = last["url"].(string)
			}
		}
	}

	if video.Image == "" && videoID != "" {
		video.Image = "https://img.youtube.com/vi/" + videoID + "/maxresdefault.jpg"
	}

	return video
}

func (t *TubeService) extractPlaylistFromAlternativeJSON(html string, result *YouTubeData) {
	patterns := []string{
		`"playlistVideoListRenderer"\s*:\s*({[^}]+(?:{[^}]*}[^}]*)+})`,
		`"contents"\s*:\s*\[\s*\{[^}]*"playlistVideoRenderer"\s*:\s*({[^}]+(?:{[^}]*}[^}]*)+})`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) > 1 {
				var renderer map[string]interface{}
				if err := json.Unmarshal([]byte(match[1]), &renderer); err == nil {
					video := t.extractVideoDetails(renderer)
					if video.Name != "" {
						result.Playlist = append(result.Playlist, YouTubePlaylistInfo{
							URL:         video.URL,
							Name:        video.Name,
							Duration:    video.Duration,
							ChannelName: video.ChannelName,
						})
						result.Videos = append(result.Videos, video)
					}
				}
			}
		}
		if len(result.Playlist) > 0 {
			result.TotalPlaylists = 1
			if result.TotalVideos == 0 {
				result.TotalVideos = len(result.Playlist)
			}
			break
		}
	}
}

func (t *TubeService) extractPlaylistFromHTML(html string, result *YouTubeData) {
	videoIDPattern := regexp.MustCompile(`"videoId":"([a-zA-Z0-9_-]{11})"`)
	matches := videoIDPattern.FindAllStringSubmatch(html, -1)

	titlePattern := regexp.MustCompile(`"title":"([^"]+)"`)
	titleMatches := titlePattern.FindAllStringSubmatch(html, -1)

	durationPattern := regexp.MustCompile(`"durationText":"([^"]+)"`)
	durationMatches := durationPattern.FindAllStringSubmatch(html, -1)

	channelPattern := regexp.MustCompile(`"ownerText":"([^"]+)"`)
	channelMatches := channelPattern.FindAllStringSubmatch(html, -1)

	seen := make(map[string]bool)
	videoCount := 0

	for i, match := range matches {
		if len(match) > 1 && !seen[match[1]] {
			seen[match[1]] = true
			videoURL := "https://www.youtube.com/watch?v=" + match[1]

			title := ""
			if i < len(titleMatches) && len(titleMatches[i]) > 1 {
				title = titleMatches[i][1]
			}

			duration := 0
			if i < len(durationMatches) && len(durationMatches[i]) > 1 {
				duration = parseDurationString(durationMatches[i][1])
			}

			channelName := ""
			if i < len(channelMatches) && len(channelMatches[i]) > 1 {
				channelName = channelMatches[i][1]
			}

			playlistInfo := YouTubePlaylistInfo{
				Name:        title,
				URL:         videoURL,
				Duration:    duration,
				ChannelName: channelName,
			}
			result.Playlist = append(result.Playlist, playlistInfo)

			videoInfo := YouTubeVideoInfo{
				Name:        title,
				ChannelName: channelName,
				URL:         videoURL,
				Duration:    duration,
				Image:       "https://img.youtube.com/vi/" + match[1] + "/maxresdefault.jpg",
			}
			result.Videos = append(result.Videos, videoInfo)

			videoCount++
		}
	}

	if result.TotalVideos == 0 && videoCount > 0 {
		result.TotalVideos = videoCount
	}
	if videoCount > 0 {
		result.TotalPlaylists = 1
	}
}

func parseDurationString(duration string) int {
	if duration == "" {
		return 0
	}

	parts := strings.Split(duration, ":")
	total := 0

	if len(parts) == 3 {
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.Atoi(parts[2])
		total = hours*3600 + minutes*60 + seconds
	} else if len(parts) == 2 {
		minutes, _ := strconv.Atoi(parts[0])
		seconds, _ := strconv.Atoi(parts[1])
		total = minutes*60 + seconds
	} else if len(parts) == 1 {
		total, _ = strconv.Atoi(parts[0])
	}

	return total
}
