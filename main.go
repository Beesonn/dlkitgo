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
	url := "https://www.instagram.com/reel/DN6IZn3Eh2z/?igsh=MWVxZWM2dWlwMnJudQ=="
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