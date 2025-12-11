package providers

import "regexp"

type YTSource struct {
	URL      string `json:"url"`
	Duration int    `json:"duration"`
	Type     string `json:"type"`
	Quality  string `json:"quality"`
}

type YTResults struct {
	Caption   string     `json:"caption"`
	Thumbnail string     `json:"thumbnail"`
	Duration  int        `json:"duration"`
	Source    []YTSource `json:"sources"`
}

func IsYouTubeURL(url string) bool {
	patterns := []string{
		`^(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/(?:watch\?v=|embed\/|v\/|shorts\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})`,
		`^(?:https?:\/\/)?(?:www\.)?youtube\.com\/live\/([a-zA-Z0-9_-]+)`,
		`^(?:https?:\/\/)?(?:www\.)?youtube\.com\/(?:c|channel|user)\/[a-zA-Z0-9_-]+`,
	}

	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, url)
		if matched {
			return true
		}
	}
	return false
}
