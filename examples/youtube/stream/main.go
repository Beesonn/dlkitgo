package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://youtu.be/WTJSt4wP2ME?si=rFMdirx0T4ZhCRX1"
	stream, err := client.Youtube.Stream(url)
	if err != nil {
		fmt.Printf("ERROR: Stream failed: %v", err)
		return
	}

	// Print basic info
	fmt.Printf("Title: %s\n", stream.Caption)
	fmt.Printf("Duration: %d seconds\n", stream.Duration)
	fmt.Printf("Thumbnail: %s\n", stream.Thumbnail)

	// Print all available streams
	for i, source := range stream.Source {
		fmt.Printf("\n[Stream %d]\n", i+1)
		fmt.Printf("  Quality: %s\n", source.Quality)
		fmt.Printf("  URL: %s\n", source.URL)
		fmt.Printf("  Type: %s\n", source.Type)
	}
}
