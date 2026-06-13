package providers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type SaveTube struct {
	Client *http.Client
}

type saveTubeCDNResponse struct {
	CDN string `json:"cdn"`
}

type saveTubeInfoResponse struct {
	Data string `json:"data"`
}

type saveTubeInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Thumbnail   string `json:"thumbnail"`
	Duration    int    `json:"duration"`
	Key         string `json:"key"`
	FromCache   bool   `json:"fromCache"`
	DurationRaw string `json:"durationLabel"`
}

type saveTubeDownloadResponse struct {
	Data struct {
		DownloadURL string `json:"downloadUrl"`
	} `json:"data"`
}

type qualityPair struct {
	quality string
	label   string
}

func (p *SaveTube) Name() string {
	return "savetube"
}

func (p *SaveTube) BaseURL() string {
	return "https://media.savetube.vip"
}

func (p *SaveTube) Stream(url string) (YTResults, error) {
	if url == "" {
		return YTResults{}, errors.New("url cannot be empty")
	}

	if !IsYouTubeURL(url) {
		return YTResults{}, errors.New("invalid YouTube URL")
	}

	info, err := p.getVideoInfo(url)
	if err != nil {
		return YTResults{}, err
	}

	return p.buildResultsFast(info, url), nil
}

func (p *SaveTube) getVideoInfo(url string) (*saveTubeInfo, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	cdnResp, err := p.Client.Get("https://media.savetube.vip/api/random-cdn")
	if err != nil {
		return nil, fmt.Errorf("failed to get CDN: %v", err)
	}
	defer cdnResp.Body.Close()

	var cdnData saveTubeCDNResponse
	if err := json.NewDecoder(cdnResp.Body).Decode(&cdnData); err != nil {
		return nil, fmt.Errorf("failed to decode CDN response: %v", err)
	}

	reqBody, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	apiURL := fmt.Sprintf("https://%s/v2/info", cdnData.CDN)
	resp, err := p.Client.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var encResponse saveTubeInfoResponse
	if err := json.Unmarshal(body, &encResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	decrypted, err := p.decryptAESCBC(encResponse.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	var info saveTubeInfo
	if err := json.Unmarshal(decrypted, &info); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %v", err)
	}

	return &info, nil
}

func (p *SaveTube) decryptAESCBC(encryptedData string) ([]byte, error) {
	buf, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return nil, err
	}

	if len(buf) < 16 {
		return nil, errors.New("encrypted data too short")
	}

	iv := buf[:16]
	ciphertext := buf[16:]

	key := []byte{0xC5, 0xD5, 0x8E, 0xF6, 0x7A, 0x75, 0x84, 0xE4, 0xA2, 0x9F, 0x6C, 0x35, 0xBB, 0xC4, 0xEB, 0x12}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	paddingLen := int(ciphertext[len(ciphertext)-1])
	if paddingLen > len(ciphertext) || paddingLen > aes.BlockSize {
		return nil, errors.New("invalid padding")
	}

	return ciphertext[:len(ciphertext)-paddingLen], nil
}

func (p *SaveTube) buildResultsFast(info *saveTubeInfo, originalURL string) YTResults {
	results := YTResults{
		Caption:   info.Title,
		Thumbnail: info.Thumbnail,
		Duration:  info.Duration,
		Source:    []YTSource{},
	}

	videoQualities := []qualityPair{
		{"144", "144p"},
		{"360", "360p"},
		{"480", "480p"},
		{"720", "720p"},
		{"1080", "1080p"},
	}

	audioQualities := []qualityPair{
		{"144", "144kbps"},
		{"360", "360kbps"},
		{"480", "480kbps"},
		{"720", "720kbps"},
		{"1080", "1080kbps"},
	}

	cdn, err := p.getFastestCDN()
	if err != nil {
		cdn = "media.savetube.vip"
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	sources := make([]YTSource, 0, 10)

	for _, vq := range videoQualities {
		wg.Add(1)
		go func(qual qualityPair) {
			defer wg.Done()
			downloadURL := p.getDownloadURLFast(cdn, info.Key, "video", qual.quality)
			if downloadURL != "" {
				mu.Lock()
				sources = append(sources, YTSource{
					URL:      downloadURL,
					Duration: info.Duration,
					Type:     "video",
					Quality:  qual.label,
				})
				mu.Unlock()
			}
		}(vq)
	}

	for _, aq := range audioQualities {
		wg.Add(1)
		go func(qual qualityPair) {
			defer wg.Done()
			downloadURL := p.getDownloadURLFast(cdn, info.Key, "audio", qual.quality)
			if downloadURL != "" {
				mu.Lock()
				sources = append(sources, YTSource{
					URL:      downloadURL,
					Duration: info.Duration,
					Type:     "audio",
					Quality:  qual.label,
				})
				mu.Unlock()
			}
		}(aq)
	}

	wg.Wait()
	results.Source = sources
	return results
}

func (p *SaveTube) getFastestCDN() (string, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	resp, err := p.Client.Get("https://media.savetube.vip/api/random-cdn")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var cdnData saveTubeCDNResponse
	if err := json.NewDecoder(resp.Body).Decode(&cdnData); err != nil {
		return "", err
	}

	return cdnData.CDN, nil
}

func (p *SaveTube) getDownloadURLFast(cdn, key, downloadType, quality string) string {
	reqBody, err := json.Marshal(map[string]interface{}{
		"downloadType": downloadType,
		"quality":      quality,
		"key":          key,
	})
	if err != nil {
		return ""
	}

	apiURL := fmt.Sprintf("https://%s/download", cdn)
	resp, err := p.Client.Post(apiURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var downloadResp saveTubeDownloadResponse
	if err := json.Unmarshal(body, &downloadResp); err != nil {
		return ""
	}

	return downloadResp.Data.DownloadURL
}
