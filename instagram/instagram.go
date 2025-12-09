package instagram

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Beesonn/dlkitgo/instagram/providers"
)

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
		return providers.InstaStreamResult{}, errors.New("Invalid Instagram URL")
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
		fmt.Printf("Provider '%s' failed to stream: %v\n", provider.Name(), err)
	}
	return providers.InstaStreamResult{}, errors.New("all configured providers failed to stream the content")
}
