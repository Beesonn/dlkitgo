package instagram

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Beesonn/dlkitgo/instagram/providers"
)

type InstaService struct {
	Client        *http.Client
	Providers     []Provider
	FastVideoSave Provider
	TheSocialCat  Provider
}

func NewInsta(client *http.Client) *InstaService {
	service := &InstaService{
		Client: client,
	}

	service.Providers = DefaultProviders(client)

	for _, provider := range service.Providers {
		if provider.Name() == "fastvideosave" {
			service.FastVideoSave = provider
		} else if provider.Name() == "thesocialcat" {
			service.TheSocialCat = provider
		}
	}

	return service
}

func (i *InstaService) GetProvider(name string) (Provider, error) {
	if name == "" {
		return nil, errors.New("please provide the provider name")
	}
	for _, provider := range i.Providers {
		fmt.Println(provider.Name())
		if provider.Name() == strings.ToLower(strings.TrimSpace(name)) {
			return provider, nil
		}
	}
	return nil, errors.New("sorry provider not found")
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
