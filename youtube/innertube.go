package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type InnerTubeClient struct {
	client  *http.Client
	headers map[string]string
	cookies map[string]string
}

var VisitorCookies = map[string]string{
	"VISITOR_INFO1_LIVE":       "CdBfWKlCOYY",
	"VISITOR_PRIVACY_METADATA": "CgJQSxIEGgAgZw%3D%3D",
	"PREF":                     "f4=4000000&f6=40000000&tz=America.New_York&f7=100",
	"GL":                       "US",
}

var WebClient = map[string]string{
	"clientName":    "WEB",
	"clientVersion": "2.20260227.01.00",
	"platform":      "DESKTOP",
	"hl":            "en",
	"gl":            "US",
}

var AndroidClient = map[string]string{
	"clientName":    "ANDROID",
	"clientVersion": "20.10.38",
}

const (
	VideosTabParams    = "EgZ2aWRlb3PyBgQKAjoA"
	ShortsTabParams    = "EgZzaG9ydHPyBgUKA5oBAA%3D%3D"
	PlaylistsTabParams = "EglwbGF5bGlzdHPyBgQKAkIA"
	SearchTabParams    = "EgZzZWFyY2jyBgQKA1oA"
)

const (
	InnerTubeSearchURL = "https://www.youtube.com/youtubei/v1/search"
	InnerTubeBrowseURL = "https://www.youtube.com/youtubei/v1/browse"
	InnerTubePlayerURL = "https://www.youtube.com/youtubei/v1/player"
)

type InnerTubeContext struct {
	Client map[string]string `json:"client"`
}

type InnerTubeSearchPayload struct {
	Context      InnerTubeContext `json:"context"`
	Query        string           `json:"query"`
	Params       string           `json:"params,omitempty"`
	Continuation string           `json:"continuation,omitempty"`
}

type InnerTubeBrowsePayload struct {
	Context      InnerTubeContext `json:"context"`
	BrowseID     string           `json:"browseId,omitempty"`
	Params       string           `json:"params,omitempty"`
	Continuation string           `json:"continuation,omitempty"`
	Query        string           `json:"query,omitempty"`
}

type InnerTubePlayerPayload struct {
	Context InnerTubeContext `json:"context"`
	VideoID string           `json:"videoId"`
}

func NewInnerTubeClient(httpClient *http.Client) *InnerTubeClient {
	if httpClient == nil {
		jar, _ := cookiejar.New(nil)
		httpClient = &http.Client{
			Jar: jar,
		}
	}

	client := &InnerTubeClient{
		client: httpClient,
		headers: map[string]string{
			"Accept-Language": "en-US,en;q=0.9",
			"Content-Type":    "application/json",
			"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36",
		},
		cookies: make(map[string]string),
	}

	client.SetCookies(VisitorCookies)

	return client
}

func NewInnerTubeClientWithCookies(httpClient *http.Client, cookies map[string]string) *InnerTubeClient {
	client := NewInnerTubeClient(httpClient)
	client.SetCookies(cookies)
	return client
}

func (it *InnerTubeClient) SetCookies(cookies map[string]string) {
	if len(cookies) == 0 {
		return
	}

	for k, v := range cookies {
		it.cookies[k] = v
	}

	if it.client.Jar != nil {
		parsedUrl, _ := url.Parse("https://youtube.com")
		var httpCookies []*http.Cookie
		for name, value := range cookies {
			httpCookies = append(httpCookies, &http.Cookie{
				Name:  name,
				Value: value,
				Path:  "/",
			})
		}
		it.client.Jar.SetCookies(parsedUrl, httpCookies)

		wwwUrl, _ := url.Parse("https://www.youtube.com")
		it.client.Jar.SetCookies(wwwUrl, httpCookies)
	}
}

func (it *InnerTubeClient) AddCookie(name, value string) {
	it.cookies[name] = value

	if it.client.Jar != nil {
		parsedUrl, _ := url.Parse("https://youtube.com")
		it.client.Jar.SetCookies(parsedUrl, []*http.Cookie{{
			Name:  name,
			Value: value,
			Path:  "/",
		}})
	}
}

func (it *InnerTubeClient) GetCookies() map[string]string {
	return it.cookies
}

func (it *InnerTubeClient) SetHeader(key, value string) {
	it.headers[key] = value
}

func (it *InnerTubeClient) doRequest(url string, payload interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range it.headers {
		req.Header.Set(k, v)
	}

	resp, err := it.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if it.client.Jar != nil {
		for _, cookie := range resp.Cookies() {
			it.cookies[cookie.Name] = cookie.Value
		}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

func (it *InnerTubeClient) Search(query string, params string, continuation string) (map[string]interface{}, error) {
	payload := InnerTubeSearchPayload{
		Context: InnerTubeContext{Client: WebClient},
		Query:   query,
	}
	if params != "" {
		payload.Params = params
	}
	if continuation != "" {
		payload.Continuation = continuation
	}

	return it.doRequest(InnerTubeSearchURL, payload)
}

func (it *InnerTubeClient) Browse(browseID string, params string, continuation string, query string) (map[string]interface{}, error) {
	payload := InnerTubeBrowsePayload{
		Context: InnerTubeContext{Client: WebClient},
	}

	if continuation != "" {
		payload.Continuation = continuation
	} else {
		payload.BrowseID = browseID
		if params != "" {
			payload.Params = params
		}
		if query != "" {
			payload.Query = query
		}
	}

	return it.doRequest(InnerTubeBrowseURL, payload)
}

func (it *InnerTubeClient) GetPlayer(videoID string) (map[string]interface{}, error) {
	payload := InnerTubePlayerPayload{
		Context: InnerTubeContext{Client: AndroidClient},
		VideoID: videoID,
	}

	return it.doRequest(InnerTubePlayerURL, payload)
}

func (it *InnerTubeClient) BrowseChannel(channelID string, continuation string) (map[string]interface{}, error) {
	return it.Browse(channelID, VideosTabParams, continuation, "")
}

func (it *InnerTubeClient) BrowseChannelShorts(channelID string, continuation string) (map[string]interface{}, error) {
	return it.Browse(channelID, ShortsTabParams, continuation, "")
}

func (it *InnerTubeClient) BrowseChannelPlaylists(channelID string, continuation string) (map[string]interface{}, error) {
	return it.Browse(channelID, PlaylistsTabParams, continuation, "")
}

func (it *InnerTubeClient) SearchChannel(channelID string, query string) (map[string]interface{}, error) {
	return it.Browse(channelID, SearchTabParams, "", query)
}

func (it *InnerTubeClient) BrowsePlaylist(playlistID string, continuation string) (map[string]interface{}, error) {
	browseID := playlistID
	if len(playlistID) < 2 || playlistID[:2] != "VL" {
		browseID = "VL" + playlistID
	}
	return it.Browse(browseID, "", continuation, "")
}