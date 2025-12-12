package dlkitgo_test

import (
	"encoding/json"
	"fmt"
)

func TestSpInfo() {
	client := dlkitgo.NewClient()
	url := "https://open.spotify.com/track/0B6ZJaS3I891FP8Ewx43Oh"

	info, err := client.Spotify.GetInfo(url)
	if err != nil {
		fmt.Printf("ERROR: GetInfo failed: %v\n", err)
		return
	}

	if len(info.Tracks) == 0 {
		fmt.Println("ERROR: No track information available")
		return
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		fmt.Printf("ERROR: Failed to marshal to JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}