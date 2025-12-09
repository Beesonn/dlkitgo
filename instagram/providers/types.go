package providers

type MediaSource struct {
	URL       string `json:"url"`
	Type      string `json:"type"`
	Thumbnail string `json:"thumbnail"`
	Index     int    `json:"index"`
}

type InstaStreamResult struct {
	Caption  string        `json:"caption"`
	Username string        `json:"username"`
	Total    int           `json:"total"`
	Video    int           `json:"video"`
	Photo    int           `json:"photo"`
	Source   []MediaSource `json:"source"`
}
