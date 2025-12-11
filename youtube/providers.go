package youtube 

import (
	"net/http"

  	"github.com/Beesonn/dlkitgo/youtube/providers"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (providers.YTResults, error)
}

func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.VidFly{Client: client},
	}
}