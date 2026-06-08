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

	if info.Type == "video" {
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("Video ID: %s\n", info.ID)
		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Duration: %d seconds\n", info.Videos[0].Duration)
		fmt.Printf("URL: %s\n", info.URL)
	} else if info.Type == "shorts" {
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("Shorts ID: %s\n", info.ID)
		fmt.Printf("Name: %s\n", info.Name)
		fmt.Printf("Duration: %d seconds\n", info.Videos[0].Duration)
		fmt.Printf("URL: %s\n", info.URL)
	} else {
		fmt.Printf("Unknown type: %s\n", info.Type)
	}
}