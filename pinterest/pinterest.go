package pinterest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Beesonn/dlkitgo/pinterest/providers"
)

type PinService struct {
	Client    *http.Client
	Providers []Provider
}

func NewPin(client *http.Client) *PinService {
	return &PinService{
		Client:    client,
		Providers: DefaultProviders(client),
	}
}

func (p *PinService) Stream(url string) (providers.PinResults, error) {
	if url == "" {
		return providers.PinResults{}, errors.New("url cannot be empty")
	}

	for _, provider := range p.Providers {
		res, err := provider.Stream(url)
		if err == nil {
			return res, nil
		}
		fmt.Printf("Provider '%s' failed to stream: %v\n", provider.Name(), err)
	}
	return providers.PinResults{}, errors.New("all configured providers failed to stream the content")
}
