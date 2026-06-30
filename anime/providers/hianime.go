package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type HianimeProvider struct {
	Client *http.Client
	Base   string
	API    string
}

func NewHianimeProvider(client *http.Client) *HianimeProvider {
	return &HianimeProvider{
		Client: client,
		Base:   "https://hianimes.se",
		API:    "https://animedata.cfd",
	}
}

func (p *HianimeProvider) Name() string {
	return "hianime"
}

func (p *HianimeProvider) BaseURL() string {
	return p.Base
}

func (p *HianimeProvider) Search(query string) ([]AnimeSearchResult, error) {
	if query == "" {
		return nil, errors.New("search query cannot be empty")
	}

	payload := map[string]string{"title": query}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", p.API+"/api/search", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Referer", p.Base+"/")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResults []map[string]interface{}
	if err := json.Unmarshal(body, &apiResults); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return p.parseSearchResults(apiResults)
}

func (p *HianimeProvider) GetInfo(animeID string) (AnimeInfo, error) {
	if animeID == "" {
		return AnimeInfo{}, errors.New("anime ID cannot be empty")
	}

	encodedID := url.PathEscape(animeID)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/anime/%s", p.API, encodedID), nil)
	if err != nil {
		return AnimeInfo{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Referer", p.Base+"/details/"+animeID)

	resp, err := p.Client.Do(req)
	if err != nil {
		return AnimeInfo{}, fmt.Errorf("failed to fetch anime info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AnimeInfo{}, fmt.Errorf("failed to fetch info: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return AnimeInfo{}, fmt.Errorf("failed to read response: %w", err)
	}

	var apiResult map[string]interface{}
	if err := json.Unmarshal(body, &apiResult); err != nil {
		return AnimeInfo{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return p.parseAnimeInfo(apiResult)
}

func (p *HianimeProvider) Stream(animeID, episodeNum string) ([]VideoSource, error) {
	if animeID == "" || episodeNum == "" {
		return nil, errors.New("anime ID and episode number cannot be empty")
	}

	info, err := p.GetInfo(animeID)
	if err != nil {
		return nil, err
	}

	var targetEpisode *Episode
	for i := range info.Episodes {
		if info.Episodes[i].EpisodeNum == episodeNum {
			targetEpisode = &info.Episodes[i]
			break
		}
	}

	if targetEpisode == nil {
		return nil, fmt.Errorf("episode %s not found", episodeNum)
	}

	var allSources []VideoSource

	for _, link := range targetEpisode.SubLinks {
		sources, err := p.resolveMegaPlay(link, "sub")
		if err == nil {
			allSources = append(allSources, sources...)
		}
	}

	for _, link := range targetEpisode.DubLinks {
		sources, err := p.resolveMegaPlay(link, "dub")
		if err == nil {
			allSources = append(allSources, sources...)
		}
	}

	if len(allSources) == 0 {
		return nil, errors.New("no playable sources found")
	}

	return allSources, nil
}

func (p *HianimeProvider) Home() (HomeData, error) {
	var homeData HomeData

	req, err := http.NewRequest("GET", p.API+"/api/home", nil)
	if err != nil {
		return homeData, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Referer", p.Base+"/")

	resp, err := p.Client.Do(req)
	if err != nil {
		return homeData, fmt.Errorf("failed to fetch home data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return homeData, fmt.Errorf("home request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return homeData, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &homeData); err != nil {
		return homeData, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return homeData, nil
}

func (p *HianimeProvider) resolveMegaPlay(embedURL, qualityLabel string) ([]VideoSource, error) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0"

	req, err := http.NewRequest("GET", embedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create embed request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", p.Base+"/")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch embed page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed page returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embed page: %w", err)
	}
	html := string(body)

	re := regexp.MustCompile(`id="megaplay-player"[^>]*data-id="(\d+)"`)
	matches := re.FindStringSubmatch(html)
	if len(matches) < 2 {
		re2 := regexp.MustCompile(`data-id="(\d+)"[^>]*data-realid=`)
		matches = re2.FindStringSubmatch(html)
	}
	if len(matches) < 2 {
		return nil, errors.New("player ID not found in embed page")
	}
	playerID := matches[1]

	apiURL := "https://megaplay.buzz/stream/getSources?id=" + playerID
	apiReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
	}
	apiReq.Header.Set("User-Agent", userAgent)
	apiReq.Header.Set("Referer", embedURL)
	apiReq.Header.Set("X-Requested-With", "XMLHttpRequest")

	apiResp, err := p.Client.Do(apiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call megaplay API: %w", err)
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", apiResp.StatusCode)
	}

	apiBody, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read API response: %w", err)
	}

	var apiData map[string]interface{}
	if err := json.Unmarshal(apiBody, &apiData); err != nil {
		return nil, fmt.Errorf("failed to parse API JSON: %w", err)
	}

	var sources []VideoSource

	if sourcesField, ok := apiData["sources"]; ok {
		switch v := sourcesField.(type) {
		case []interface{}:
			for _, src := range v {
				if srcMap, ok := src.(map[string]interface{}); ok {
					file, _ := srcMap["file"].(string)
					label, _ := srcMap["label"].(string)
					if file != "" {
						sources = append(sources, VideoSource{
							URL:     file,
							Quality: label,
							IsM3U8:  true,
						})
					}
				}
			}
		case map[string]interface{}:
			if file, ok := v["file"].(string); ok && file != "" {
				sources = append(sources, VideoSource{
					URL:     file,
					Quality: qualityLabel,
					IsM3U8:  true,
				})
			}
		}
	}

	if len(sources) == 0 {
		return nil, errors.New("no video sources found in API response")
	}

	return sources, nil
}

func (p *HianimeProvider) parseSearchResults(apiResults []map[string]interface{}) ([]AnimeSearchResult, error) {
	var results []AnimeSearchResult

	for _, item := range apiResults {
		result := AnimeSearchResult{}

		if title, ok := item["title"].(string); ok {
			result.Title = title
		}
		if english, ok := item["English"].(string); ok && english != "" {
			result.Title = english
		}

		if slugs, ok := item["slugs"].([]interface{}); ok && len(slugs) > 0 {
			if slug, ok := slugs[0].(string); ok {
				result.ID = slug
			}
		}

		if image, ok := item["image"].(string); ok {
			result.Image = image
		}
		if typ, ok := item["Type"].(string); ok {
			result.Type = typ
		}
		if status, ok := item["Status"].(string); ok {
			result.Status = status
		}

		if episodes, ok := item["totalEpisodes"]; ok {
			switch v := episodes.(type) {
			case float64:
				result.Episodes = int(v)
			case string:
				if eps, err := strconv.Atoi(v); err == nil {
					result.Episodes = eps
				}
			}
		}

		if result.ID != "" {
			results = append(results, result)
		}
	}

	return results, nil
}

func (p *HianimeProvider) parseAnimeInfo(apiResult map[string]interface{}) (AnimeInfo, error) {
	info := AnimeInfo{}

	animeData, ok := apiResult["anime"].(map[string]interface{})
	if !ok {
		return info, errors.New("invalid anime data")
	}

	if slugs, ok := animeData["slugs"].([]interface{}); ok && len(slugs) > 0 {
		if slug, ok := slugs[0].(string); ok {
			info.ID = slug
		}
	}

	if title, ok := animeData["title"].(string); ok {
		info.Title = title
	}
	if english, ok := animeData["English"].(string); ok && english != "" {
		info.Title = english
	}
	if japanese, ok := animeData["Japanese"].(string); ok {
		info.JapaneseTitle = japanese
	}
	if image, ok := animeData["image"].(string); ok {
		info.Image = image
	}
	if synopsis, ok := animeData["synopsis"].(string); ok {
		info.Description = synopsis
	}
	if typ, ok := animeData["Type"].(string); ok {
		info.Type = typ
	}
	if status, ok := animeData["Status"].(string); ok {
		info.Status = status
	}
	if aired, ok := animeData["Aired"].(string); ok {
		info.Year = aired
	}

	if genres, ok := animeData["genres"].([]interface{}); ok {
		for _, g := range genres {
			if genre, ok := g.(string); ok {
				info.Genres = append(info.Genres, genre)
			}
		}
	}

	if episodesData, ok := animeData["episodes"].([]interface{}); ok {
		for _, ep := range episodesData {
			epMap, ok := ep.(map[string]interface{})
			if !ok {
				continue
			}

			episode := Episode{}

			if slugs, ok := epMap["slugs"].([]interface{}); ok && len(slugs) > 0 {
				if slug, ok := slugs[0].(string); ok {
					episode.ID = slug
				}
			}

			if num, ok := epMap["episodeNumber"]; ok {
				switch v := num.(type) {
				case float64:
					episode.EpisodeNum = strconv.Itoa(int(v))
				case string:
					episode.EpisodeNum = v
				}
			}

			if title, ok := epMap["title"].(string); ok {
				episode.Title = title
			}

			if link, ok := epMap["link"].(map[string]interface{}); ok {
				if sub, ok := link["sub"].([]interface{}); ok {
					for _, s := range sub {
						if str, ok := s.(string); ok {
							episode.SubLinks = append(episode.SubLinks, str)
						}
					}
				}
				if dub, ok := link["dub"].([]interface{}); ok {
					for _, d := range dub {
						if str, ok := d.(string); ok {
							episode.DubLinks = append(episode.DubLinks, str)
						}
					}
				}
			}

			if episode.ID != "" {
				info.Episodes = append(info.Episodes, episode)
			}
		}
	}

	return info, nil
}
