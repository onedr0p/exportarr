package model

// Movie - Stores struct of JSON response
type Movie []struct {
	Status    string `json:"status"`
	HasFile   bool   `json:"hasFile"`
	Monitored bool   `json:"monitored"`
	MovieFile struct {
		Size    int64 `json:"size"`
		Quality struct {
			Quality struct {
				Name string `json:"name"`
			} `json:"quality"`
		} `json:"quality"`
	} `json:"movieFile"`
	QualityProfileID int `json:"qualityProfileId"`
}
