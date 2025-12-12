package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://open.spotify.com/track/0B6ZJaS3I891FP8Ewx43Oh"

	info, err := client.Spotify.GetInfo(url)
	if err != nil {
		fmt.Println("ERROR: GetInfo failed:", err)
		return
	}

	if len(info.Tracks) == 0 {
		fmt.Println("ERROR: No track information available")
		return
	}

	fmt.Printf("Type: %s\n", info.Type)
	fmt.Printf("Spotify ID: %s\n", info.SpotifyID)
	fmt.Printf("Name: %s\n", info.Name)
	fmt.Printf("Artist: %s\n", info.Artist)
	fmt.Printf("Preview URL: %s\n", info.PreviewURL)
	fmt.Printf("Image: %s\n", info.Image)
	fmt.Printf("Duration: %d seconds\n", info.Duration)
	fmt.Printf("Release Date: %s\n", info.ReleaseDate)
	fmt.Printf("URL: %s\n", info.URL)

	fmt.Printf("\nTrack Details:\n")
	for i, track := range info.Tracks {
		fmt.Printf("Track %d:\n", i+1)
		fmt.Printf("  Name: %s\n", track.Name)
		fmt.Printf("  Artist: %s\n", track.Artist)
		fmt.Printf("  Preview URL: %s\n", track.PreviewURL)
		fmt.Printf("  Image: %s\n", track.Image)
		fmt.Printf("  Duration: %d seconds\n", track.Duration)
		fmt.Printf("  Release Date: %s\n", track.ReleaseDate)
		fmt.Printf("  URL: %s\n", track.URL)
	}
}
