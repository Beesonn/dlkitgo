package dlkitgo

import (
	"net/http"
	"time"

	"github.com/Beesonn/dlkitgo/instagram"
	"github.com/Beesonn/dlkitgo/spotify"
)

type Dlkit struct {
	client    *http.Client
	Spotify   *spotify.SpotifyService
	Instagram *instagram.InstaService
}

func NewClient() *Dlkit {
	c := &Dlkit{
		client: &http.Client{Timeout: 10 * time.Second},
	}

	c.Spotify = spotify.NewSpotify(c.client)
	c.Instagram = instagram.NewInsta(c.client)

	return c
}
