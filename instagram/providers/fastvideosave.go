package providers

import (
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type FastVideoSave struct {
	Client *http.Client
}

func (p *FastVideoSave) Name() string {
	return "FastVideoSave"
}

func (p *FastVideoSave) BaseURL() string {
	return "https://api.videodropper.app"
}

func (p *FastVideoSave) Reel() bool {
	return true
}

func (p *FastVideoSave) Story() bool {
	return true
}

func (p *FastVideoSave) Post() bool {
	return true
}

func (p *FastVideoSave) EncodeURL(text string) (string, error) {
	key := []byte("qwertyuioplkjhgf")

	data := []byte(text)
	blockSize := 16
	padding := blockSize - len(data)%blockSize
	padText := make([]byte, padding)
	for i := range padText {
		padText[i] = byte(padding)
	}
	paddedData := append(data, padText...)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	encrypted := make([]byte, len(paddedData))
	for bs, be := 0, block.BlockSize(); bs < len(paddedData); bs, be = bs+block.BlockSize(), be+block.BlockSize() {
		block.Encrypt(encrypted[bs:be], paddedData[bs:be])
	}

	return hex.EncodeToString(encrypted), nil
}

func (p *FastVideoSave) Stream(url string) (InstaStreamResult, error) {
	result := InstaStreamResult{
		Caption:  "",
		Username: "",
		Video:    0,
		Photo:    0,
		Source:   []MediaSource{},
	}

	if url == "" {
		return result, fmt.Errorf("url cannot be empty")
	}

	encryptedURL, err := p.EncodeURL(url)
	if err != nil {
		return result, fmt.Errorf("failed to encrypt URL: %v", err)
	}

	if p.Client == nil {
		p.Client = &http.Client{}
	}

	apiURL := fmt.Sprintf("%s/allinone", p.BaseURL())

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return result, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://fastvideo.net")
	req.Header.Set("Referer", "https://fastvideo.net/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Url", encryptedURL)

	resp, err := p.Client.Do(req)
	if err != nil {
		return result, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, fmt.Errorf("failed to read response: %v", err)
	}

	var apiResult map[string]interface{}
	if err := json.Unmarshal(body, &apiResult); err != nil {
		return result, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	if videoData, ok := apiResult["video"].([]interface{}); ok && videoData != nil {
		result.Video = len(videoData)

		for i, v := range videoData {
			if videoObj, ok := v.(map[string]interface{}); ok {
				videoURL := ""
				thumbnail := ""

				if vURL, ok := videoObj["video"].(string); ok && vURL != "" {
					videoURL = vURL
				}
				if thumb, ok := videoObj["thumbnail"].(string); ok && thumb != "" {
					thumbnail = thumb
				}

				if videoURL != "" {
					result.Source = append(result.Source, MediaSource{
						URL:       videoURL,
						Type:      "video",
						Thumbnail: thumbnail,
						Index:     i,
					})
				}
			} else if videoStr, ok := v.(string); ok && videoStr != "" {
				result.Source = append(result.Source, MediaSource{
					URL:       videoStr,
					Type:      "video",
					Thumbnail: "",
					Index:     i,
				})
			}
		}
	}

	if images, ok := apiResult["image"].([]interface{}); ok && images != nil {
		result.Photo = len(images)
		startIndex := result.Video

		for i, img := range images {
			if urlStr, ok := img.(string); ok && urlStr != "" {
				result.Source = append(result.Source, MediaSource{
					URL:       urlStr,
					Type:      "photo",
					Thumbnail: "",
					Index:     startIndex + i,
				})
			}
		}
	}

	result.Total = result.Video + result.Photo

	return result, nil
}
