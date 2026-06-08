package youtube

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
	"regexp"
	"strconv"
	"strings"
)

type YouTubeVideoInfo struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Image    string `json:"image"`
}

type YouTubeData struct {
	Type        string             `json:"type"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	URL         string             `json:"url"`
	Image       string             `json:"image"`
	TotalVideos int                `json:"total_videos,omitempty"`
	Videos      []YouTubeVideoInfo `json:"videos,omitempty"`
}

type saveTubeResponse struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Thumbnail     string `json:"thumbnail"`
	Duration      int    `json:"duration"`
	DurationLabel string `json:"durationLabel"`
	FromCache     bool   `json:"fromCache"`
}

func (t *TubeService) GetInfo(url string) (YouTubeData, error) {
	if url == "" {
		return YouTubeData{}, errors.New("URL cannot be empty")
	}

	contentType, id := detectYouTubeType(url)
	if contentType == "" {
		return YouTubeData{}, errors.New("unsupported YouTube URL (only video or shorts)")
	}

	if strings.Contains(url, "&si=") {
		url = strings.Split(url, "&si=")[0]
	}

	result := YouTubeData{
		Type: contentType,
		ID:   id,
		URL:  url,
	}

	err := t.parseVideoPageDirect(url, &result)
	if err != nil {
		return t.getInfoFromAPI(url, contentType, id, result)
	}

	return result, nil
}

func (t *TubeService) parseVideoPageDirect(url string, result *YouTubeData) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	html := string(body)

	playerResponse := extractYtInitialPlayerResponse(html)
	if playerResponse == nil {
		return errors.New("could not extract video data")
	}

	videoDetails, ok := playerResponse["videoDetails"].(map[string]interface{})
	if !ok {
		return errors.New("invalid video page structure")
	}

	if title, ok := videoDetails["title"].(string); ok {
		result.Name = title
	}

	result.Image = "https://img.youtube.com/vi/" + result.ID + "/maxresdefault.jpg"

	var duration int
	if lengthSeconds, ok := videoDetails["lengthSeconds"].(string); ok {
		duration, _ = strconv.Atoi(lengthSeconds)
	}

	videoInfo := YouTubeVideoInfo{
		Name:     result.Name,
		URL:      result.URL,
		Duration: duration,
		Image:    result.Image,
	}

	result.Videos = []YouTubeVideoInfo{videoInfo}
	result.TotalVideos = 1

	return nil
}

func (t *TubeService) getInfoFromAPI(originalURL, contentType, id string, result YouTubeData) (YouTubeData, error) {
	apiData, err := fetchFromSaveTube(originalURL)
	if err != nil {
		return YouTubeData{}, fmt.Errorf("scraping failed and API error: %v", err)
	}

	result.Name = apiData.Title
	result.Image = apiData.Thumbnail

	videoInfo := YouTubeVideoInfo{
		Name:     apiData.Title,
		URL:      originalURL,
		Duration: apiData.Duration,
		Image:    apiData.Thumbnail,
	}

	result.Videos = []YouTubeVideoInfo{videoInfo}
	result.TotalVideos = 1

	return result, nil
}

func fetchFromSaveTube(url string) (*saveTubeResponse, error) {
	cdnResp, err := http.Get("https://media.savetube.vip/api/random-cdn")
	if err != nil {
		return nil, err
	}
	defer cdnResp.Body.Close()

	var cdnData struct {
		CDN string `json:"cdn"`
	}
	if err := json.NewDecoder(cdnResp.Body).Decode(&cdnData); err != nil {
		return nil, err
	}

	requestBody, err := json.Marshal(map[string]string{"url": url})
	if err != nil {
		return nil, err
	}

	apiResp, err := http.Post(fmt.Sprintf("https://%s/v2/info", cdnData.CDN), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer apiResp.Body.Close()

	var encResponse struct {
		Data string `json:"data"`
	}
	if err := json.NewDecoder(apiResp.Body).Decode(&encResponse); err != nil {
		return nil, err
	}

	decrypted, err := decryptAESCBC(encResponse.Data)
	if err != nil {
		return nil, err
	}

	var result saveTubeResponse
	if err := json.Unmarshal(decrypted, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func decryptAESCBC(encryptedData string) ([]byte, error) {
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

func detectYouTubeType(url string) (typ string, id string) {
	videoPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`youtube\.com/shorts/([a-zA-Z0-9_-]{11})`),
	}
	for _, re := range videoPatterns {
		if matches := re.FindStringSubmatch(url); len(matches) > 1 {
			if strings.Contains(url, "/shorts/") {
				return "shorts", matches[1]
			}
			return "video", matches[1]
		}
	}
	return "", ""
}

func extractYtInitialPlayerResponse(html string) map[string]interface{} {
	patterns := []string{
		`var ytInitialPlayerResponse\s*=\s*({.*?});</script>`,
		`var ytInitialPlayerResponse\s*=\s*({.*?});\s*var`,
		`ytInitialPlayerResponse\s*=\s*({.*?});`,
		`"ytInitialPlayerResponse"\s*:\s*({.*?})\s*,\s*"`,
		`ytInitialPlayerResponse"\s*:\s*({.*?})\s*}`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(html)
		if len(matches) >= 2 {
			var ytData map[string]interface{}
			if err := json.Unmarshal([]byte(matches[1]), &ytData); err == nil {
				return ytData
			}
		}
	}
	return nil
}
