package instagram

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type InstagramData struct {
	Username  string `json:"username"`
	Likes     string `json:"likes"`
	Comments  string `json:"comments"`
	Caption   string `json:"caption"`
	Date      string `json:"date"`
	Thumbnail string `json:"thumbnail"`
}

var (
	InstagramURLPattern = regexp.MustCompile(`^https?://(?:www\.)?instagram\.com/(?:p|reel|tv)/[a-zA-Z0-9_-]+`)
	LikesRegex         = regexp.MustCompile(`^([0-9Kk\.,]+) likes`)
	CommentsRegex      = regexp.MustCompile(`,\s*([0-9Kk\.,]+)\s*comments`)
	UserRegex          = regexp.MustCompile(`-\s*(.*?)\s*on`)
	DateRegex          = regexp.MustCompile(`on\s(.*?):`)
	CaptionRegex       = regexp.MustCompile(`:\s*"(.*)"`)
)

func (insta *InstaService) GetInfo(url string) (InstagramData, error) {
	var data InstagramData
	
	if url == "" {
		return data, errors.New("url cannot be empty")
	}
	
	if !InstagramURLPattern.MatchString(url) {
		return data, errors.New("Invalid Instagram URL")
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	resp, err := insta.Client.Do(req)
	if err != nil {
		return data, errors.New("Invalid Instagram URL")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return data, errors.New("Invalid Instagram URL")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return data, fmt.Errorf("failed to parse HTML: %w", err)
	}

	data = insta.ExtractInstagramData(doc)
	return data, nil
}

func (insta *InstaService) ExtractInstagramData(doc *goquery.Document) InstagramData {
	data := InstagramData{
		Username:  "",
		Likes:     "",
		Comments:  "",
		Caption:   "",
		Date:      "",
		Thumbnail: "",
	}

	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		if property, exists := s.Attr("property"); exists {
			if property == "og:image" {
				if content, exists := s.Attr("content"); exists {
					data.Thumbnail = content
				}
			}
			if property == "og:description" {
				if content, exists := s.Attr("content"); exists {
					insta.ParseMetaDescription(content, &data)
				}
			}
		}
	})

	return data
}

func (insta *InstaService) ParseMetaDescription(descText string, data *InstagramData) {
	if matches := LikesRegex.FindStringSubmatch(descText); len(matches) > 1 {
		data.Likes = matches[1]
	}

	if matches := CommentsRegex.FindStringSubmatch(descText); len(matches) > 1 {
		data.Comments = matches[1]
	}

	if matches := UserRegex.FindStringSubmatch(descText); len(matches) > 1 {
		data.Username = matches[1]
	}

	if matches := DateRegex.FindStringSubmatch(descText); len(matches) > 1 {
		data.Date = matches[1]
	}

	if matches := CaptionRegex.FindStringSubmatch(descText); len(matches) > 1 {
		data.Caption = matches[1]
	} else if strings.Contains(descText, ":") {
		parts := strings.SplitN(descText, ":", 2)
		if len(parts) > 1 {
			data.Caption = strings.Trim(strings.TrimSpace(parts[1]), `"`)
		}
	}
}