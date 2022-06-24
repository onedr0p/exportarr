package model

// Series - Stores struct of JSON response
type Series []struct {
	Id         int       `json:"id"`
	Monitored  bool      `json:"monitored"`
	Seasons    []Seasons `json:"seasons"`
	Statistics struct {
		SeasonCount       int   `json:"seasonCount"`
		EpisodeFileCount  int   `json:"episodeFileCount"`
		EpisodeCount      int   `json:"episodeCount"`
		TotalEpisodeCount int   `json:"totalEpisodeCount"`
		SizeOnDisk        int64 `json:"sizeOnDisk"`
	} `json:"statistics"`
}

// Seasons - Stores struct of JSON response
type Seasons struct {
	Monitored bool `json:"monitored"`
}

// Missing - Stores struct of JSON response
type Missing struct {
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

// EpisodeFile - Stores struct of JSON response
// https://github.com/Sonarr/Sonarr/wiki/Episode
type Episode []struct {
	Size      int    `json:"episodeFileId"`
	Title     string `json:"title"`
	HasFile   bool   `json:"hasFile"`
	Monitored bool   `json:"monitored"`
}
