package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://youtu.be/YVkUvmDQ3HY?si=WX_soUJPp66u-mcF"
	stream, err := client.Youtube.Stream(url)
	if err != nil {
		fmt.Printf("ERROR: Stream failed: %v", err)
	}

	fmt.Printf("Title: %s\n", stream.Caption)
	fmt.Printf("Duration: %d\n", stream.Duration)
	fmt.Printf("Thumbnail: %s\n", stream.Thumbnail)
	fmt.Printf("Quality: %s\n", stream.Source[0].Quality)
	fmt.Printf("URL: %s\n", stream.Source[0].URL)
}
