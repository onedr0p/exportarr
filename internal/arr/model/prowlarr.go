package model

// Indexer is the response from prowlarr's indexer endpoint.
type Indexer []struct {
	Name     string `json:"name"`
	SortName string `json:"sortName"`
	Enabled  bool   `json:"enable"`
	Fields   []struct {
		Name string `json:"name"`
		// Value has multiple types, depending on the field, so it
		// must be typecast at the call site.
		Value any `json:"value"`
	} `json:"fields"`
}

// IndexerStats holds per-indexer query/grab counters.
type IndexerStats struct {
	Name                      string `json:"indexerName"`
	AverageResponseTime       int    `json:"averageResponseTime"`
	NumberOfQueries           int    `json:"numberOfQueries"`
	NumberOfGrabs             int    `json:"numberOfGrabs"`
	NumberOfRssQueries        int    `json:"numberOfRssQueries"`
	NumberOfAuthQueries       int    `json:"numberOfAuthQueries"`
	NumberOfFailedQueries     int    `json:"numberOfFailedQueries"`
	NumberOfFailedGrabs       int    `json:"numberOfFailedGrabs"`
	NumberOfFailedRssQueries  int    `json:"numberOfFailedRssQueries"`
	NumberOfFailedAuthQueries int    `json:"numberOfFailedAuthQueries"`
}

// UserAgentStats holds per-user-agent query/grab counters.
type UserAgentStats struct {
	UserAgent       string `json:"userAgent"`
	NumberOfQueries int    `json:"numberOfQueries"`
	NumberOfGrabs   int    `json:"numberOfGrabs"`
}

// IndexerStatResponse is the response from prowlarr's indexerstats endpoint.
type IndexerStatResponse struct {
	Indexers   []IndexerStats   `json:"indexers"`
	UserAgents []UserAgentStats `json:"userAgents"`
}
