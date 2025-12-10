package spotify

import (
	"github.com/Beesonn/dlkitgo/spotify/providers"
	"net/http"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (string, error)
}

// Provider list
func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.SpotMate{Client: client},
	}
}
