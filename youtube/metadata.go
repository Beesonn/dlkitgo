package youtube

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

	result := YouTubeData{
		Type: contentType,
		ID:   id,
		URL:  url,
	}

	switch contentType {
	case "video", "shorts":
		err := t.getVideoInfo(id, &result)
		if err != nil {
			return result, err
		}
	case "playlist":
		err := t.getPlaylistInfo(id, &result)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

func (t *TubeService) getVideoInfo(videoID string, result *YouTubeData) error {
	innerTube := NewInnerTubeClient(t.Client)
	playerResponse, err := innerTube.GetPlayer(videoID)
	if err != nil {
		return fmt.Errorf("failed to get video info: %w", err)
	}

	if playability, ok := playerResponse["playabilityStatus"].(map[string]interface{}); ok {
		if status, ok := playability["status"].(string); ok && status != "OK" {
			reason, _ := playability["reason"].(string)
			return fmt.Errorf("video unavailable: %s (%s)", status, reason)
		}
	}

	videoDetails, ok := playerResponse["videoDetails"].(map[string]interface{})
	if !ok {
		return errors.New("invalid video details")
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

	if microformat, ok := playerResponse["microformat"].(map[string]interface{}); ok {
		if playerMicroformat, ok := microformat["playerMicroformatRenderer"].(map[string]interface{}); ok {
			if publishDate, ok := playerMicroformat["publishDate"].(string); ok {
				videoInfo.ReleaseDate = publishDate
			}
		}
	}

	result.Videos = []YouTubeVideoInfo{videoInfo}
	result.TotalVideos = 1
	result.TotalPlaylists = 0

	return nil
}

func (t *TubeService) getPlaylistInfo(playlistID string, result *YouTubeData) error {
	innerTube := NewInnerTubeClient(t.Client)
	response, err := innerTube.BrowsePlaylist(playlistID, "")
	if err != nil {
		return fmt.Errorf("failed to get playlist info: %w", err)
	}

	contents, ok := response["contents"].(map[string]interface{})
	if !ok {
		return errors.New("invalid playlist response")
	}

	twoColumn, ok := contents["twoColumnBrowseResultsRenderer"].(map[string]interface{})
	if !ok {
		return errors.New("invalid playlist structure")
	}

	tabs, ok := twoColumn["tabs"].([]interface{})
	if len(tabs) == 0 {
		return errors.New("no tabs found")
	}

	firstTab, ok := tabs[0].(map[string]interface{})
	if !ok {
		return errors.New("invalid tab structure")
	}

	tabRenderer, ok := firstTab["tabRenderer"].(map[string]interface{})
	if !ok {
		return errors.New("invalid tab renderer")
	}

	if title, ok := tabRenderer["title"].(string); ok {
		result.Name = title
	}

	content, ok := tabRenderer["content"].(map[string]interface{})
	if !ok {
		return errors.New("invalid content structure")
	}

	sectionList, ok := content["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return errors.New("invalid section list")
	}

	contentsList, ok := sectionList["contents"].([]interface{})
	if len(contentsList) == 0 {
		return errors.New("no contents found")
	}

	itemSection, ok := contentsList[0].(map[string]interface{})
	if !ok {
		return errors.New("invalid item section")
	}

	itemSectionRenderer, ok := itemSection["itemSectionRenderer"].(map[string]interface{})
	if !ok {
		return errors.New("invalid item section renderer")
	}

	items, ok := itemSectionRenderer["contents"].([]interface{})
	if !ok {
		return errors.New("no items found")
	}

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if playlistVideoRenderer, ok := itemMap["playlistVideoRenderer"].(map[string]interface{}); ok {
			video := extractPlaylistVideoFromRenderer(playlistVideoRenderer)
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

	result.TotalVideos = len(result.Playlist)
	result.TotalPlaylists = 1

	if result.Name == "" {
		result.Name = "YouTube Playlist"
	}

	return nil
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

func extractPlaylistVideoFromRenderer(renderer map[string]interface{}) YouTubeVideoInfo {
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
