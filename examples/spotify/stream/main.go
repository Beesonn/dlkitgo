package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://open.spotify.com/track/0B6ZJaS3I891FP8Ewx43Oh"
	stream, err := client.Spotify.Stream(url)
	if err != nil {
		fmt.Println("ERROR: Stream failed:", err)
	}

	if len(stream.Source) == 0 {
		fmt.Println("ERROR: No stream sources available")
	}
	fmt.Printf("Artists: %s\n", stream.Source[0].Artist)
	fmt.Printf("Title: %s\n", stream.Source[0].Title)
	fmt.Printf("Image: %s\n", stream.Source[0].Image)
	fmt.Printf("Stream URL: %s\n", stream.Source[0].URL)
}
