// Package model contains typed representations of *arr API responses.
package model

// Subtitle - Stores struct of JSON response
// https://github.com/morpheus65535/bazarr/blob/master/bazarr/api/swaggerui.py#L12
type Subtitle []struct {
	Size     int64  `json:"file_size"`
	Language string `json:"name"`
	Code2    string `json:"code2"`
	Code3    string `json:"code3"`
}

// BazarrSeries is the response from bazarr's series endpoint
// (/api/swagger.json#/definitions/SeriesGetResponse,
// bazarr/api/series/series.py).
type BazarrSeries struct {
	Data []struct {
		ID        int  `json:"sonarrSeriesId"`
		Monitored bool `json:"monitored"`
	} `json:"data"`
}

// BazarrEpisodes is the response from bazarr's episodes endpoint.
type BazarrEpisodes struct {
	Data []struct {
		ID               int      `json:"sonarrEpisodeId"`
		SeriesID         int      `json:"sonarrSeriesId"`
		Monitored        bool     `json:"monitored"`
		Subtitles        Subtitle `json:"subtitles"`
		MissingSubtitles Subtitle `json:"missing_subtitles"`
	} `json:"data"`
}

// BazarrMovies is the response from bazarr's movies endpoint
// (/api/swagger.json#/definitions/MoviesGetResponse,
// bazarr/api/movies/movies.py).
type BazarrMovies struct {
	Data []struct {
		ID               int      `json:"radarrId"`
		Monitored        bool     `json:"monitored"`
		Subtitles        Subtitle `json:"subtitles"`
		MissingSubtitles Subtitle `json:"missing_subtitles"`
	} `json:"data"`
}

// BazarrHistory is the response from bazarr's history endpoints.
type BazarrHistory struct {
	Data []struct {
		Score    string `json:"score"`
		Provider string `json:"provider"`
	} `json:"data"`
	TotalRecords int `json:"total"`
}

// BazarrHealth is the response from bazarr's system/health endpoint.
type BazarrHealth struct {
	Data []struct {
		Object string `json:"object"`
		Issue  string `json:"issue"`
	} `json:"data"`
}

// BazarrBadges is the subset of bazarr's badges endpoint the collector uses
// (the endpoint also returns wanted counts, health-issue and announcement
// totals, which are covered by other collectors/endpoints).
type BazarrBadges struct {
	Episodes      int    `json:"episodes"`  // episodes with missing subtitles
	Providers     int    `json:"providers"` // currently throttled providers
	SonarrSignalr string `json:"sonarr_signalr"`
	RadarrSignalr string `json:"radarr_signalr"`
}

// BazarrStatus is the response from bazarr's system/status endpoint.
type BazarrStatus struct {
	Data struct {
		Version       string  `json:"bazarr_version"`
		PythonVersion string  `json:"python_version"`
		StartTime     float32 `json:"start_time"`
	} `json:"data"`
}
