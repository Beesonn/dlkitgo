package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()

	fmt.Println("Searching for YouTube video...")
	search, err := client.Youtube.Search("sheriya", 1)
	if err != nil {
		fmt.Println("ERROR: Search failed:", err)
		return
	}

	if len(search.Results) == 0 {
		fmt.Println("ERROR: No results found")
		return
	}

	video := search.Results[0]
	fmt.Printf("ID: %s\n", video.ID)
	fmt.Printf("URL: %s\n", video.URL)
	fmt.Printf("Title: %s\n", video.Name)
	fmt.Printf("Channel: %s\n", video.Channel)
	fmt.Printf("Duration: %d seconds\n", video.Duration)
	fmt.Printf("Thumbnail: %s\n", video.Image)
}
