package model

// Artist - Stores struct of JSON response
type Artist []struct {
	Id         int    `json:"id"`
	Status     string `json:"status"`
	Ended      bool   `json:"ended"`
	Monitored  bool   `json:"monitored"`
	Statistics struct {
		AlbumCount      int   `json:"albumCount"`
		TrackFileCount  int   `json:"trackFileCount"`
		TrackCount      int   `json:"trackCount"`
		TotalTrackCount int   `json:"totalTrackCount"`
		SizeOnDisk      int64 `json:"sizeOnDisk"`
	} `json:"statistics"`
	Genres           []string `json:"genres"`
	QualityProfileID int      `json:"qualityProfileId"`
}

// Album - Stores struct of JSON response
type Album []struct {
	Id        int      `json:"id"`
	Monitored bool     `json:"monitored"`
	Genres    []string `json:"genres"`
	Duration  int      `json:"duration"`
}

// SongFile - Stores struct of JSON response
type SongFile []struct {
	Size    int64 `json:"size"`
	Quality struct {
		Quality struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"quality"`
	} `json:"quality"`
}
