package dlkitgo_test

import (
	"fmt"

   "github.com/Beesonn/dlkitgo"
)

func InstaByName() {
	client := dlkitgo.NewClient()
	url := "https://www.instagram.com/reel/DKrA73pIjFn"
	insta, _ := client.Instagram.GetProvider("thesocialcat")
	stream, err := insta.Stream(url)
	if err != nil {
		fmt.Printf("ERROR: Stream failed: %v", err)
	}

	if len(stream.Source) == 0 {
		fmt.Println("ERROR: No stream sources available")
	}
	fmt.Printf("From: %s\n", stream.Username)
	fmt.Printf("Caption: %s\n", stream.Caption)
	fmt.Printf("Thumbnail: %s\n", stream.Source[0].Thumbnail)
	fmt.Printf("Stream URL: %s\n", stream.Source[0].URL)
}