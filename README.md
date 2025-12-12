<p align="center">
    <img src="https://github.com/Beesonn/dlkitgo/raw/main/logo.png" alt="dlkitgo" width="256">
</p>


# dlkitgo

A powerful Golang library for downloading content from popular platforms and third-party services. Completely free and open-source.

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/Beesonn/dlkitgo/dlkitgo"
)

func main() {
    client := dlkitgo.NewClient()
    url := "https://open.spotify.com/track/0B6ZJaS3I891FP8Ewx43Oh"
    stream, err := client.Spotify.Stream(url)
    if err != nil {
        fmt.Printf("ERROR: Stream failed: %v", err)
        return
    }

    if len(stream.Source) == 0 {
        fmt.Println("ERROR: No stream sources available")
        return
    }
    
    fmt.Printf("Artists: %s\n", stream.Source[0].Artist)
    fmt.Printf("Title: %s\n", stream.Source[0].Title)
    fmt.Printf("Image: %s\n", stream.Source[0].Image)
    fmt.Printf("Stream URL: %s\n", stream.Source[0].URL)
}
```

## Installation

```bash
go get github.com/Beesonn/dlkitgo
```


## Examples

Check out our examples for different platforms:

* Spotify: [example](examples/spotify)
* Instagram: [examples](examples/instagram)
* Coming soon...

## Provider Request

If you would like to see an example for a platform that is not listed here, you can request it by opening an issue.

Make a new [issue](https://github.com/Beesonn/dlkitgo/issues/new?assignees=&labels=new+provider+request&template=request.yml) with the name of the provider on the title, as well as a link to the provider in the body paragraph.

## Contributing

We welcome contributions! Feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b new`)
3. Commit your changes (`git commit -m 'Add some new'`)
4. Push to the branch (`git push origin new`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

If you encounter any problems or have questions, please file an issue on the GitHub [issue](https://github.com/Beesonn/dlkitgo/issues).