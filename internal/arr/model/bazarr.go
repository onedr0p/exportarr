package model

// Subtitle - Stores struct of JSON response
// https://github.com/morpheus65535/bazarr/blob/master/bazarr/api/swaggerui.py#L12
type Subtitle []struct {
	Size     int64  `json:"file_size"`
	Language string `json:"name"`
	Code2    string `json:"code2"`
	Code3    string `json:"code3"`
}

// Series - Stores struct of JSON response
// /api/swagger.json#/definitions/SeriesGetResponse
// https://github.com/morpheus65535/bazarr/blob/master/bazarr/api/series/series.py
type BazarrSeries struct {
	Data []struct {
		Id        int  `json:"sonarrSeriesId"`
		Monitored bool `json:"monitored"`
	} `json:"data"`
}

type BazarrEpisodes struct {
	Data []struct {
		Id               int      `json:"sonarrEpisodeId"`
		SeriesId         int      `json:"sonarrSeriesId"`
		Monitored        bool     `json:"monitored"`
		Subtitles        Subtitle `json:"subtitles"`
		MissingSubtitles Subtitle `json:"missing_subtitles"`
	} `json:"data"`
}

// /api/swagger.json#/definitions/MoviesGetResponse
// https://github.com/morpheus65535/bazarr/blob/master/bazarr/api/movies/movies.py
type BazarrMovies struct {
	Data []struct {
		Id               int      `json:"radarrId"`
		Monitored        bool     `json:"monitored"`
		Subtitles        Subtitle `json:"subtitles"`
		MissingSubtitles Subtitle `json:"missing_subtitles"`
	} `json:"data"`
}

type BazarrHistory struct {
	Data []struct {
		Score    string `json:"score"`
		Provider string `json:"provider"`
	} `json:"data"`
	TotalRecords int `json:"total"`
}

type BazarrHealth struct {
	Data []struct {
		Object string `json:"object"`
		Issue  string `json:"issue"`
	} `json:"data"`
}

type BazarrStatus struct {
	Data struct {
		Version       string  `json:"bazarr_version"`
		PythonVersion string  `json:"python_version"`
		StartTime     float32 `json:"start_time"`
	} `json:"data"`
}
