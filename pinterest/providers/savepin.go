package providers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type SavePin struct {
	Client *http.Client
}

func (p *SavePin) Name() string {
	return "savepin"
}

func (p *SavePin) BaseURL() string {
	return "https://www.savepin.app"
}

func (p *SavePin) Stream(pinterestURL string) (PinResults, error) {
	if pinterestURL == "" {
		return PinResults{}, errors.New("pinterest URL cannot be empty")
	}

	htmlContent, err := p.DoRequest(pinterestURL)
	if err != nil {
		return PinResults{}, err
	}

	return p.ParseHTML(htmlContent)
}

func (p *SavePin) DoRequest(pinterestURL string) (string, error) {
	if p.Client == nil {
		p.Client = &http.Client{}
	}

	encodedURL := url.QueryEscape(pinterestURL)
	fullURL := fmt.Sprintf("%s/download.php?url=%s&lang=en&type=redirect", p.BaseURL(), encodedURL)

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("request error: %v", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Ch-Ua", `"Not)A;Brand";v="8", "Chromium";v="138", "Brave";v="138"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Referer", "https://www.savepin.app/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("api error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read error: %v", err)
	}

	return string(body), nil
}

func (p *SavePin) ParseHTML(htmlContent string) (PinResults, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return PinResults{}, fmt.Errorf("failed to parse HTML: %v", err)
	}

	title := doc.Find("h1").First().Text()
	title = strings.TrimSpace(title)

	thumbnail := ""
	doc.Find(".image-container img").Each(func(i int, s *goquery.Selection) {
		if thumbnail == "" {
			if src, exists := s.Attr("src"); exists {
				thumbnail = src
			}
		}
	})

	var sources []PinSource
	doc.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
		quality := s.Find(".video-quality").Text()
		quality = strings.TrimSpace(quality)

		format := s.Find("td:nth-child(2)").Text()
		format = strings.TrimSpace(format)

		href := ""
		s.Find("a").Each(func(j int, a *goquery.Selection) {
			if h, exists := a.Attr("href"); exists {
				href = h
			}
		})

		directURL := ""
		if href != "" {
			directURL = p.ExtractDirectURL(href)
		}

		if quality != "" && format != "" && directURL != "" {
			sourceType := p.DetermineType(format)
			sourceQuality := p.ExtractPureQuality(quality)

			sources = append(sources, PinSource{
				URL:     directURL,
				Type:    sourceType,
				Quality: sourceQuality,
			})
		}
	})

	return PinResults{
		Title:     title,
		Thumbnail: thumbnail,
		Source:    sources,
	}, nil
}

func (p *SavePin) ExtractDirectURL(href string) string {
	if strings.Contains(href, "url=") {
		parts := strings.Split(href, "url=")
		if len(parts) > 1 {
			decodedURL, err := url.QueryUnescape(parts[1])
			if err == nil {
				return decodedURL
			}
			return parts[1]
		}
	}
	return href
}

func (p *SavePin) DetermineType(format string) string {
	format = strings.ToLower(format)

	switch {
	case strings.Contains(format, "mp4"), strings.Contains(format, "video"):
		return "video"
	case strings.Contains(format, "mp3"), strings.Contains(format, "audio"):
		return "audio"
	case strings.Contains(format, "jpg"), strings.Contains(format, "jpeg"),
		strings.Contains(format, "png"), strings.Contains(format, "webp"):
		return "image"
	default:
		return "unknown"
	}
}

func (p *SavePin) ExtractPureQuality(quality string) string {
	quality = strings.TrimSpace(quality)

	parts := strings.Split(quality, "(")
	if len(parts) > 0 {
		pureQuality := strings.TrimSpace(parts[0])
		return pureQuality
	}

	return quality
}
