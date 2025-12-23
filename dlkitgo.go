package dlkitgo

import (
	"net/http"

	"github.com/Beesonn/dlkitgo/instagram"
	"github.com/Beesonn/dlkitgo/spotify"
	"github.com/Beesonn/dlkitgo/youtube"
)

type Dlkit struct {
	Client    *http.Client
	Spotify   *spotify.SpotifyService
	Instagram *instagram.InstaService
	Youtube   *youtube.TubeService
}

func NewClient() *Dlkit {
	c := &Dlkit{
		Client: &http.Client{},
	}

	c.Spotify = spotify.NewSpotify(c.Client)
	c.Instagram = instagram.NewInsta(c.Client)
	c.Youtube = youtube.NewTube(c.Client)

	return c
}
