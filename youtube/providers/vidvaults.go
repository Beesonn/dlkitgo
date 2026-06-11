package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type VidVaults struct {
	Client *http.Client
}

func (p *VidVaults) Name() string {
	return "vidvaults"
}

func (p *VidVaults) BaseURL() string {
	return "https://api.vidvaults.com"
}

func (p *VidVaults) Stream(url string) (YTResults, error) {
	if url == "" {
		return YTResults{}, errors.New("url cannot be empty")
	}

	if !IsYouTubeURL(url) {
		return YTResults{}, errors.New("invalid YouTube URL")
	}

	apiResponse, err := p.DoRequest(url)
	if err != nil {
		return YTResults{}, err
	}

	return p.ParseResponse(apiResponse, url)
}

func (p *VidVaults) DoRequest(url string) (map[string]interface{}, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	apiURL := fmt.Sprintf("%s/api/v1/instant/metadata?url=%s", p.BaseURL(), url)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %v", err)
	}

	var apiResponse map[string]interface{}
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("json error: %v", err)
	}

	if status, ok := apiResponse["status"].(string); ok && status != "success" {
		return nil, fmt.Errorf("api error: %v", apiResponse["message"])
	}

	data, ok := apiResponse["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("api response missing data field")
	}

	return data, nil
}

func (p *VidVaults) ParseResponse(data map[string]interface{}, originalURL string) (YTResults, error) {
	results := YTResults{
		Source: []YTSource{},
	}

	if title, ok := data["title"].(string); ok {
		results.Caption = title
	}

	if thumbnail, ok := data["thumbnail"].(string); ok {
		results.Thumbnail = thumbnail
	}

	if duration, ok := data["duration"].(float64); ok {
		results.Duration = int(duration)
	}

	encodedURL := url.QueryEscape(originalURL)

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=low&format=mp4&audioOnly=false", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "video",
		Quality:  "480p",
	})

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=medium&format=mp4&audioOnly=false", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "video",
		Quality:  "720p",
	})

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=high&format=mp4&audioOnly=false", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "video",
		Quality:  "1080p",
	})

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=highest&format=mp4&audioOnly=false", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "video",
		Quality:  "4K",
	})

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=high&format=mp3&audioOnly=true", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "audio",
		Quality:  "320kbps",
	})

	results.Source = append(results.Source, YTSource{
		URL:      fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=medium&format=mp3&audioOnly=true", p.BaseURL(), encodedURL),
		Duration: results.Duration,
		Type:     "audio",
		Quality:  "192kbps",
	})

	return results, nil
}