package dlkitgo

import (
	"net"
	"net/http"
	"time"

	"github.com/Beesonn/dlkitgo/anime"
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
	Anime     *anime.AnimeService
}

func NewClient() *Dlkit {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     30,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	c := &Dlkit{
		Client: &http.Client{Timeout: 15555 * time.Second, Transport: transport},
	}

	c.Spotify = spotify.NewSpotify(c.Client)
	c.Instagram = instagram.NewInsta(c.Client)
	c.Youtube = youtube.NewTube(c.Client)
	c.Pinterest = pinterest.NewPin(c.Client)
	c.Anime = anime.NewAnime(c.Client)

	return c
}
