package main

import (
	"fmt"
	"log"

	"github.com/Beesonn/dlkitgo"
)

func main() {
	client := dlkitgo.NewClient()

	results, err := client.Anime.Hianime.Search("Naruto")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d results\n", len(results))
	for i, result := range results {
		fmt.Printf("%d. %s (Type: %s, Episodes: %d)\n",
			i+1, result.Title, result.Type, result.Episodes)
		fmt.Printf("   ID: %s\n", result.ID)
	}

	if len(results) > 0 {
		info, err := client.Anime.Hianime.GetInfo(results[0].ID)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\nTitle: %s\n", info.Title)
		fmt.Printf("Japanese Title: %s\n", info.JapaneseTitle)
		fmt.Printf("Description: %s\n", info.Description[:200]+"...")
		fmt.Printf("Genres: %v\n", info.Genres)
		fmt.Printf("Status: %s\n", info.Status)
		fmt.Printf("Year: %s\n", info.Year)
		fmt.Printf("Type: %s\n", info.Type)
		fmt.Printf("Total Episodes: %d\n", len(info.Episodes))

		if len(info.Episodes) > 0 {
			fmt.Printf("\nFirst 5 episodes:\n")
			max := 5
			if len(info.Episodes) < max {
				max = len(info.Episodes)
			}
			for i := 0; i < max; i++ {
				ep := info.Episodes[i]
				fmt.Printf("  %s. %s (ID: %s)\n", ep.EpisodeNum, ep.Title, ep.ID)
			}

			sources, err := client.Anime.Hianime.Stream(results[0].ID, info.Episodes[0].EpisodeNum)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("\nSources for Episode %s: %d\n", info.Episodes[0].EpisodeNum, len(sources))
			for i, source := range sources {
				fmt.Printf("%d. Quality: %s, URL: %s\n",
					i+1, source.Quality, source.URL)
			}
		}
	}

	homeData, err := client.Anime.Hianime.Home()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nFeatured Anime: %d\n", len(homeData.Featured))
	for i, item := range homeData.Featured {
		if i < 3 {
			fmt.Printf("%d. %s (Slug: %s)\n", i+1, item.Anime.Title, item.Anime.Slug)
		}
	}
	if len(homeData.Featured) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.Featured)-3)
	}

	fmt.Printf("\nTrending Anime: %d\n", len(homeData.Trending.Animes))
	for i, anime := range homeData.Trending.Animes {
		if i < 3 {
			fmt.Printf("%d. %s (Views: %d)\n", i+1, anime.Title, anime.RecentViewsCount)
		}
	}
	if len(homeData.Trending.Animes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.Trending.Animes)-3)
	}

	fmt.Printf("\nPopular Anime: %d\n", len(homeData.Popular.Animes))
	for i, anime := range homeData.Popular.Animes {
		if i < 3 {
			fmt.Printf("%d. %s (Score: %s)\n", i+1, anime.Title, anime.Score)
		}
	}
	if len(homeData.Popular.Animes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.Popular.Animes)-3)
	}

	fmt.Printf("\nCurrently Airing: %d\n", len(homeData.CurrentlyAiring.Animes))
	for i, anime := range homeData.CurrentlyAiring.Animes {
		if i < 3 {
			fmt.Printf("%d. %s (Status: %s)\n", i+1, anime.Title, anime.Status)
		}
	}
	if len(homeData.CurrentlyAiring.Animes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.CurrentlyAiring.Animes)-3)
	}

	fmt.Printf("\nFinished Airing: %d\n", len(homeData.FinishedAiring.Animes))
	for i, anime := range homeData.FinishedAiring.Animes {
		if i < 3 {
			fmt.Printf("%d. %s (Score: %s)\n", i+1, anime.Title, anime.Score)
		}
	}
	if len(homeData.FinishedAiring.Animes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.FinishedAiring.Animes)-3)
	}

	fmt.Printf("\nLatest Anime: %d\n", len(homeData.LatestAnime.Animes))
	for i, anime := range homeData.LatestAnime.Animes {
		if i < 3 {
			fmt.Printf("%d. %s (Type: %s)\n", i+1, anime.Title, anime.Type)
		}
	}
	if len(homeData.LatestAnime.Animes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.LatestAnime.Animes)-3)
	}

	fmt.Printf("\nLatest Episodes: %d\n", len(homeData.LatestEpisodes.Episodes))
	for i, ep := range homeData.LatestEpisodes.Episodes {
		if i < 3 {
			fmt.Printf("%d. %s - %s\n", i+1, ep.AnimeInfo.Title, ep.Title)
		}
	}
	if len(homeData.LatestEpisodes.Episodes) > 3 {
		fmt.Printf("... and %d more\n", len(homeData.LatestEpisodes.Episodes)-3)
	}
}
