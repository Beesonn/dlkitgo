package spotify 

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "errors"
)

type SearchResult struct {
    ID         string `json:"id"`
    URL string `json:"url"`
    Image      string `json:"image"`
    Duration   int    `json:"duration,omitempty"`
    Artists    string `json:"artists"`
    Type       string `json:"type"`
    Name       string `json:"name"`
    Album      string `json:"album,omitempty"`
    ReleaseDate string `json:"release_date"`
    PreviewURL string `json:"preview_url,omitempty"`
}

type SearchResponse struct {
    Query        string         `json:"query"`
    Type         string         `json:"type"`
    Limit        int            `json:"limit"`
    TotalResults int            `json:"total_results"`
    Results      []SearchResult `json:"results"`
}

func  (s *SpotifyService) Search(query string, searchType ...string) (*SearchResponse, error) {
    if query == "" {
        return nil, errors.New("query cannot be empty")
    }
    sType := "all"
    if len(searchType) > 0 && searchType[0] != "" {
        sType = searchType[0]
    }

    api := "https://meow.mangoi.in/search"
    queryParams := url.Values{}
    queryParams.Add("q", query)
    queryParams.Add("type", sType)
    
    fullURL := api + "?" + queryParams.Encode()
    
    resp, err := s.Client.Get(fullURL)
    if err != nil {
        return nil, errors.New("Something went wrong!")
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error: %d", resp.StatusCode)
    }
    
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    
    var results SearchResponse
    if err := json.Unmarshal(body, &results); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %w", err)
    }
    
    return &results, nil
}