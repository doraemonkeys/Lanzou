package lanzou

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

//从蓝奏云获取指定文件的下载直链
func GetDownloadUrl(homeUrl string, pwd string, filename string) (string, error) {
	content, err := accessHomepage(homeUrl)
	if err != nil {
		return "", err
	}
	urlpath, parame, err := getPostField(content)
	if err != nil {
		return "", err
	}
	if parame.Has("pwd") {
		parame.Set("pwd", pwd)
	}
	postUrl := homeUrl[:strings.LastIndexByte(homeUrl, '/')] + urlpath
	filedata, err := postPwdToGetJsonData(homeUrl, postUrl, parame, filename)
	if err != nil {
		return "", err
	}
	fileUrl, err := findFile(filedata, filename, homeUrl)
	if err != nil {
		return "", err
	}
	fileSrc, err := accessFilePageToGetFileSrc(fileUrl)
	if err != nil {
		return "", err
	}
	filePage2Url := fileUrl[:strings.LastIndexByte(fileUrl, '/')] + fileSrc
	content, err = accessFilePage2(filePage2Url)
	if err != nil {
		return "", err
	}
	parames, urlpath2, err := matchFilePageData(content)
	if err != nil {
		return "", err
	}
	postUrl2 := fileUrl[:strings.LastIndexByte(fileUrl, '/')] + urlpath2
	return getDirectURL(postUrl2, filePage2Url, parames)
}

func postPwdToGetJsonData(homeUrl string, postUrl string, parame url.Values, filename string) (LanzouyPostRes, error) {
	data := parame.Encode()
	request, err := http.NewRequest("POST", postUrl, strings.NewReader(data))
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	request.Header.Set("referer", homeUrl)
	request.Header.Set("content-type", `application/x-www-form-urlencoded`)
	request.Header.Set("accept", `application/json, text/javascript, */*`)
	if err != nil {
		return LanzouyPostRes{}, err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return LanzouyPostRes{}, err
	}
	defer resp.Body.Close()
	bodycontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return LanzouyPostRes{}, err
	}
	var filedata = LanzouyPostRes{}
	err = json.Unmarshal(bodycontent, &filedata)
	if err != nil {
		bodycontent = unicodeToUtf8(bodycontent)
		re := regexp.MustCompile(`"info"[ ]*:"([^"]*)"`)
		info := re.FindStringSubmatch(string(bodycontent))
		if len(info) > 0 {
			return LanzouyPostRes{}, errors.New(info[1])
		}
		return LanzouyPostRes{}, fmt.Errorf("json解析错误:%w", err)
	}
	return filedata, nil
}

//对Unicode进行转码
func unicodeToUtf8(str []byte) []byte {
	reg := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	return reg.ReplaceAllFunc(str, func(s []byte) []byte {
		r, _ := strconv.ParseInt(string(s[2:]), 16, 32)
		return []byte(string(rune(r)))
	})
}

func findFile(filedata LanzouyPostRes, filename string, homeUrl string) (string, error) {
	var fileID string
	ok := false
	for _, v := range filedata.Text {
		if v.NameAll == filename {
			fileID = v.ID
			ok = true
			break
		}
	}
	if !ok {
		return "", errors.New("文件不存在")
	}
	fileUrl := homeUrl[:strings.LastIndexByte(homeUrl, '/')] + "/" + fileID
	return fileUrl, nil
}

