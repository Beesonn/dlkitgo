package youtube

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type SearchResult struct {
	ID          string `json:"id"`
	URL         string `json:"url"`
	Image       string `json:"image"`
	Duration    int    `json:"duration"`
	Channel     string `json:"channel"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	ReleaseDate string `json:"release_date"`
}

type SearchResponse struct {
	Query        string         `json:"query"`
	Type         string         `json:"type"`
	Limit        int            `json:"limit"`
	TotalResults int            `json:"total_results"`
	Results      []SearchResult `json:"results"`
}

func (t *TubeService) Search(query string, limit int, searchType ...string) (*SearchResponse, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	maxResults := 20
	if limit > 0 {
		maxResults = limit
		if maxResults > 50 {
			maxResults = 50
		}
	}

	sType := "video"
	if len(searchType) > 0 && searchType[0] != "" {
		sType = searchType[0]
	}

	searchQuery := query
	switch sType {
	case "channel":
		searchQuery = query + " channel"
	case "playlist":
		searchQuery = query + " playlist"
	}

	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(searchQuery)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := t.Client.Do(req)
	if err != nil {
		return nil, errors.New("failed to connect to YouTube")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("YouTube error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	results := parseYouTubeResponse(string(body), query, sType, maxResults)

	for i := range results.Results {
		if results.Results[i].Type == "video" && results.Results[i].ID != "" {
			results.Results[i].ReleaseDate = fetchAbsoluteDate(t.Client, results.Results[i].ID)
		}
	}

	return results, nil
}

func fetchAbsoluteDate(client *http.Client, videoID string) string {
	videoURL := "https://www.youtube.com/watch?v=" + videoID
	req, _ := http.NewRequest("GET", videoURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	html := string(body)

	re := regexp.MustCompile(`"(?:uploadDate|publishDate)"\s*:\s*"([0-9]{4}-[0-9]{2}-[0-9]{2})`)
	match := re.FindStringSubmatch(html)
	if len(match) >= 2 {
		return match[1]
	}

	re = regexp.MustCompile(`<meta itemprop="(?:uploadDate|datePublished)" content="([0-9]{4}-[0-9]{2}-[0-9]{2})`)
	match = re.FindStringSubmatch(html)
	if len(match) >= 2 {
		return match[1]
	}

	return ""
}

