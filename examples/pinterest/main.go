package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://pin.it/23ujuAyAW"
	stream, err := client.Pinterest.Stream(url)
	if err != nil {
		fmt.Println("ERROR: Stream failed:", err)
	}

	if len(stream.Source) == 0 {
		fmt.Println("ERROR: No stream sources available")
	}
	fmt.Printf("Title: %s\n", stream.Title)
	fmt.Printf("Thumbnail: %s\n", stream.Thumbnail)
	fmt.Printf("Stream URL: %s\n", stream.Source[0].URL)
}
