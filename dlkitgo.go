package dlkitgo

import (
    "net/http"
    "time"

    "github.com/Beesonn/dlkitgo/spotify"
)

type Dlkit struct {
    client  *http.Client
    Spotify *spotify.SpotifyService
}

func NewClient() *Dlkit {
    c := &Dlkit{
        client: &http.Client{ Timeout: 10 * time.Second },
    }

    c.Spotify = spotify.NewSpotify(c.client)

    return c
}