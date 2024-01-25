package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Status int

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

const (
	UNKNOWN Status = iota
	IDLE
	PAUSED
	DOWNLOADING
)

func (s Status) Float64() float64 {
	return float64(s)
}
func (s Status) String() string {
	switch s {
	case IDLE:
		return "Idle"
	case PAUSED:
		return "Paused"
	case DOWNLOADING:
		return "Downloading"
	default:
		return "Unknown"
	}
}

func StatusFromString(s string) Status {
	switch s {
	case "Idle":
		return IDLE
	case "Paused":
		return PAUSED
	case "Downloading":
		return DOWNLOADING
	default:
		return UNKNOWN
	}
}

type ServerStats struct {
	Total   int                   `json:"total"` // Total Data Downloaded in bytes
	Servers map[string]ServerStat `json:"servers"`
}

type ServerStat struct {
	Total           int    // Total Data Downloaded in bytes
	ArticlesTried   int    // Number of Articles Tried
	ArticlesSuccess int    // Number of Articles Successfully Downloaded
	DayParsed       string // Last Date Parsed
}

func (s *ServerStat) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Total           int            `json:"total"`            // Total Data Downloaded in bytes
		ArticlesTried   map[string]int `json:"articles_tried"`   // Number of Articles Tried (YYYY-MM-DD -> count)
		ArticlesSuccess map[string]int `json:"articles_success"` // Number of Articles Successfully Downloaded (YYYY-MM-DD -> count)
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	d, tried := latestStat(tmp.ArticlesTried)
	_, success := latestStat(tmp.ArticlesSuccess)
	s.Total = tmp.Total
	s.ArticlesTried = tried
	s.ArticlesSuccess = success
	s.DayParsed = d
	return nil
}

// QueueStatsResponse is the response from the sabnzbd queue endpoint
// Paused vs PausedAll -- as best I can tell, Paused is
// "pause the queue but finish anything in flight"
// PausedAll is "hard pause, including pausing in progress downloads"
type QueueStats struct {
	Version                    string        // Sabnzbd Version
	Paused                     bool          // Is the sabnzbd queue globally paused?
	PausedAll                  bool          // Paused All actions which causes disk activity
	PauseDuration              time.Duration // Duration sabnzbd will remain paused
	DownloadDirDiskspaceUsed   float64       // Download Directory Used in bytes
	DownloadDirDiskspaceTotal  float64       // Download Directory Total in bytes
	CompletedDirDiskspaceUsed  float64       // Completed Directory Used in bytes
	CompletedDirDiskspaceTotal float64       // Completed Directory Total in bytes
	SpeedLimit                 float64       // The Speed Limit set as a percentage of configured line speed
	SpeedLimitAbs              float64       // The Speed Limit set in B/s
	HaveWarnings               float64       // Number of Warnings present
	Quota                      float64       // Total Quota configured Bytes
	HaveQuota                  bool          // Is a Periodic Quota set for Sabnzbd?
	RemainingQuota             float64       // Quota Remaining Bytes
	CacheArt                   float64       // Number of Articles in Cache
	CacheSize                  float64       // Size of Cache in bytes
	Speed                      float64       // Float String representing bps
	RemainingSize              float64       // Bytes left to download in queue
	Size                       float64       // total bytes represented by queue
	ItemsInQueue               float64       // Total number of items in queue
	Status                     Status        // Status of sabnzbd (1 = Idle, 2 = Paused, 3 = Downloading)
	TimeEstimate               time.Duration // Estimated time remaining to download queue
}

