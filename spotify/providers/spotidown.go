package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

type Spotidown struct {
	Client *http.Client
}

func (p *Spotidown) Name() string {
	return "Spotidown"
}

func (p *Spotidown) BaseURL() string {
	return "https://spotidown.app"
}

func (p *Spotidown) Stream(spotifyURL string) (string, error) {
	if spotifyURL == "" {
		return "", errors.New("url cannot be empty")
	}

	jar, _ := cookiejar.New(nil)
	if p.Client == nil {
		p.Client = &http.Client{Jar: jar}
	} else {
		p.Client.Jar = jar
	}

	tokenFieldName, tokenValue, err := p.GetInitialTokens()
	if err != nil {
		return "", err
	}

	formData, err := p.GetFormData(spotifyURL, tokenFieldName, tokenValue)
	if err != nil {
		return "", err
	}

	rawURL, err := p.GetRawDownloadLink(formData)
	if err != nil {
		return "", err
	}

	return p.ProxyURL(rawURL), nil
}

func (p *Spotidown) GetInitialTokens() (string, string, error) {
	userAgent := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Mobile Safari/537.36"

	reqGet, err := http.NewRequest("GET", p.BaseURL()+"/en3", nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create initial GET request: %w", err)
	}
	reqGet.Header.Set("User-Agent", userAgent)
	reqGet.Header.Set("Accept-Language", "en-US,en;q=0.9")

	respGet, err := p.Client.Do(reqGet)
	if err != nil {
		return "", "", fmt.Errorf("initial handshake failed: %w", err)
	}
	defer respGet.Body.Close()

	if respGet.StatusCode != 200 {
		return "", "", fmt.Errorf("handshake returned status %d", respGet.StatusCode)
	}

	bodyBytes, err := io.ReadAll(respGet.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read response body: %w", err)
	}
	html := string(bodyBytes)

	hiddenInputs := regexp.MustCompile(`<input[^>]+type=["']hidden["'][^>]*>`).FindAllString(html, -1)

	var tokenFieldName, tokenValue string
	for _, inputTag := range hiddenInputs {
		nameMatch := regexp.MustCompile(`name=["']([^"']+)["']`).FindStringSubmatch(inputTag)
		if len(nameMatch) > 1 && nameMatch[1] != "" && nameMatch[1] != "g-recaptcha-response" {
			tokenFieldName = nameMatch[1]
			valueMatch := regexp.MustCompile(`value=["']([^"']*?)["']`).FindStringSubmatch(inputTag)
			if len(valueMatch) > 1 {
				tokenValue = valueMatch[1]
			} else {
				tokenValue = ""
			}
			break
		}
	}

	if tokenFieldName == "" {
		return "", "", errors.New("token field name not found")
	}

	return tokenFieldName, tokenValue, nil
}

func (p *Spotidown) GetFormData(spotifyURL, tokenFieldName, tokenValue string) (map[string]string, error) {
	userAgent := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Mobile Safari/537.36"

	formData := url.Values{}
	formData.Set("url", spotifyURL)
	formData.Set("g-recaptcha-response", "")
	formData.Set(tokenFieldName, tokenValue)

	reqPost, err := http.NewRequest("POST", p.BaseURL()+"/action", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %w", err)
	}

	reqPost.Header.Set("User-Agent", userAgent)
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("X-Requested-With", "XMLHttpRequest")
	reqPost.Header.Set("Referer", p.BaseURL()+"/en3")

	respPost, err := p.Client.Do(reqPost)
	if err != nil {
		return nil, fmt.Errorf("action request failed: %w", err)
	}
	defer respPost.Body.Close()

	bodyBytes, err := io.ReadAll(respPost.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var actionResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &actionResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if actionResponse["error"] != nil {
		errorVal := actionResponse["error"]
		if errorVal == true || (errorVal != nil && errorVal != false) {
			return nil, errors.New("API returned error")
		}
	}

	dataField, ok := actionResponse["data"].(string)
	if !ok {
		return nil, errors.New("data field not found in response")
	}

	dataMatch := regexp.MustCompile(`name="data"\s+value='([^']+)'`).FindStringSubmatch(dataField)
	baseMatch := regexp.MustCompile(`name="base"\s+value="([^"]+)"`).FindStringSubmatch(dataField)
	tokenMatch := regexp.MustCompile(`name="token"\s+value="([^"]+)"`).FindStringSubmatch(dataField)

	if len(dataMatch) < 2 || len(baseMatch) < 2 || len(tokenMatch) < 2 {
		return nil, errors.New("failed to extract form fields")
	}

	return map[string]string{
		"data":  dataMatch[1],
		"base":  baseMatch[1],
		"token": tokenMatch[1],
	}, nil
}

func (p *Spotidown) GetRawDownloadLink(formData map[string]string) (string, error) {
	userAgent := "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Mobile Safari/537.36"

	body2 := url.Values{}
	body2.Set("data", formData["data"])
	body2.Set("base", formData["base"])
	body2.Set("token", formData["token"])

	reqPost, err := http.NewRequest("POST", p.BaseURL()+"/action/track", strings.NewReader(body2.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create POST request: %w", err)
	}

	reqPost.Header.Set("User-Agent", userAgent)
	reqPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqPost.Header.Set("X-Requested-With", "XMLHttpRequest")
	reqPost.Header.Set("Referer", p.BaseURL()+"/en3")

	respPost, err := p.Client.Do(reqPost)
	if err != nil {
		return "", fmt.Errorf("track request failed: %w", err)
	}
	defer respPost.Body.Close()

	bodyBytes, err := io.ReadAll(respPost.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var trackResponse map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &trackResponse); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	if trackResponse["error"] != nil {
		errorVal := trackResponse["error"]
		if errorVal == true || (errorVal != nil && errorVal != false) {
			return "", errors.New("API returned error")
		}
	}

	dataField, ok := trackResponse["data"].(string)
	if !ok {
		return "", errors.New("data field not found in response")
	}

	urlRegex := regexp.MustCompile(`href="(https://rapid\.spotidown\.app/v2\?token=[^"]+)"`)
	urls := urlRegex.FindStringSubmatch(dataField)

	if len(urls) < 2 {
		return "", errors.New("no download URL found")
	}

	return urls[1], nil
}

func (p *Spotidown) ProxyURL(rawURL string) string {
	return "https://proxy-xi-dun.vercel.app/proxy_audio?url=" + rawURL
}
