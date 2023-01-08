package tikwm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
)

// TiktokInfo there are much more fields, tho I omitted unnecessary ones
type TiktokInfo struct {
	Id          string `json:"id"`
	SourceURL   string `json:"play,omitempty"`
	HDSourceURL string `json:"hdplay,omitempty"`
	CreateTime  int64  `json:"create_time"`
	Author      struct {
		Username string `json:"unique_id"`
	} `json:"author"`
}

type ResponseTiktokInfo struct {
	Code          int        `json:"code"`
	Message       string     `json:"msg"`
	ProcessedTime float64    `json:"processed_time"`
	Data          TiktokInfo `json:"data,omitempty"`
}

func GetTiktokInfo(link string) (*TiktokInfo, error) {
	r, err := SyncedRequest("https://www.tikwm.com/api/", url.Values{"url": {link}, "hd": {"1"}})
	if err != nil {
		return nil, err
	}
	buffer, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var response ResponseTiktokInfo
	err = json.Unmarshal(buffer, &response)
	if err != nil {
		return nil, err
	}
	if response.Code != 0 {
		return nil, errors.New(response.Message)
	}

	return &response.Data, nil
}

func DownloadTiktok(link string, destination ...string) (string, error) {
	info, err := GetTiktokInfo(link)
	if err != nil {
		return "", err
	}

	var sourceURL string
	if info.HDSourceURL != "" {
		sourceURL = info.HDSourceURL
	} else if info.SourceURL != "" {
		log.Printf("tikwm couldn't find HD version for %s, downloading how it is...", link)
		sourceURL = info.SourceURL
	} else {
		return "", errors.New(fmt.Sprintf("no download links found :c for %s", link))
	}

	filename := valueOrDefault(destination, GenerateFilename(info))
	if stat, err := os.Stat(filename); err == nil {
		if stat.IsDir() {
			filename = path.Join(filename, GenerateFilename(info))
		}
	}
	err = Wget(sourceURL, filename)
	if err != nil {
		return "", err
	}

	return filename, nil
}
