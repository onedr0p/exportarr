package model

// Movie - Stores struct of JSON response
type Movie []struct {
	Status    string `json:"status"`
	HasFile   bool   `json:"hasFile"`
	Available bool   `json:"isAvailable"`
	Monitored bool   `json:"monitored"`
	MovieFile struct {
		Edition string `json:"edition"`
		Size    int64  `json:"size"`
		Quality struct {
			Quality struct {
				Name string `json:"name"`
			} `json:"quality"`
		} `json:"quality"`
	} `json:"movieFile"`
	QualityProfileID int `json:"qualityProfileId"`
}

type TagMovies []struct {
	ID       int    `json:"id"`
	Label    string `json:"label"`
	MovieIds []int  `json:"movieIds"`
}