func accessFilePageToGetFileSrc(fileUrl string) (string, error) {
	request, err := http.NewRequest("GET", fileUrl, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodycontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	//TODO: find fileSrc
	re := regexp.MustCompile(`src[ ]*=[ ]*"(/fn[^"]{20,})"`)
	match := re.FindSubmatch(bodycontent)
	if match == nil {
		return "", errors.New("未找到文件fileSrc,可能是蓝奏云结构的变化")
	}
	fileSrc := string(match[1])
	return fileSrc, nil
}

func getDirectURL(postUrl string, referer string, parames url.Values) (string, error) {
	data := parames.Encode()
	request, err := http.NewRequest("POST", postUrl, strings.NewReader(data))
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	request.Header.Set("referer", referer)
	request.Header.Set("content-type", `application/x-www-form-urlencoded`)
	request.Header.Set("accept", `application/json, text/javascript, */*`)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodycontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var urlInfo FileDirectUrl
	err = json.Unmarshal(bodycontent, &urlInfo)
	if err != nil {
		return "", fmt.Errorf("json解析错误:%w", err)
	}
	directURL := urlInfo.Dom + "/file/" + urlInfo.URL
	return directURL, nil
}

func matchFilePageData(content string) (url.Values, string, error) {
	re := regexp2.MustCompile(`(?<![/a-z]+)data[ ]*:[ ]*{([^}]+)}`, 0)
	match, err := re.FindStringMatch(content)
	if err != nil {
		return nil, "", err
	}
	if match == nil {
		return nil, "", errors.New("未找到data")
	}
	data := match.GroupByNumber(1).String()
	var parames = url.Values{}
	//匹配data中固定的参数和值(单引号包裹)
	re = regexp2.MustCompile(`'([\w]+)'[ ]*:[ ]*('[^']*'|[0-9]+)`, 0)
	match, err = re.FindStringMatch(data)
	if err != nil {
		return nil, "", err
	}
	if match == nil {
		return nil, "", errors.New("未找到data中的参数")
	}
	for match != nil {
		parames.Add(match.GroupByNumber(1).String(), strings.Trim(match.GroupByNumber(2).String(), "'"))
		match, err = re.FindNextMatch(match)
		if err != nil {
			return nil, "", err
		}
	}
	//匹配data中可变的参数
	//var tempParames = url.Values{}
	//匹配data中固定的参数(单引号包裹)
	re = regexp2.MustCompile(`'([\w]+)'[ ]*:[ ]*([a-zA-Z][\w]+)`, 0)
	match, err = re.FindStringMatch(data)
	if err != nil {
		return nil, "", err
	}
	if match == nil {
		return nil, "", errors.New("未找到data中的参数")
	}

	for match != nil {
		//从html中寻找可变参数的值
		re2 := regexp2.MustCompile(match.GroupByNumber(2).String()+`[ ]*=[ ]*'([^']*)'`, 0)
		match2, err := re2.FindStringMatch(content)
		if err != nil {
			return nil, "", err
		}
		if match2 == nil {
			return nil, "", errors.New("未找到可变参数的值")
		}
		parames.Add(match.GroupByNumber(1).String(), match2.GroupByNumber(1).String())
		match, err = re.FindNextMatch(match)
		if err != nil {
			return nil, "", err
		}
	}
	//find urlpath
	re = regexp2.MustCompile(`url[ ]*:[ ]*'([^']+)'`, 0)
	rematch, err := re.FindStringMatch(content)
	if err != nil {
		return nil, "", err
	}
	if rematch == nil {
		return nil, "", errors.New("未找到urlpath")
	}
	urlpath2 := rematch.GroupByNumber(1).String()
	return parames, urlpath2, nil
}

func accessFilePage2(filePage2Url string) (string, error) {

	request, err := http.NewRequest("GET", filePage2Url, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodycontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodycontent), nil
}

func mathcVariables(content string) (map[string]string, error) {
	re := regexp2.MustCompile(`(?<![\w.])([a-zA-Z0-9_]+)[ ]*=[ ]*[']?([a-zA-Z0-9]+)[']?(?![\w.]+)`, 0)
	rematch, err := re.FindStringMatch(content)
	if err != nil {
		return nil, err
	}
	if rematch == nil {
		return nil, errors.New("地址可能已经失效")
	}
	var variables = make(map[string]string, 10)
	for rematch != nil {
		variables[rematch.GroupByNumber(1).String()] = rematch.GroupByNumber(2).String()
		rematch, err = re.FindNextMatch(rematch)
		if err != nil {
			return nil, err
		}
	}
	return variables, nil
}

func matchPostParame(content string) (parame url.Values, err error) {
	re := regexp2.MustCompile(`'([\w]+)'[ ]*:[ ]*[']?([\w]+)[']?`, 0)
	rematch, err := re.FindStringMatch(content)
	if err != nil {
		return nil, err
	}
	if rematch == nil {
		return nil, errors.New("匹配网页结构错误")
	}
	parame = make(url.Values, 10)
	for rematch != nil {
		parame.Add(rematch.GroupByNumber(1).String(), rematch.GroupByNumber(2).String())
		rematch, err = re.FindNextMatch(rematch)
		if err != nil {
			return nil, err
		}
	}
	return parame, nil
}

func getPostField(content string) (urlpath string, parame url.Values, err error) {
	variables, err := mathcVariables(content)
	if err != nil {
		return "", nil, err
	}
	parame, err = matchPostParame(content)
	if err != nil {
		return "", nil, err
	}
	for k, v := range variables {
		for k1, v1 := range parame {
			if k == v1[0] {
				parame.Set(k1, v)
				break
			}
		}
	}
	re := regexp2.MustCompile(`url[ ]*:[ ]*'([^']+)'`, 0)
	rematch, err := re.FindStringMatch(content)
	if err != nil {
		return "", nil, err
	}
	if rematch == nil {
		return "", nil, errors.New("匹配网页结构错误")
	}
	urlpath = rematch.GroupByNumber(1).String()
	//fmt.Println(urlpath, parame.Encode())
	return urlpath, parame, nil
}

func accessHomepage(url string) (string, error) {
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("访问url失败,err:%w", err)
	}
	defer resp.Body.Close()
	bodycontent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取网页内容失败,err:%w", err)
	}
	re := regexp2.MustCompile(`<script type="[a-z/_A-Z0-9]+[^" ]+">([\s\S]+?)dataType`, 0)
	rematch, err := re.FindStringMatch(string(bodycontent))
	if err != nil {
		return "", err
	}
	if rematch == nil {
		return "", errors.New("匹配网页结构错误")
	}
	return rematch.Capture.String(), nil
}
