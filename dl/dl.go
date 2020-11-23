package dl

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/PuerkitoBio/goquery"
)

//GetVipAudioInfo 获取VIP音频信息
func GetVipAudioInfo(trackId int, cookie string) (ai AudioItem, err error) {
	ts := time.Now().Unix()
	url := fmt.Sprintf(
		"https://mpay.ximalaya.com/mobile/track/pay/%d/%d?device=pc&isBackend=true&_=%d",
		trackId, ts, ts)

	resp, err := httpGetByCookie(url, cookie)
	if err != nil {
		return ai, fmt.Errorf("获取音频信息失败: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ai, err
	}

	var vipAudioInfo VipAudioInfo
	err = jsoniter.Unmarshal(body, &vipAudioInfo)
	if err != nil {
		return ai, fmt.Errorf("解析Json失败: %v", err)
	}

	if vipAudioInfo.Ret != 0 {
		return ai, fmt.Errorf("获取音频信息失败: %v", jsoniter.Get(body, "msg").ToString())
	}

	fileName := DecryptFileName(vipAudioInfo.Seed, vipAudioInfo.FileID)
	sign, _, token, timestamp := DecryptUrlParams(vipAudioInfo.Ep)

	args := fmt.Sprintf("?sign=%s&buy_key=%s&token=%d&timestamp=%d&duration=%d",
		sign, vipAudioInfo.BuyKey, token, timestamp, vipAudioInfo.Duration)

	ai.TrackId = trackId
	ai.Title = vipAudioInfo.Title
	ai.URL = vipAudioInfo.Domain + "/download/" + vipAudioInfo.APIVersion + fileName + args
	return ai, nil
}

//GetAudioInfo 获取音频信息
func GetAudioInfo(albumID, page, pageSize int) (audioList []AudioItem, err error) {
	format := fmt.Sprintf("https://m.ximalaya.com/m-revision/common/album/queryAlbumTrackRecordsByPage?albumId=%d&page=%d&pageSize=%d&asc=true", albumID, page, pageSize)
	log.Printf("Get: \u001B[1;33m%v\u001B[0m", format)

	resp, err := client.Get(format)
	if err != nil {
		return nil, fmt.Errorf("http get %v fail:%v", format, err.Error())
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	list := jsoniter.Get(body, "data").Get("trackDetailInfos")
	for i2 := 0; i2 < list.Size(); i2++ {
		v := list.Get(i2).Get("trackInfo")
		audioList = append(audioList, AudioItem{TrackId: v.Get("id").ToInt(), URL: v.Get("playPath").ToString(), Title: v.Get("title").ToString()})
	}

	return audioList, nil
}

//GetAudioInfoList 获取音频信息列表
func GetAudioInfoList(albumID, audioCount int) (audioList []AudioItem) {
	number := audioCount / 100
	j := audioCount % 100
	if j > 0 {
		number++
	}

	for i := 1; i < number+1; i++ {
		list, err := GetAudioInfo(albumID, i, 100)
		if err != nil {
			log.Println(err)
			return audioList
		}
		audioList = append(audioList, list...)
	}

	return audioList
}

//GetAlbumInfo 获取专辑信息
func GetAlbumInfo(url string) (title string, audioCount, pageCount int, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", 0, 0, fmt.Errorf("获取专辑信息失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", -1, -1, fmt.Errorf("获取专辑信息失败: StatusCode != 200")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", -1, -1, fmt.Errorf("new goquery document fail:" + err.Error())
	}

	//标题
	title = doc.Find("h1.title").Text()

	//音频数量
	r, _ := regexp.Compile("\\d+\\.?\\d*")
	num := r.FindString(doc.Find("div.head").Text())
	audioCount, _ = strconv.Atoi(num)

	//页面数量
	list := doc.Find("ul.pagination-page").Children()
	size := list.Size()
	if size > 6 { //超过5页
		i, _ := strconv.Atoi(list.Eq(list.Size() - 2).Text())
		pageCount = i
	} else if size == 0 { //仅一页
		pageCount = 1
	} else { //大于0页 && 小于等于5页
		pageCount = size - 1 //1为下一页按钮
	}
	return
}

//GetAlbumInfoByMobileAPI 使用Mobile API获取专辑信息
func GetAlbumInfoByMobileAPI(albumID int) error {
	url := fmt.Sprintf(
		"https://mobile.ximalaya.com/mobile/v1/album/track/ts-%d?ac=4G&albumID=%d&device=android&isAsc=true&pageId=%d&pageSize=200&pre_page=0&source=0&supportWebp=false",
		time.Now().Unix(), albumID, 1)
	resp, err := httpGet(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Println(string(data))
	return nil
}

//使用 专辑ID & 页面ID 获取音频信息列表
func GetAudioListByPageId(albumID, pageID int) (err error, ai *AudioInfoMobile) {
	url := fmt.Sprintf(
		"https://mobile.ximalaya.com/mobile/v1/album/track/ts-%d?ac=4G&albumID=%d&device=android&isAsc=true&pageID=%d&pageSize=200&pre_page=0&source=0&supportWebp=false",
		time.Now().Unix(), albumID, pageID)
	resp, err := httpGet(url)
	if err != nil {
		return err, nil
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err, nil
	}

	ai = &AudioInfoMobile{}
	err = jsoniter.Unmarshal(data, ai)
	if err != nil {
		return fmt.Errorf("解析Json失败: %v", err), ai
	}

	if ai.Ret != 0 {
		return fmt.Errorf("服务器返回异常: %s", ai.Msg), ai
	}

	return nil, ai
}

//GetUserInfo 使用Cookie获取用户信息
func GetUserInfo(cookie string) (*UserInfo, error) {
	resp, err := httpGetByCookie("https://www.ximalaya.com/revision/main/getCurrentUser", cookie)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ui := &UserInfo{}
	err = jsoniter.Unmarshal(data, ui)
	if err != nil {
		return nil, err
	}

	return ui, nil
}

var client http.Client

func httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("user-agent", "ting_6.3.60(sdk,Android16)")

	return client.Do(req)
}

func httpGetByCookie(url, cookie string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Cookie", cookie)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("user-agent", "ting_6.3.60(sdk,Android16)")

	return client.Do(req)
}
