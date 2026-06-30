package providers

type AnimeSearchResult struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Image    string `json:"image"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Episodes int    `json:"episodes"`
}

type AnimeInfo struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	JapaneseTitle string    `json:"japaneseTitle"`
	Image         string    `json:"image"`
	Description   string    `json:"description"`
	Type          string    `json:"type"`
	Status        string    `json:"status"`
	Year          string    `json:"year"`
	Genres        []string  `json:"genres"`
	Episodes      []Episode `json:"episodes"`
}

type Episode struct {
	ID         string   `json:"id"`
	EpisodeNum string   `json:"episodeNum"`
	Title      string   `json:"title"`
	SubLinks   []string `json:"subLinks"`
	DubLinks   []string `json:"dubLinks"`
}

type VideoSource struct {
	URL     string `json:"url"`
	Quality string `json:"quality"`
	IsM3U8  bool   `json:"isM3U8"`
}

type HomeAnime struct {
	ID               string    `json:"_id"`
	Title            string    `json:"title"`
	Japanese         string    `json:"Japanese"`
	English          string    `json:"English"`
	Score            string    `json:"Score"`
	Status           string    `json:"Status"`
	Type             string    `json:"Type"`
	Genres           []string  `json:"genres"`
	Image            string    `json:"image"`
	LandScapeImage   string    `json:"landScapeImage"`
	MalID            int       `json:"mal_id"`
	Synopsis         string    `json:"synopsis"`
	Slug             string    `json:"slug"`
	TotalSubbed      int       `json:"totalSubbed"`
	TotalDubbed      int       `json:"totalDubbed"`
	RecentViewsCount int       `json:"recentViewsCount"`
	Episodes         []Episode `json:"episodes"`
}

type HomeFeaturedItem struct {
	Anime HomeAnime `json:"anime"`
}

type HomeTrending struct {
	Animes []HomeAnime `json:"animes"`
}

type HomePopular struct {
	Animes []HomeAnime `json:"animes"`
}

type HomeCurrentlyAiring struct {
	Animes []HomeAnime `json:"animes"`
}

type HomeFinishedAiring struct {
	Animes []HomeAnime `json:"animes"`
}

type HomeLatestAnime struct {
	Animes []HomeAnime `json:"animes"`
}

type LatestEpisode struct {
	ID         string `json:"_id"`
	AnimeID    string `json:"anime_id"`
	EpisodeNum int    `json:"episodeNumber"`
	Title      string `json:"title"`
	Link       struct {
		Sub []string `json:"sub"`
		Dub []string `json:"dub"`
	} `json:"link"`
	CreatedAt string `json:"createdAt"`
	Slug      string `json:"slug"`
	AnimeInfo struct {
		Title          string   `json:"title"`
		MalID          int      `json:"mal_id"`
		Image          string   `json:"image"`
		LandScapeImage string   `json:"landScapeImage"`
		Synopsis       string   `json:"synopsis"`
		Genres         []string `json:"genres"`
		Japanese       string   `json:"Japanese"`
		Type           string   `json:"Type"`
		Status         string   `json:"Status"`
		Aired          string   `json:"Aired"`
		Duration       string   `json:"Duration"`
		Score          string   `json:"Score"`
		TotalSubbed    int      `json:"totalSubbed"`
		TotalDubbed    int      `json:"totalDubbed"`
		Slug           string   `json:"slug"`
	} `json:"anime_info"`
}

type HomeLatestEpisodes struct {
	Episodes []LatestEpisode `json:"episodes"`
}

type HomeData struct {
	Featured        []HomeFeaturedItem  `json:"featured"`
	Trending        HomeTrending        `json:"trending"`
	Popular         HomePopular         `json:"popular"`
	CurrentlyAiring HomeCurrentlyAiring `json:"currentlyAiring"`
	FinishedAiring  HomeFinishedAiring  `json:"finishedAiring"`
	LatestAnime     HomeLatestAnime     `json:"latestAnime"`
	LatestEpisodes  HomeLatestEpisodes  `json:"latestEpisodes"`
}
