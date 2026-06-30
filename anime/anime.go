package anime

import (
	"net/http"

	"github.com/Beesonn/dlkitgo/anime/providers"
)

type AnimeService struct {
	Client  *http.Client
	Hianime *providers.HianimeProvider
}

func NewAnime(client *http.Client) *AnimeService {
	service := &AnimeService{
		Client: client,
	}

	service.Hianime = providers.NewHianimeProvider(client)

	return service
}
