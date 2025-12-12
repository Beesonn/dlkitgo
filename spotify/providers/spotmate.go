package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

type SpotMate struct {
	Client *http.Client
}

func (p *SpotMate) Name() string {
	return "SpotMate"
}

func (p *SpotMate) BaseURL() string {
	return "https://spotmate.online"
}

func (p *SpotMate) Stream(spotifyURL string) (string, error) {
	if spotifyURL == "" {
		return "", errors.New("cannot be empty")
	}

	initialResp, err := p.DoInitialRequest()
	if err != nil {
		return "", err
	}

	csrfToken, sessionCookie, err := p.ExtractTokens(initialResp)
	if err != nil {
		return "", err
	}

	return p.DoConversionRequest(spotifyURL, csrfToken, sessionCookie)
}

func (p *SpotMate) DoInitialRequest() (*http.Response, error) {
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

func (p *SpotMate) ExtractTokens(resp *http.Response) (string, string, error) {
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	csrfToken, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists || csrfToken == "" {
		return "", "", errors.New("csrf token not found")
	}

	var sessionCookieValue string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "spotmateonline_session" {
			sessionCookieValue = cookie.Value
			break
		}
	}
	if sessionCookieValue == "" {
		return "", "", errors.New("session cookie 'spotmateonline_session' not found")
	}

	return csrfToken, sessionCookieValue, nil
}

func (p *SpotMate) DoConversionRequest(spotifyURL, csrfToken, sessionCookie string) (string, error) {
	jsonPayload, err := p.CreatePayload(spotifyURL)
	if err != nil {
		return "", err
	}

	apiURL := fmt.Sprintf("%s/convert", p.BaseURL())
	reqPost, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create POST request: %w", err)
	}

	p.SetRequestHeaders(reqPost, csrfToken, sessionCookie)

	respPost, err := p.Client.Do(reqPost)
	if err != nil {
		return "", fmt.Errorf("conversion request failed: %w", err)
	}
	defer respPost.Body.Close()

	return p.ParseResponse(respPost)
}

func (p *SpotMate) CreatePayload(spotifyURL string) ([]byte, error) {
	payload := map[string]string{
		"urls": spotifyURL,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON payload: %w", err)
	}
	return jsonPayload, nil
}

func (p *SpotMate) SetRequestHeaders(req *http.Request, csrfToken, sessionCookie string) {
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Referer", p.BaseURL()+"/en")
	req.Header.Set("Origin", p.BaseURL())
	req.Header.Set("Cookie", fmt.Sprintf("spotmateonline_session=%s", sessionCookie))
}

func (p *SpotMate) ParseResponse(resp *http.Response) (string, error) {
	if resp.StatusCode != 200 {
		return p.HandleErrorResponse(resp)
	}

	return p.ExtractDownloadURL(resp)
}

func (p *SpotMate) HandleErrorResponse(resp *http.Response) (string, error) {
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", fmt.Errorf("conversion returned status %d", resp.StatusCode)
	}
	return "", fmt.Errorf("conversion returned status %d. Body: %s", resp.StatusCode, string(bodyBytes))
}

func (p *SpotMate) ExtractDownloadURL(resp *http.Response) (string, error) {
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if errVal, exists := result["error"]; exists {
		if errValBool, ok := errVal.(bool); ok && errValBool {
			return "", errors.New("API error")
		}
	}

	downloadURL, ok := result["url"].(string)
	if !ok || downloadURL == "" {
		return "", errors.New("download url not found in response")
	}

	return downloadURL, nil
}
