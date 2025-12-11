package instagram

import (
	"net/http"

   "github.com/Beesonn/dlkitgo/instagram/providers"
)

type Provider interface {
	Name() string
	BaseURL() string
	Stream(url string) (providers.InstaStreamResult, error)
}

func DefaultProviders(client *http.Client) []Provider {
	return []Provider{
		&providers.FastVideoSave{Client: client},
		&providers.TheSocialCat{Client: client},
	}
}