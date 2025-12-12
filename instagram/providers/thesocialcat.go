package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"errors"
	"io"
	"net/http"
)

type TheSocialCat struct {
	Client *http.Client
}

func (p *TheSocialCat) Name() string {
	return "thesocialcat"
}

func (p *TheSocialCat) BaseURL() string {
	return "https://thesocialcat.com"
}

func (p *TheSocialCat) Reel() bool {
	return true
}

func (p *TheSocialCat) Story() bool {
	return false
}

func (p *TheSocialCat) Post() bool {
	return true
}

func (p *TheSocialCat) Stream(url string) (InstaStreamResult, error) {
	result := InstaStreamResult{
		Caption:  "",
		Username: "",
		Video:    0,
		Photo:    0,
		Source:   []MediaSource{},
	}

	if url == "" {
		return result, errors.New("url cannot be empty")
	}

	apiResult, err := p.DoRequest(url)
	if err != nil {
		return result, err
	}

	p.ExtractData(apiResult, &result)
	result.Total = result.Video + result.Photo

	return result, nil
}

func (p *TheSocialCat) DoRequest(url string) (map[string]interface{}, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	payload := map[string]string{"url": url}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/instagram-download", p.BaseURL())
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var apiResult map[string]interface{}
	if err := json.Unmarshal(body, &apiResult); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return apiResult, nil
}

func (p *TheSocialCat) ExtractData(apiResult map[string]interface{}, result *InstaStreamResult) {
	if capText, ok := apiResult["caption"].(string); ok && capText != "" {
		result.Caption = capText
	}

	if userName, ok := apiResult["username"].(string); ok && userName != "" {
		result.Username = userName
	}

	mediaData, mediaOk := apiResult["mediaUrls"].([]interface{})
	if !mediaOk || mediaData == nil {
		return
	}

	mediaType, typeOk := apiResult["type"].(string)
	if !typeOk {
		mediaType = ""
	}

	thumb, thumbOk := apiResult["thumbnail"].(string)
	if !thumbOk {
		thumb = ""
	}

	p.ExtractMedia(mediaData, mediaType, thumb, result)
}

func (p *TheSocialCat) ExtractMedia(mediaData []interface{}, mediaType string, thumbnail string, result *InstaStreamResult) {
	if mediaType == "video" {
		result.Video = len(mediaData)
	} else if mediaType == "image" {
		result.Photo = len(mediaData)
	}

	startIndex := 0

	for i, medias := range mediaData {
		if urlStr, ok := medias.(string); ok && urlStr != "" {
			result.Source = append(result.Source, MediaSource{
				URL:       urlStr,
				Type:      mediaType,
				Thumbnail: thumbnail,
				Index:     startIndex + i,
			})
		}
	}
}
