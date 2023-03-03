package model

//
// curl "http://localhost:8989/api/v1/$ENDPOINT?apiKey=$APIKEY"
//

// Author - Stores struct of JSON response

type Author []struct {
	Id         int    `json:"id"`
	AuthorName string `json:"authorName"`
	Monitored  bool   `json:"monitored"`
	Statistics struct {
		BookCount      int     `json:"bookCount"`
		BookFileCount  int     `json:"bookFileCount"`
		TotalBookCount int     `json:"totalBookCount"`
		SizeOnDisk     int64   `json:"sizeOnDisk"`
		PercentOfBooks float32 `json:"percentOfBooks"`
	} `json:"statistics"`
}

type Book []struct {
	Monitored  bool `json:"monitored"`
	Grabbed    bool `json:"grabbed"`
	Statistics struct {
		BookFileCount int `json:"bookFileCount"`
	} `json:"statistics"`
}
