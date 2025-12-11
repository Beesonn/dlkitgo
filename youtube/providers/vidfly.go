package providers

import (
	"encoding/json"
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
		return YTResults{}, fmt.Errorf("url cannot be empty")
	}
	
	if !IsYouTubeURL(url) {
		return YTResults{}, fmt.Errorf("invalid YouTube URL")
	}
	
	if p.Client == nil {
		p.Client = &http.Client{}
	}
	
	apiURL := fmt.Sprintf("%s/api/media/youtube/download?url=%s", p.BaseURL(), url)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return YTResults{}, fmt.Errorf("request error: %v", err)
	}
	
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.Client.Do(req)
	if err != nil {
		return YTResults{}, fmt.Errorf("api error: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return YTResults{}, fmt.Errorf("HTTP %s", resp.Status)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return YTResults{}, fmt.Errorf("read error: %v", err)
	}
	
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return YTResults{}, fmt.Errorf("json error: %v", err)
	}
	
	codeVal, ok := data["code"]
	if !ok {
		return YTResults{}, fmt.Errorf("api response missing code field")
	}
	
	var code int

	switch v := codeVal.(type) {
	case float64:
		code = int(v)
	case int:
		code = v
	case int64:
		code = int(v)
	default:
		return YTResults{}, fmt.Errorf("invalid code type")
	}
	
	if code != 0 {
		return YTResults{}, fmt.Errorf("api code error: %d", code)
	}
	
	dataField, ok := data["data"].(map[string]interface{})
	if !ok {
		return YTResults{}, fmt.Errorf("api response missing data field")
	}
	
	title, _ := dataField["title"].(string)
	cover, _ := dataField["cover"].(string)
	
	var duration int
	if dur, ok := dataField["duration"].(float64); ok {
		duration = int(dur)
	}
	
	results := YTResults{
		Caption:   title,
		Thumbnail: cover,
		Duration:  duration,
	}
	
	items, ok := dataField["items"].([]interface{})
	if !ok {
		return results, nil 
	}
	
	results.Source = make([]YTSource, 0, len(items))
	
	for _, item := range items {
		itm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		
		streamURL, _ := itm["url"].(string)
		if streamURL == "" {
			continue
		}
		
		streamType, _ := itm["type"].(string)
		label, _ := itm["label"].(string)
		
		var quality string
		var height int
		
		if h, ok := itm["height"].(float64); ok {
			height = int(h)
		}
		
		if height > 0 {
			quality = strconv.Itoa(height) + "p"
		} else if streamType == "audio" {
			quality = p.ExtractAudioQuality(label)
		} else {
			quality = "N/A"
		}
		
		source := YTSource{
			URL:      streamURL,
			Duration: results.Duration,
			Type:     streamType,
			Quality:  quality,
		}
		results.Source = append(results.Source, source)
	}
	
	return results, nil
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