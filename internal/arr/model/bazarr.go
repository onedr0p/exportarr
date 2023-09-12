package model

// Subtitle - Stores struct of JSON response
type Subtitle []struct {
	Status       string `json:"status"`
	HasFile      bool   `json:"hasFile"`
	Available    bool   `json:"isAvailable"`
	Monitored    bool   `json:"monitored"`
	SubtitleFile struct {
		Size     int64 `json:"size"`
		Language struct {
			Name string `json:"name"`
		} `json:"language"`
	} `json:"subtitleFile"`
}
