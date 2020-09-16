package dl

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type AlbumDetail struct {
	classify   string
	albumId    int
	trackId    int
	rawurl     string
	parsedUrl  *url.URL
	title      string
	audioCount int
	pageCount  int
	output     string
	audioList  []AudioItem
}

var (
	errFormat     = "\u001B[91m[%v]\u001B[0m 下载失败(第\u001B[32m%v\u001B[0m次重试) \u001B[36m%s\u001B[0m: %s"
	maxRetryCount = 3
	defaultDir    = "./downloads"
)

func NewAlbumDetail(rawurl string) (*AlbumDetail, error) {
	var classify string
	var albumId int
	var trackId int

	parsedUrl, err := url.Parse(rawurl)
	if err != nil {
		return nil, errors.WithMessage(err, "Parse url failed")
	}

	splitStr := strings.Split(parsedUrl.Path, "/")
	if len(splitStr) < 3 {
		return nil, errors.New("Read url failed ")
	}

	classify = splitStr[1]
	albumId, _ = strconv.Atoi(splitStr[2])

	if len(splitStr) > 3 {
		trackId, _ = strconv.Atoi(splitStr[3])
	}

	detail := AlbumDetail{
		classify:  classify,
		albumId:   albumId,
		trackId:   trackId,
		rawurl:    rawurl,
		parsedUrl: parsedUrl,
		output:    defaultDir,
	}
	return &detail, nil
}

func (a *AlbumDetail) Fetch() error {
	title, audioCount, pageCount, err := GetAlbumInfo(a.albumUrl())
	if err != nil {
		return errors.WithMessage(err, "Get album info failed")
	}
	splitStr := strings.Split(a.parsedUrl.Path, "/")
	albumId, err := strconv.Atoi(splitStr[2])
	var trackId int
	if len(splitStr) > 3 {
		trackId, _ = strconv.Atoi(splitStr[3])
	}
	if err != nil {
		return errors.WithMessage(err, "Parse album id failed")
	}
	audioList := GetAudioInfoList(albumId, audioCount)

	a.albumId = albumId
	a.trackId = trackId
	a.title = title
	a.audioCount = audioCount
	a.pageCount = pageCount
	a.audioList = audioList
	return nil
}

func (a *AlbumDetail) SetOutput(output string) {
	a.output = output
}

func (a AlbumDetail) Display() {
	fmt.Println("Album Info:")
	fmt.Println("Id:", a.albumId)
	fmt.Println("TrackId:", a.trackId)
	fmt.Println("Classify", a.classify)
	fmt.Println("Title:", a.title)
	fmt.Println("Amount:", a.audioCount)
	fmt.Println("Audio List:")
	for index, audio := range a.audioList {
		if index > 2 {
			fmt.Println("...")
			fmt.Printf("Another %d audios skip.\n", len(a.audioList)-2)
			break
		}
		fmt.Println(audio)
	}
}

func (a AlbumDetail) DownLoad() error {
	if a.trackId == 0 {
		return a.DownloadAll()
	} else {
		return a.DownloadTrack()
	}
}

func (a AlbumDetail) DownloadTrack() error {
	target := AudioItem{}
	for _, audio := range a.audioList {
		if audio.TrackId == a.trackId {
			target = audio
			break
		}
	}
	if target.TrackId == 0 {
		return errors.Errorf("Download track with id %d failed", a.trackId)
	}
	fileName := a.combineFileName(&target)
	if err := a.DownloadFile(target.URL, fileName); err != nil {
		return errors.WithMessage(err, "Download track failed")
	}

	return nil
}

func (a AlbumDetail) DownloadAll() error {
	for _, audio := range a.audioList {
		fileName := a.combineFileName(&audio)

		if err := a.DownloadFile(audio.URL, fileName); err != nil {
			return errors.WithMessage(err, "Download list failed")
		}
	}
	fmt.Println("Download finished.")
	return nil
}

func (a AlbumDetail) DownloadFile(rawurl, fileName string) error {
	filePath := path.Join(a.output, fileName)

	tried := 0
	var resp *http.Response
	var err error
	for tried < maxRetryCount {
		resp, err = a.httpGet(rawurl)
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			break
		}
		tried++
		resp.Body.Close()
	}
	if tried == maxRetryCount {
		return errors.New("Response failed after max tries")
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0777)
	if err != nil && !os.IsExist(err) {
		return errors.WithMessage(err, "Make dir failed")
	}

	file, err := os.Create(filePath)
	if err != nil {
		return errors.WithMessage(err, "Create file failed")
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return errors.WithMessage(err, "Save file failed")
	}
	fmt.Println("Downloaded:", filePath)
	return nil
}

func (a AlbumDetail) albumUrl() string {
	return fmt.Sprintf("%s://%s/%s/%d", a.parsedUrl.Scheme, a.parsedUrl.Host, a.classify, a.albumId)
}

func (a AlbumDetail) combineFileName(ai *AudioItem) string {
	u, _ := url.Parse(ai.URL)
	splits := strings.Split(u.Path, ".")
	postfix := splits[len(splits)-1]
	fileName := fmt.Sprintf("%s.%s", ai.Title, postfix)
	return fileName
}

func (a AlbumDetail) httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4170.0 Safari/537.36 Edg/85.0.552.1")

	return client.Do(req)
}

func (a AlbumDetail) ansiRed(v interface{}) string {
	return fmt.Sprintf("\u001B[91m%v\u001B[0m", v)
}
