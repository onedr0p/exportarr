package model

//
// curl "http://localhost:8989/api/v3/$ENDPOINT?apiKey=$APIKEY"
//

// Series - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/Series
type Series []struct {
	Id         int       `json:"id"`
	Monitored  bool      `json:"monitored"`
	Seasons    []Seasons `json:"seasons"`
	Statistics struct {
		SeasonCount       int     `json:"seasonCount"`
		EpisodeFileCount  int     `json:"episodeFileCount"`
		EpisodeCount      int     `json:"episodeCount"`
		TotalEpisodeCount int     `json:"totalEpisodeCount"`
		SizeOnDisk        int64   `json:"sizeOnDisk"`
		PercentOfEpisodes float32 `json:"percentOfEpisodes"`
	} `json:"statistics"`
}

// Seasons - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/Seasons
type Seasons struct {
	Monitored         bool `json:"monitored"`
	EpisodeFileCount  int  `json:"episodeFileCount"`
	EpisodeCount      int  `json:"episodeCount"`
	TotalEpisodeCount int  `json:"totalEpisodeCount"`
	Statistics        struct {
		EpisodeFileCount  int     `json:"episodeFileCount"`
		EpisodeCount      int     `json:"episodeCount"`
		TotalEpisodeCount int     `json:"totalEpisodeCount"`
		SizeOnDisk        int64   `json:"sizeOnDisk"`
		PercentOfEpisodes float32 `json:"percentOfEpisodes"`
	} `json:"statistics"`
}

// Missing - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/Wanted-Missing
type Missing struct {
	TotalRecords int `json:"totalRecords"`
}

// CutoffUnmet - Stores struct of JSON response
// https://sonarr.tv/docs/api/#/Cutoff/get_api_v3_wanted_cutoff
type CutoffUnmet struct {
	TotalRecords int `json:"totalRecords"`
}

// EpisodeFile - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/EpisodeFile
type EpisodeFile []struct {
	Size    int64 `json:"size"`
	Quality struct {
		Quality struct {
			ID         int    `json:"id"`
			Name       string `json:"name"`
			Source     string `json:"source"`
			Resolution int    `json:"resolution"`
		} `json:"quality"`
	} `json:"quality"`
}

// Episode - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/Episode
type Episode []struct {
	Size      int    `json:"episodeFileId"`
	Title     string `json:"title"`
	HasFile   bool   `json:"hasFile"`
	Monitored bool   `json:"monitored"`
}

// TagSeries - Stores struct of JSON response for tag details
// https://sonarr.tv/docs/api/#/TagDetails/get_api_v3_tag_detail
type TagSeries []struct {
	ID        int    `json:"id"`
	Label     string `json:"label"`
	SeriesIds []int  `json:"seriesIds"`
}
