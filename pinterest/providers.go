package pinterest

import (
	"net/http"

	"github.com/Beesonn/dlkitgo/pinterest/providers"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (providers.PinResults, error)
}

func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.SavePin{Client: client},
	}
}
