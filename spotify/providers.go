package spotify

import (
	"net/http"

	"github.com/Beesonn/dlkitgo/spotify/providers"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (string, error)
}

// Provider list
func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.Downloaderize{Client: client},
		&providers.SpotMate{Client: client},
	}
}
