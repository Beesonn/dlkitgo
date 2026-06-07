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
	ID       string `json:"id"`
	URL      string `json:"url"`
	Image    string `json:"image"`
	Duration int    `json:"duration"`
	Channel  string `json:"channel"`
	Name     string `json:"name"`
}

type SearchResponse struct {
	Query        string         `json:"query"`
	Limit        int            `json:"limit"`
	TotalResults int            `json:"total_results"`
	Results      []SearchResult `json:"results"`
}

func (t *TubeService) Search(query string, limit ...int) (*SearchResponse, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	maxResults := 20
	if len(limit) > 0 && limit[0] > 0 {
		maxResults = limit[0]
		if maxResults > 50 {
			maxResults = 50
		}
	}

	searchURL := "https://www.youtube.com/results?search_query=" + url.QueryEscape(query)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
   req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
   req.Header.Set("Cookie", "CONSENT=YES+cb.20210328-17-p0.en+FX+478")


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

	results := parseYouTubeResponse(string(body), query, maxResults)

	return results, nil
}

func parseYouTubeResponse(html, query string, limit int) *SearchResponse {
	response := &SearchResponse{
		Query:        query,
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
		}
	}
	response.TotalResults = len(response.Results)
	return response
}

func parseVideoRenderer(video map[string]interface{}) SearchResult {
	result := SearchResult{}
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