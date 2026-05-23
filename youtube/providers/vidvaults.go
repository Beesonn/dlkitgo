package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	results := p.ExtractBasicInfo(data)
	p.ExtractDownloadOptions(data, &results, originalURL)

	return results, nil
}

func (p *VidVaults) ExtractBasicInfo(data map[string]interface{}) YTResults {
	title, _ := data["title"].(string)
	thumbnail, _ := data["thumbnail"].(string)

	var duration int
	if dur, ok := data["duration"].(float64); ok {
		duration = int(dur)
	}

	return YTResults{
		Caption:   title,
		Thumbnail: thumbnail,
		Duration:  duration,
		Source:    []YTSource{},
	}
}

func (p *VidVaults) ExtractDownloadOptions(data map[string]interface{}, results *YTResults, originalURL string) {
	downloadOptions, ok := data["downloadOptions"].(map[string]interface{})
	if !ok {
		return
	}

	if videoOpts, ok := downloadOptions["video"].([]interface{}); ok {
		for _, opt := range videoOpts {
			optMap, ok := opt.(map[string]interface{})
			if !ok {
				continue
			}

			source := p.CreateVideoSource(optMap, originalURL, results.Duration)
			if source.URL != "" {
				results.Source = append(results.Source, source)
			}
		}
	}

	if audioOpts, ok := downloadOptions["audio"].([]interface{}); ok {
		for _, opt := range audioOpts {
			optMap, ok := opt.(map[string]interface{})
			if !ok {
				continue
			}

			source := p.CreateAudioSource(optMap, originalURL, results.Duration)
			if source.URL != "" {
				results.Source = append(results.Source, source)
			}
		}
	}
}

func (p *VidVaults) CreateVideoSource(opt map[string]interface{}, originalURL string, duration int) YTSource {
	quality, _ := opt["quality"].(string)
	format, _ := opt["format"].(string)
	label, _ := opt["label"].(string)

	if quality == "" || format == "" {
		return YTSource{}
	}

	streamURL := fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=%s&format=%s&audioOnly=false",
		p.BaseURL(), originalURL, quality, format)

	return YTSource{
		URL:      streamURL,
		Duration: duration,
		Type:     "video",
		Quality:  p.FormatVideoQuality(quality, label),
	}
}

func (p *VidVaults) CreateAudioSource(opt map[string]interface{}, originalURL string, duration int) YTSource {
	quality, _ := opt["quality"].(string)
	format, _ := opt["format"].(string)
	label, _ := opt["label"].(string)

	if quality == "" || format == "" {
		return YTSource{}
	}

	streamURL := fmt.Sprintf("%s/api/v1/download/stream?url=%s&quality=%s&format=%s&audioOnly=true",
		p.BaseURL(), originalURL, quality, format)

	return YTSource{
		URL:      streamURL,
		Duration: duration,
		Type:     "audio",
		Quality:  p.FormatAudioQuality(quality, label),
	}
}

func (p *VidVaults) FormatVideoQuality(quality, label string) string {
	if quality != "" {
		return quality
	}
	if label != "" {
		return label
	}
	return "N/A"
}

func (p *VidVaults) FormatAudioQuality(quality, label string) string {
	if label != "" {
		parts := strings.Split(label, "(")
		if len(parts) > 1 {
			bitrate := strings.TrimSuffix(parts[1], ")")
			return bitrate
		}
		if strings.Contains(label, "kbps") {
			words := strings.Split(label, " ")
			for _, word := range words {
				if strings.Contains(word, "kbps") {
					return word
				}
			}
		}
	}
	if quality == "high" {
		return "320kbps"
	}
	if quality == "medium" {
		return "192kbps"
	}
	if quality != "" {
		return quality
	}
	return "audio"
}
