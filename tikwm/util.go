package tikwm

import (
	"errors"
	"fmt"
	"github.com/cavaliergopher/grab/v3"
	"net/http"
	"net/url"
	"os/exec"
	"sync"
	"time"
)

const DefaultFfmpegPath = "ffmpeg"

var (
	Timeout         = 11 * time.Second
	LastRequestTime = time.Time{}
	requestMutex    = sync.Mutex{}
)

func SyncedRequest(url string, payload url.Values) (resp *http.Response, err error) {
	requestMutex.Lock()
	defer requestMutex.Unlock()
	time.Sleep(Timeout - time.Since(LastRequestTime))
	LastRequestTime = time.Now()

	return http.PostForm(url, payload)
}

func GenerateFilename(info *TiktokInfo) string {
	return fmt.Sprintf(
		"%s_%s_%s.mp4",
		info.Author.Username,
		time.Unix(info.CreateTime, 0).Format("2006-01-02"),
		info.Id,
	)
}

func Wget(url string, filename string) error {
	_, err := grab.Get(filename, url)
	return err
}

func ConvertToH264(filename string, ffmpegPath ...string) (string, error) {
	var ffmpeg = valueOrDefault(ffmpegPath, DefaultFfmpegPath)

	h264Filename := fmt.Sprintf("%s.h264.mp4", filename)
	output, err := exec.Command(ffmpeg, "-i", filename, "-vcodec", "libx264", "-acodec", "aac", "-y", "-preset", "fast", h264Filename).Output()
	if err != nil {
		return "", errors.New(fmt.Sprintf("while converting %s, an error occured:\n%s\n%s", filename, err.Error(), string(output)))
	}

	return h264Filename, nil
}

func valueOrDefault(Optional []string, Default string) string {
	if len(Optional) == 0 {
		return Default
	} else {
		return Optional[0]
	}
}