func (q *QueueStats) UnmarshalJSON(data []byte) error {
	var v map[string]map[string]interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	queue := v["queue"]

	q.Version, _ = queue["version"].(string)
	q.Paused, _ = queue["paused"].(bool)
	q.PausedAll, _ = queue["paused_all"].(bool)
	q.HaveQuota, _ = queue["have_quota"].(bool)
	q.ItemsInQueue, _ = queue["noofslots_total"].(float64)
	status, ok := queue["status"].(string)
	if ok {
		q.Status = StatusFromString(status)
	}

	var err error
	q.PauseDuration, err = parseDuration(queue["pause_int"], err)
	downloadDirDiskspaceFree, err := parseFloat(queue["diskspace1"], err)
	completedDirDiskspaceFree, err := parseFloat(queue["diskspace2"], err)
	q.DownloadDirDiskspaceTotal, err = parseFloat(queue["diskspacetotal1"], err)
	q.CompletedDirDiskspaceTotal, err = parseFloat(queue["diskspacetotal2"], err)
	q.SpeedLimit, err = parseSize(queue["speedlimit"], err)
	q.SpeedLimitAbs, err = parseSize(queue["speedlimit_abs"], err)
	q.HaveWarnings, err = parseFloat(queue["have_warnings"], err)
	q.Quota, err = parseSize(queue["quota"], err)
	q.RemainingQuota, err = parseSize(queue["left_quota"], err)
	q.CacheArt, err = parseSize(queue["cache_art"], err)
	q.CacheSize, err = parseSize(queue["cache_size"], err)
	q.Speed, err = parseFloat(queue["kbpersec"], err)
	q.RemainingSize, err = parseFloat(queue["mbleft"], err)
	q.Size, err = parseFloat(queue["mb"], err)
	q.TimeEstimate, err = parseDuration(queue["timeleft"], err)

	if err != nil {
		return fmt.Errorf("Error parsing queue stats: %w", err)
	}

	q.DownloadDirDiskspaceTotal *= GB

	q.CompletedDirDiskspaceTotal *= GB
	q.DownloadDirDiskspaceUsed = q.DownloadDirDiskspaceTotal - (downloadDirDiskspaceFree * GB)
	q.CompletedDirDiskspaceUsed = q.CompletedDirDiskspaceTotal - (completedDirDiskspaceFree * GB)
	q.Speed *= KB
	q.RemainingSize *= MB
	q.Size *= MB

	return nil
}

// latestStat gets the most recent date's value from a map of dates to values
func latestStat(m map[string]int) (string, int) {
	if len(m) == 0 {
		return "", 0
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)
	key := keys[len(keys)-1]

	return key, m[key]
}

// parseFloat is a monad version of strconv.ParseFloat
func parseFloat(s interface{}, prevErr error) (float64, error) {
	if prevErr != nil {
		return 0, prevErr
	}

	if s == nil {
		return 0, nil
	}

	f, ok := s.(string)
	if !ok {
		return 0, fmt.Errorf("Invalid float: %v", s)
	}

	if f == "" {
		return 0, nil
	}

	ret, err := strconv.ParseFloat(f, 64)
	if err != nil {
		return 0, err
	}

	return ret, nil
}

// parseSize is a monad which parses a size string in the format of "123.45 KB" or "123.45"
func parseSize(s interface{}, prevErr error) (float64, error) {
	if prevErr != nil {
		return 0, prevErr
	}

	if s == nil {
		return 0, nil
	}

	sz, ok := s.(string)
	if !ok {
		return 0, fmt.Errorf("Invalid float: %v", s)
	}

	fields := strings.Fields(strings.TrimSpace(sz))
	if len(fields) == 0 {
		return 0, nil
	}

	if len(fields) > 2 {
		return 0, fmt.Errorf("Invalid size: %s", sz)
	}

	ret, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	if len(fields) == 1 {
		return ret, nil
	}

	switch fields[1] {
	case "B":
		return ret, nil
	case "KB", "K":
		return ret * 1024, nil
	case "MB", "M":
		return ret * 1024 * 1024, nil
	case "GB", "G":
		return ret * 1024 * 1024 * 1024, nil
	case "TB", "T":
		return ret * 1024 * 1024 * 1024 * 1024, nil
	case "PB", "P":
		return ret * 1024 * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("Invalid size suffix: %s", sz)
	}
}

// parseDuration is a monad which parses a duration string in the format of "HH:MM:SS" or "MM:SS"
func parseDuration(sd interface{}, prevErr error) (time.Duration, error) {
	if prevErr != nil {
		return 0, prevErr
	}

	if sd == nil {
		return 0, nil
	}

	s, ok := sd.(string)
	if !ok {
		return 0, fmt.Errorf("Invalid float: %v", sd)
	}

	if s == "" {
		return 0, nil
	}

	fields := strings.Split(strings.TrimSpace(s), ":")
	if len(fields) < 1 || len(fields) > 4 {
		return 0, fmt.Errorf("Invalid duration: %s", s)
	}

	intFields := make([]int, len(fields))

	for i, f := range fields {
		var err error
		// Reverse the order of the fields
		intFields[len(intFields)-1-i], err = strconv.Atoi(f)
		if err != nil {
			return 0, fmt.Errorf("Invalid integer in duration: %s: %w", f, err)
		}
	}

	ret := time.Duration(intFields[0]) * time.Second

	fieldCount := len(intFields)
	if fieldCount > 1 {
		ret += time.Duration(intFields[1]) * time.Minute
	}

	if fieldCount > 2 {
		ret += time.Duration(intFields[2]) * time.Hour
	}

	if fieldCount > 3 {
		ret += time.Duration(intFields[3]) * 24 * time.Hour
	}

	return ret, nil
}
