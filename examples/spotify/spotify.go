package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo/dlkitgo"
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
	fmt.Printf("Title: %s\n", track.Name)
	fmt.Printf("Artist: %s\n", track.Artists)
	fmt.Printf("Duration: %d\n", track.Duration)

	stream, err := client.Spotify.Stream(track.URL)
	if err != nil {
		fmt.Println("ERROR: Stream failed: %v", err)
	}

	if len(stream.Source) == 0 {
		fmt.Println("ERROR: No stream sources available")
	}

	fmt.Printf("Stream URL: %s\n", stream.Source[0].URL)
}
