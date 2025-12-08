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
		return "", errors.New("spotmate: empty url")
	}

	if p.Client == nil {
		p.Client = &http.Client{}
	}

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"

	reqGet, err := http.NewRequest("GET", p.BaseURL(), nil)
	if err != nil {
		return "", fmt.Errorf("spotmate: failed to create initial GET request: %w", err)
	}
	reqGet.Header.Set("User-Agent", userAgent)
	reqGet.Header.Set("Accept", "text/html")

	respGet, err := p.Client.Do(reqGet)
	if err != nil {
		return "", fmt.Errorf("spotmate: initial handshake failed: %w", err)
	}
	defer respGet.Body.Close()

	if respGet.StatusCode != 200 {
		return "", fmt.Errorf("spotmate: handshake returned status %d", respGet.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(respGet.Body)
	if err != nil {
		return "", fmt.Errorf("spotmate: failed to parse HTML: %w", err)
	}

	csrfToken, exists := doc.Find("meta[name='csrf-token']").Attr("content")
	if !exists || csrfToken == "" {
		return "", errors.New("spotmate: csrf token not found")
	}

	var sessionCookieValue string
	for _, cookie := range respGet.Cookies() {
		if cookie.Name == "spotmateonline_session" {
			sessionCookieValue = cookie.Value
			break
		}
	}
	if sessionCookieValue == "" {
		return "", errors.New("spotmate: session cookie 'spotmateonline_session' not found")
	}

	payload := map[string]string{
		"urls": spotifyURL,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("spotmate: failed to marshal JSON payload: %w", err)
	}

	apiURL := fmt.Sprintf("%s/convert", p.BaseURL())
	reqPost, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("spotmate: failed to create POST request: %w", err)
	}

	reqPost.Header.Set("User-Agent", userAgent)
	reqPost.Header.Set("Content-Type", "application/json")
	reqPost.Header.Set("X-CSRF-Token", csrfToken)
	reqPost.Header.Set("Referer", p.BaseURL()+"/en")
	reqPost.Header.Set("Origin", p.BaseURL())

	reqPost.Header.Set("Cookie", fmt.Sprintf("spotmateonline_session=%s", sessionCookieValue))

	respPost, err := p.Client.Do(reqPost)
	if err != nil {
		return "", fmt.Errorf("spotmate: conversion request failed: %w", err)
	}
	defer respPost.Body.Close()

	if respPost.StatusCode != 200 {
		bodyBytes, readErr := io.ReadAll(respPost.Body)
		if readErr != nil {
			return "", fmt.Errorf("spotmate: conversion returned status %d", respPost.StatusCode)
		}
		return "", fmt.Errorf("spotmate: conversion returned status %d. Body: %s", respPost.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(respPost.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("spotmate: failed to decode JSON response: %w", err)
	}

	if errVal, exists := result["error"]; exists {
		if errValBool, ok := errVal.(bool); ok && errValBool {
			return "", errors.New("spotmate: API error")
		}
	}

	downloadURL, ok := result["url"].(string)
	if !ok || downloadURL == "" {
		return "", errors.New("spotmate: download url not found in response")
	}

	return downloadURL, nil
}