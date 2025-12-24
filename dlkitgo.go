package dlkitgo

import (
	"net/http"
	"time"

	"github.com/Beesonn/dlkitgo/instagram"
	"github.com/Beesonn/dlkitgo/pinterest"
	"github.com/Beesonn/dlkitgo/spotify"
	"github.com/Beesonn/dlkitgo/youtube"
)

type Dlkit struct {
	Client    *http.Client
	Spotify   *spotify.SpotifyService
	Instagram *instagram.InstaService
	Youtube   *youtube.TubeService
	Pinterest *pinterest.PinService
}

func NewClient() *Dlkit {
	c := &Dlkit{
		Client: &http.Client{Timeout: 15 * time.Second},
	}

	c.Spotify = spotify.NewSpotify(c.Client)
	c.Instagram = instagram.NewInsta(c.Client)
	c.Youtube = youtube.NewTube(c.Client)
	c.Pinterest = pinterest.NewPin(c.Client)

	return c
}
