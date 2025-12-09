package instagram 

import (
	"errors"
	"net/http"

   "github.com/Beesonn/dlkitgo/instagram/providers"
)

type MediaSource struct {
	URL   string `json:"url"`
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type InstaStreamResult struct {
	Caption  string        `json:"caption"`
	Username string        `json:"username"`
	Total    int           `json:"total"`
	Video    int           `json:"video"`
	Photo    int           `json:"photo"`
	Source   []MediaSource `json:"source"`
}

type InstaService struct {
	Client    *http.Client
	Providers []Provider
}

func NewInsta(client *http.Client) *InstaService {
	return &InstaService{
		Client:    client,
		Providers: DefaultProviders(client),
	}
}

func (i *InstaService) Stream(url string) (providers.InstaStreamResult, error) {
	if url == "" {
		return providers.InstaStreamResult{}, errors.New("url cannot be empty")
	}
   info, err := i.GetInfo(url)
   if err != nil {
       return providers.InstaStreamResult{}, errors.New("url cannot be empty")
	}
	
	for _, provider := range i.Providers {
	    res, err := provider.Stream(url)
        if res.Caption == "" {
          res.Caption = info.Caption
        }
        if res.Username == "" {
          res.Username = info.Username
        }
        if err == nil {
          return res, nil
        }
    }
    return providers.InstaStreamResult{}, nil
}