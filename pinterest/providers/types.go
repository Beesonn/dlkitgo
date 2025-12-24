package providers

type PinSource struct {
	URL     string `json:"url"`
	Type    string `json:"type"`
	Quality string `json:"quality"`
}

type PinResults struct {
	Title     string      `json:"title"`
	Thumbnail string      `json:"thumbnail"`
	Source    []PinSource `json:"source"`
}
