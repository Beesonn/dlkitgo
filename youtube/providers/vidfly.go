package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type VidFly struct {
	Client *http.Client
}

func (p *VidFly) Name() string {
	return "vidfly"
}

func (p *VidFly) BaseURL() string {
	return "https://api.vidfly.ai"
}

func (p *VidFly) Stream(url string) (YTResults, error) {
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

	return p.ParseResponse(apiResponse)
}

func (p *VidFly) DoRequest(url string) (map[string]interface{}, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	apiURL := fmt.Sprintf("%s/api/media/youtube/download?url=%s", p.BaseURL(), url)
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

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("json error: %v", err)
	}

	return data, nil
}

func (p *VidFly) ParseResponse(data map[string]interface{}) (YTResults, error) {
	dataField, ok := data["data"].(map[string]interface{})
	if !ok {
		return YTResults{}, fmt.Errorf("api response missing data field")
	}

	results := p.ExtractBasicInfo(dataField)
	p.ExtractItems(dataField, &results)

	return results, nil
}

func (p *VidFly) ExtractBasicInfo(dataField map[string]interface{}) YTResults {
	title, _ := dataField["title"].(string)
	cover, _ := dataField["cover"].(string)

	var duration int
	if dur, ok := dataField["duration"].(float64); ok {
		duration = int(dur)
	}

	return YTResults{
		Caption:   title,
		Thumbnail: cover,
		Duration:  duration,
		Source:    []YTSource{},
	}
}

func (p *VidFly) ExtractItems(dataField map[string]interface{}, results *YTResults) {
	items, ok := dataField["items"].([]interface{})
	if !ok {
		return
	}

	for _, item := range items {
		itm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		source := p.CreateSource(itm, results.Duration)
		if source.URL != "" {
			results.Source = append(results.Source, source)
		}
	}
}

func (p *VidFly) CreateSource(item map[string]interface{}, duration int) YTSource {
	streamURL, _ := item["url"].(string)
	if streamURL == "" {
		return YTSource{}
	}

	streamType, _ := item["type"].(string)
	label, _ := item["label"].(string)
	quality := p.DetermineQuality(item, streamType, label)

	return YTSource{
		URL:      streamURL,
		Duration: duration,
		Type:     streamType,
		Quality:  quality,
	}
}

func (p *VidFly) DetermineQuality(item map[string]interface{}, streamType, label string) string {
	if h, ok := item["height"].(float64); ok {
		height := int(h)
		if height > 0 {
			return strconv.Itoa(height) + "p"
		}
	}

	if streamType == "audio" {
		return p.ExtractAudioQuality(label)
	}

	return "N/A"
}

func (p *VidFly) ExtractAudioQuality(label string) string {
	if label == "" {
		return "audio"
	}

	parts := strings.Split(label, "(")
	if len(parts) > 1 {
		quality := strings.TrimSpace(parts[1])
		quality = strings.TrimSuffix(quality, ")")
		return quality
	}
	return label
}
