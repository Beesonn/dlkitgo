package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://youtu.be/Zi_XLOBDo_Y?si=uKwtZAWYRB_MpNLt"

	info, err := client.Youtube.GetInfo(url)
	if err != nil {
		fmt.Println("ERROR: GetInfo failed:", err)
		return
	}

	// Check if it's a playlist
	if info.Type == "playlist" {
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("YouTube ID: %s\n", info.ID)
		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Image: %s\n", info.Image)
		fmt.Printf("Total Videos: %d\n", info.TotalPlaylists)
		fmt.Printf("URL: %s\n", info.URL)

		fmt.Printf("\nPlaylist Videos:\n")
		for i, item := range info.Playlist {
			fmt.Printf("Video %d:\n", i+1)
			fmt.Printf("  Name: %s\n", item.Name)
			fmt.Printf("  Channel Name: %s\n", item.ChannelName)
			fmt.Printf("  Duration: %d seconds\n", item.Duration)
			fmt.Printf("  URL: %s\n", item.URL)
			fmt.Println()
		}
	} else if info.Type == "video" {
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("Video ID: %s\n", info.ID)
		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Duration: %d seconds\n", info.Videos[0].Duration)
		fmt.Printf("Channel: %s\n", info.Videos[0].ChannelName)
		fmt.Printf("URL: %s\n", info.URL)
	} else if info.Type == "shorts" {
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("Shorts ID: %s\n", info.ID)
		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Duration: %d seconds\n", info.Videos[0].Duration)
		fmt.Printf("Channel: %s\n", info.Videos[0].ChannelName)
		fmt.Printf("URL: %s\n", info.URL)
	} else {
		fmt.Printf("Unknown type: %s\n", info.Type)
	}
}