func parseYouTubeResponse(html, query, searchType string, limit int) *SearchResponse {
	response := &SearchResponse{
		Query:        query,
		Type:         searchType,
		Limit:        limit,
		TotalResults: 0,
		Results:      []SearchResult{},
	}

	re := regexp.MustCompile(`var ytInitialData = ({.*?});</script>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		return response
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(matches[1]), &data); err != nil {
		return response
	}

	contents, _ := data["contents"].(map[string]interface{})
	twoColumn, _ := contents["twoColumnSearchResultsRenderer"].(map[string]interface{})
	primary, _ := twoColumn["primaryContents"].(map[string]interface{})
	sectionList, _ := primary["sectionListRenderer"].(map[string]interface{})
	contents2, _ := sectionList["contents"].([]interface{})
	if len(contents2) == 0 {
		return response
	}
	itemSection, _ := contents2[0].(map[string]interface{})
	itemSectionRenderer, _ := itemSection["itemSectionRenderer"].(map[string]interface{})
	items, _ := itemSectionRenderer["contents"].([]interface{})

	count := 0
	for _, item := range items {
		if count >= limit {
			break
		}
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		videoRenderer, hasVideo := itemMap["videoRenderer"].(map[string]interface{})
		if hasVideo {
			result := parseVideoRenderer(videoRenderer)
			if result.ID != "" {
				response.Results = append(response.Results, result)
				count++
			}
			continue
		}
		if searchType == "all" || searchType == "channel" {
			channelRenderer, _ := itemMap["channelRenderer"].(map[string]interface{})
			if channelRenderer != nil {
				result := parseChannelRenderer(channelRenderer)
				if result.ID != "" {
					response.Results = append(response.Results, result)
					count++
				}
			}
		}
		if searchType == "all" || searchType == "playlist" {
			playlistRenderer, _ := itemMap["playlistRenderer"].(map[string]interface{})
			if playlistRenderer != nil {
				result := parsePlaylistRenderer(playlistRenderer)
				if result.ID != "" {
					response.Results = append(response.Results, result)
					count++
				}
			}
		}
	}
	response.TotalResults = len(response.Results)
	return response
}

func parseVideoRenderer(video map[string]interface{}) SearchResult {
	result := SearchResult{Type: "video"}
	if videoId, ok := video["videoId"].(string); ok {
		result.ID = videoId
		result.URL = "https://www.youtube.com/watch?v=" + videoId
	}
	if title, ok := video["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Name = text
				}
			}
		}
	}
	if owner, ok := video["ownerText"].(map[string]interface{}); ok {
		if runs, ok := owner["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Channel = text
				}
			}
		}
	}
	if thumbnails, ok := video["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnails["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
			if thumb, ok := thumbs[0].(map[string]interface{}); ok {
				if url, ok := thumb["url"].(string); ok {
					result.Image = url
				}
			}
		}
	}
	if lengthText, ok := video["lengthText"].(map[string]interface{}); ok {
		if simpleText, ok := lengthText["simpleText"].(string); ok {
			result.Duration = durationToSeconds(simpleText)
		}
	}
	if result.Duration == 0 {
		if lengthSeconds, ok := video["lengthSeconds"].(string); ok {
			sec, _ := strconv.Atoi(lengthSeconds)
			result.Duration = sec
		} else if lengthSecondsFloat, ok := video["lengthSeconds"].(float64); ok {
			result.Duration = int(lengthSecondsFloat)
		}
	}
	return result
}

func durationToSeconds(dur string) int {
	parts := strings.Split(dur, ":")
	if len(parts) == 2 {
		minutes, _ := strconv.Atoi(parts[0])
		secs, _ := strconv.Atoi(parts[1])
		return minutes*60 + secs
	} else if len(parts) == 3 {
		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		secs, _ := strconv.Atoi(parts[2])
		return hours*3600 + minutes*60 + secs
	}
	return 0
}

func parseChannelRenderer(channel map[string]interface{}) SearchResult {
	result := SearchResult{Type: "channel"}
	if channelId, ok := channel["channelId"].(string); ok {
		result.ID = channelId
		result.URL = "https://www.youtube.com/channel/" + channelId
	}
	if title, ok := channel["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Name = text
					result.Channel = text
				}
			}
		}
	}
	if thumbnails, ok := channel["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnails["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
			if thumb, ok := thumbs[0].(map[string]interface{}); ok {
				if url, ok := thumb["url"].(string); ok {
					result.Image = url
				}
			}
		}
	}
	return result
}

func parsePlaylistRenderer(playlist map[string]interface{}) SearchResult {
	result := SearchResult{Type: "playlist"}
	if playlistId, ok := playlist["playlistId"].(string); ok {
		result.ID = playlistId
		result.URL = "https://www.youtube.com/playlist?list=" + playlistId
	}
	if title, ok := playlist["title"].(map[string]interface{}); ok {
		if runs, ok := title["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Name = text
				}
			}
		}
	}
	if owner, ok := playlist["longBylineText"].(map[string]interface{}); ok {
		if runs, ok := owner["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Channel = text
				}
			}
		}
	}
	if thumbnails, ok := playlist["thumbnails"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnails["thumbnails"].([]interface{}); ok && len(thumbs) > 0 {
			if thumb, ok := thumbs[0].(map[string]interface{}); ok {
				if url, ok := thumb["url"].(string); ok {
					result.Image = url
				}
			}
		}
	}
	return result
}
