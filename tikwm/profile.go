package tikwm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
)

type UserPostsTiktokInfo struct {
	Id          string `json:"video_id"`
	SourceURL   string `json:"play"`
	HDSourceURL string `json:"wmplay"`
	CreateTime  int64  `json:"create_time"`
	Author      struct {
		Username string `json:"unique_id"`
	} `json:"author"`
}

func (upti UserPostsTiktokInfo) ToTiktokInfo() *TiktokInfo {
	return &TiktokInfo{
		Id:          upti.Id,
		SourceURL:   upti.SourceURL,
		HDSourceURL: upti.HDSourceURL,
		CreateTime:  upti.CreateTime,
		Author:      upti.Author,
	}
}

type UserPostsInfo struct {
	Videos  []UserPostsTiktokInfo `json:"videos"`
	Cursor  string                `json:"cursor"`
	HasMore bool                  `json:"hasMore"`
}

type ResponseUserPostsInfo struct {
	Code          int           `json:"code"`
	Message       string        `json:"msg"`
	ProcessedTime float64       `json:"processed_time"`
	Data          UserPostsInfo `json:"data,omitempty"`
}

func GetUserPostsInfo(username string, cursor ...string) (*UserPostsInfo, error) {
	r, err := SyncedRequest("https://www.tikwm.com/api/user/posts/",
		url.Values{"unique_id": {username}, "count": {"34"}, "cursor": {valueOrDefault(cursor, "0")}})
	if err != nil {
		return nil, err
	}
	buffer, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	var resp ResponseUserPostsInfo
	err = json.Unmarshal(buffer, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, errors.New(resp.Message)
	}

	return &resp.Data, nil
}

type ProfileDownloader struct {
	Username   string
	Directory  string
	FfmpegPath string
	CheckAll   bool
	Convert    bool
	StopCause  func(video *UserPostsTiktokInfo) bool
}

func NewProfileDownloader(username string) *ProfileDownloader {
	username = strings.ToLower(username)
	downloader := ProfileDownloader{
		Username:   username,
		Directory:  username,
		FfmpegPath: DefaultFfmpegPath,
		CheckAll:   false,
		Convert:    false,
	}
	downloader.StopCause = func(video *UserPostsTiktokInfo) bool {
		return downloader.IsDownloaded(video.Id)
	}
	return &downloader
}

// IsDownloaded checking all files in case of user changing their username (i.e., shrimpydimpy -> kaycoree)
func (downloader *ProfileDownloader) IsDownloaded(id string) bool {
	files, err := os.ReadDir(downloader.Directory)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), id) {
			return true
		}
	}
	return false
}

// DownloadIteratively returns string -- filename, an error otherwise
func (downloader *ProfileDownloader) DownloadIteratively() <-chan interface{} {
	returnChannel := make(chan interface{})
	go func() {
		defer func() {
			close(returnChannel)
		}()
		if _, err := os.Stat(downloader.Directory); os.IsNotExist(err) {
			err := os.Mkdir(downloader.Directory, 0777)
			if err != nil {
				returnChannel <- err
				return
			}
		}

		cursor := "0"
		for {
			postsInfo, err := GetUserPostsInfo(downloader.Username, cursor)
			if err != nil {
				returnChannel <- err
				return
			}

			for _, video := range postsInfo.Videos {
				if !downloader.CheckAll && downloader.StopCause(&video) {
					return
				}

				filename, err := DownloadTiktok(fmt.Sprintf("https://www.tiktok.com/@%s/%s", video.Author.Username, video.Id),
					path.Join(downloader.Directory, GenerateFilename(video.ToTiktokInfo())))
				if err != nil {
					returnChannel <- err
					return
				}

				if downloader.Convert {
					h264, err := ConvertToH264(filename, downloader.FfmpegPath)
					if err != nil { // not that big of a deal
						returnChannel <- err
					} else {
						err = os.Rename(h264, filename)
						if err != nil {
							returnChannel <- err
							return
						}
					}
				}
				returnChannel <- filename
			}

			if postsInfo.HasMore {
				cursor = postsInfo.Cursor
			} else {
				return
			}
		}
	}()

	return returnChannel
}

func (downloader *ProfileDownloader) Download() (filenames []string, err error) {
	for filenameOrError := range downloader.DownloadIteratively() {
		switch filenameOrError.(type) {
		case string:
			filenames = append(filenames, filenameOrError.(string))
		case error:
			return filenames, filenameOrError.(error)
		}
	}
	return
}
