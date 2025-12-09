package instagram

import (
	"github.com/Beesonn/dlkitgo/instagram/providers"
	"net/http"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (providers.InstaStreamResult, error)
}

func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.FastVideoSave{Client: client},
	}
}
