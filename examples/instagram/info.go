package main

import (
	"fmt"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()
	url := "https://www.instagram.com/reel/DKrA73pIjFn"
	info, err := client.Instagram.GetInfo(url)
	if err != nil {
		fmt.Printf("ERROR: GetInfo failed: %v", err)
	}

	fmt.Printf("Username: %s\n", info.Username)
	fmt.Printf("Caption: %s\n", info.Caption)
	fmt.Printf("Thumbnail: %s\n", info.Thumbnail)
	fmt.Printf("Date: %s\n", info.Date)
	fmt.Printf("Comments: %s\n", info.Comments)
}
