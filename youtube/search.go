package youtube

import (
	"errors"
	"fmt"
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
	Limit        int            `json:"limit"`
	TotalResults int            `json:"total_results"`
	Results      []SearchResult `json:"results"`
}

func (t *TubeService) Search(query string, limit ...int) (*SearchResponse, error) {
	if query == "" {
		return nil, errors.New("query cannot be empty")
	}

	maxResults := 10
	if len(limit) > 0 && limit[0] > 0 {
		maxResults = limit[0]
		if maxResults > 50 {
			maxResults = 50
		}
	}

	innerTube := NewInnerTubeClient(t.Client)

	response, err := innerTube.Search(query, "", "")
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := parseInnerTubeSearchResults(response, query, maxResults)

	for i := range results.Results {
		if results.Results[i].Type == "video" && results.Results[i].ID != "" {
			results.Results[i].ReleaseDate = t.fetchReleaseDateViaAPI(results.Results[i].ID)
		}
	}

	return results, nil
}

func (t *TubeService) fetchReleaseDateViaAPI(videoID string) string {
	innerTube := NewInnerTubeClient(t.Client)
	playerData, err := innerTube.GetPlayer(videoID)
	if err != nil {
		return ""
	}

	if videoDetails, ok := playerData["videoDetails"].(map[string]interface{}); ok {
		if publishDate, ok := videoDetails["publishDate"].(string); ok {
			return publishDate
		}
	}

	if microformat, ok := playerData["microformat"].(map[string]interface{}); ok {
		if playerMicroformat, ok := microformat["playerMicroformatRenderer"].(map[string]interface{}); ok {
			if publishDate, ok := playerMicroformat["publishDate"].(string); ok {
				return publishDate
			}
			if uploadDate, ok := playerMicroformat["uploadDate"].(string); ok {
				return uploadDate
			}
		}
	}

	return ""
}

func parseInnerTubeSearchResults(data map[string]interface{}, query string, limit int) *SearchResponse {
	response := &SearchResponse{
		Query:        query,
		Limit:        limit,
		TotalResults: 0,
		Results:      []SearchResult{},
	}

	contents, ok := data["contents"].(map[string]interface{})
	if !ok {
		return response
	}

	sectionList, ok := contents["twoColumnSearchResultsRenderer"].(map[string]interface{})
	if !ok {
		return response
	}

	primaryContents, ok := sectionList["primaryContents"].(map[string]interface{})
	if !ok {
		return response
	}

	sectionListRenderer, ok := primaryContents["sectionListRenderer"].(map[string]interface{})
	if !ok {
		return response
	}

	contentsList, ok := sectionListRenderer["contents"].([]interface{})
	if len(contentsList) == 0 {
		return response
	}

	itemSection, ok := contentsList[0].(map[string]interface{})
	if !ok {
		return response
	}

	itemSectionRenderer, ok := itemSection["itemSectionRenderer"].(map[string]interface{})
	if !ok {
		return response
	}

	items, ok := itemSectionRenderer["contents"].([]interface{})
	if !ok {
		return response
	}

	count := 0
	for _, item := range items {
		if count >= limit {
			break
		}

		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if videoRenderer, ok := itemMap["videoRenderer"].(map[string]interface{}); ok {
			result := parseInnerTubeVideo(videoRenderer)
			if result.ID != "" {
				response.Results = append(response.Results, result)
				count++
			}
		}
	}

	response.TotalResults = len(response.Results)
	return response
}

func parseInnerTubeVideo(video map[string]interface{}) SearchResult {
	result := SearchResult{Type: "video"}

	if videoID, ok := video["videoId"].(string); ok {
		result.ID = videoID
		result.URL = "https://www.youtube.com/watch?v=" + videoID
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

	if ownerText, ok := video["ownerText"].(map[string]interface{}); ok {
		if runs, ok := ownerText["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if text, ok := run["text"].(string); ok {
					result.Channel = text
				}
			}
		}
	}

	if thumbnail, ok := video["thumbnail"].(map[string]interface{}); ok {
		if thumbnails, ok := thumbnail["thumbnails"].([]interface{}); ok && len(thumbnails) > 0 {
			if thumb, ok := thumbnails[0].(map[string]interface{}); ok {
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
		}
	}

	return result
}

func durationToSeconds(dur string) int {
	if dur == "" {
		return 0
	}

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
