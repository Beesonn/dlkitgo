package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()

	fmt.Println("Searching for track...")
	search, err := client.Spotify.Search("never gonna give you up", "track")
	if err != nil {
		fmt.Println("ERROR: Search failed: %v", err)
	}

	if len(search.Results) == 0 {
		fmt.Println("ERROR: No results found")
	}

	track := search.Results[0]
	fmt.Printf("ID: %s\n", track.ID)
	fmt.Printf("URL: %s\n", track.URL)
	fmt.Printf("Title: %s\n", track.Name)
	fmt.Printf("Artist: %s\n", track.Artists)
	fmt.Printf("Duration: %d\n", track.Duration)
	fmt.Printf("Image: %s\n", track.Image)
	fmt.Printf("Release: %s\n", track.ReleaseDate)
	fmt.Printf("Album: %s\n", track.Album)
}
