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
	"strings"
)

type Downloaderize struct {
	Client *http.Client
}

func (p *Downloaderize) Name() string {
	return "Downloaderize"
}

func (p *Downloaderize) BaseURL() string {
	return "https://spotify.downloaderize.com"
}

func (p *Downloaderize) Stream(spotifyURL string) (string, error) {
	if spotifyURL == "" {
		return "", errors.New("url cannot be empty")
	}

	initialResp, err := p.DoInitialRequest()
	if err != nil {
		return "", err
	}

	nonce, err := p.ExtractNonce(initialResp)
	if err != nil {
		return "", err
	}

	return p.DoConversionRequest(spotifyURL, nonce)
}

func (p *Downloaderize) DoInitialRequest() (*http.Response, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	reqGet, err := http.NewRequest("GET", p.BaseURL(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial GET request: %w", err)
	}
	reqGet.Header.Set("User-Agent", userAgent)
	reqGet.Header.Set("Accept", "text/html")

	respGet, err := p.Client.Do(reqGet)
	if err != nil {
		return nil, fmt.Errorf("initial handshake failed: %w", err)
	}

	if respGet.StatusCode != 200 {
		respGet.Body.Close()
		return nil, fmt.Errorf("handshake returned status %d", respGet.StatusCode)
	}

	return respGet, nil
}

func (p *Downloaderize) ExtractNonce(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	html := string(bodyBytes)
	regex := regexp.MustCompile(`"nonce":"([a-f0-9]+)"`)
	matches := regex.FindStringSubmatch(html)

	if len(matches) < 2 || matches[1] == "" {
		return "", errors.New("nonce not found in response")
	}

	return matches[1], nil
}

func (p *Downloaderize) DoConversionRequest(spotifyURL, nonce string) (string, error) {
	formData := url.Values{}
	formData.Set("action", "spotify_downloader_get_info")
	formData.Set("url", spotifyURL)
	formData.Set("nonce", nonce)

	apiURL := fmt.Sprintf("%s/wp-admin/admin-ajax.php", p.BaseURL())
	reqPost, err := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create POST request: %w", err)
	}

	p.SetRequestHeaders(reqPost)

	respPost, err := p.Client.Do(reqPost)
	if err != nil {
		return "", fmt.Errorf("conversion request failed: %w", err)
	}
	defer respPost.Body.Close()

	return p.ParseResponse(respPost)
}

func (p *Downloaderize) SetRequestHeaders(req *http.Request) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Referer", p.BaseURL()+"/")
	req.Header.Set("Origin", p.BaseURL())
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func (p *Downloaderize) ParseResponse(resp *http.Response) (string, error) {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("conversion returned status %d. Body: %s", resp.StatusCode, string(bodyBytes))
	}

	result, err := p.ParseJSONResponse(bodyBytes)
	if err != nil {
		return "", err
	}

	if err := p.CheckSuccess(result); err != nil {
		return "", err
	}

	return p.ExtractDownloadURL(result)
}

func (p *Downloaderize) ParseJSONResponse(bodyBytes []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	return result, nil
}

func (p *Downloaderize) CheckSuccess(result map[string]interface{}) error {
	successVal, exists := result["success"]
	if !exists {
		return nil
	}

	if successBool, ok := successVal.(bool); ok && !successBool {
		if dataVal, dataExists := result["data"]; dataExists {
			if errorMsg, ok := dataVal.(string); ok {
				return fmt.Errorf("API error: %s", errorMsg)
			}
		}
		return errors.New("API returned unsuccessful response")
	}
	return nil
}

func (p *Downloaderize) ExtractDownloadURL(result map[string]interface{}) (string, error) {
	dataVal, dataExists := result["data"]
	if !dataExists {
		return "", errors.New("data field not found in response")
	}

	data, ok := dataVal.(map[string]interface{})
	if !ok {
		return "", errors.New("data field is not an object")
	}

	mediaVal, mediaExists := data["medias"]
	if !mediaExists {
		return "", errors.New("media field not found in response")
	}

	media, ok := mediaVal.([]interface{})
	if !ok || len(media) == 0 {
		return "", errors.New("no download links found in response")
	}

	firstMedia, ok := media[0].(map[string]interface{})
	if !ok {
		return "", errors.New("first media item is not valid")
	}

	downloadURL, urlExists := firstMedia["url"].(string)
	if !urlExists || downloadURL == "" {
		return "", errors.New("download URL not found in media item")
	}

	return downloadURL, nil
}
