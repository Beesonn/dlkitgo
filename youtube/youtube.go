package youtube 

import (
	"errors"
	"net/http"
   "fmt"
  
  "github.com/Beesonn/dlkitgo/youtube/providers"
)

type TubeService struct {
	Client    *http.Client
	Providers []Provider
}

func NewTube(client *http.Client) *TubeService {
	return &TubeService{
		Client:    client,
		Providers: DefaultProviders(client),
	}
}


func (t *TubeService) Stream(url string) (providers.YTResults, error) {
	if url == "" {
		return providers.YTResults{}, errors.New("url cannot be empty")
	}
  
   if !providers.IsYouTubeURL(url) {
		return providers.YTResults{}, fmt.Errorf("invalid YouTube URL")
	}

	for _, provider := range t.Providers {
		res, err := provider.Stream(url)
		if err == nil {
			return res, nil
		}
		fmt.Printf("Provider '%s' failed to stream: %v\n", provider.Name(), err)
	}
	return providers.YTResults{}, errors.New("all configured providers failed to stream the content")
}