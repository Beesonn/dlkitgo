package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Beesonn/dlkitgo/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()

	//url := "https://open.spotify.com/playlist/37i9dQZF1DXcBWIGoYBM5M" // Today's Top Hits
	url := "https://www.instagram.com/stories/followers_kombat/3783495052984245483?utm_source=ig_story_item_share&igsh=MmoyOXEyNXNqZjk4"
	// url := "https://open.spotify.com/album/4yP0hdKOZPNshxUOjY0cZj"
	r, _ := client.Instagram.GetInfo(url)
	fmt.Println(r)
	result, err := client.Instagram.Stream(url)
	if err != nil {
		log.Fatal("Error:", err)
	}

	// Pretty JSON
	jsonData, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(jsonData))
}
