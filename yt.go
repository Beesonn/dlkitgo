package dlkitgo

import (
	"encoding/json"
	"fmt"
)

func YtTest() {
	client := dlkitgo.NewClient()
	url := "https://youtu.be/YVkUvmDQ3HY?si=WX_soUJPp66u-mcF"
	stream, err := client.Youtube.Stream(url)
	if err != nil {
		fmt.Printf("ERROR: Stream failed: %v", err)
	}

	jsonData, err := json.MarshalIndent(stream, "", "  ")
	if err != nil {
		fmt.Printf("ERROR: Failed to marshal to JSON: %v\n", err)
		return
	}

	fmt.Println(string(jsonData))
}