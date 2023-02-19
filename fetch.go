package lanzou

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
)

// 从蓝奏云获取指定文件的下载直链,若链接为文件夹,则filename必填，否则不知道下载哪个文件
func GetDownloadUrl(homeUrl string, pwd string, wantFilename string) (file LFile, err error) {
	content, fileName_web, err := accessHomepage(homeUrl)
	if err != nil {
		return LFile{}, err
	}
	var (
		IsSingleFile             bool = false
		IsSingleFileDownLoadPage bool = false
	)
	if fileName_web != "" {
		IsSingleFileDownLoadPage = true
	}
	var (
		filedata_folder lanzouyPostRes
		filedata_file   lanzouyPostRes2
	)

	if !IsSingleFileDownLoadPage {
		urlpath, parame, _, err := getPostField(content)
		if err != nil {
			return LFile{}, err
		}
		if parame.Has("pwd") {
			parame.Set("pwd", pwd)
			IsSingleFile = false
		} else if parame.Has("p") {
			parame.Set("p", pwd)
			IsSingleFile = true
		}
		postUrl := homeUrl[:strings.LastIndexByte(homeUrl, '/')] + urlpath
		filedata_folder, filedata_file, err = postPwdToGetJsonData(homeUrl, postUrl, parame, wantFilename, IsSingleFile)
		if err != nil {
			return LFile{}, err
		}
	}

	endUrl := ""
	if IsSingleFile {
		endUrl = filedata_file.Dom + "/file/" + filedata_file.URL
		fileName_web = filedata_file.Inf
	} else {
		var fileUrl string
		if IsSingleFileDownLoadPage {
			fileUrl = homeUrl
		} else {
			fileUrl, err = findFile(filedata_folder, wantFilename, homeUrl)
			if err != nil {
				return LFile{}, err
			}
			fileName_web = wantFilename
		}
		fileSrc, err := accessFilePageToGetFileSrc(fileUrl)
		if err != nil {
			return LFile{}, err
		}
		filePage2Url := fileUrl[:strings.LastIndexByte(fileUrl, '/')] + fileSrc
		content, err = accessFilePage2(filePage2Url)
		if err != nil {
			return LFile{}, err
		}
		parames, urlpath2, err := matchFilePageData(content)
		if err != nil {
			return LFile{}, err
		}
		postUrl2 := fileUrl[:strings.LastIndexByte(fileUrl, '/')] + urlpath2
		endUrl, err = getDirectURL(postUrl2, filePage2Url, parames)
		if err != nil {
			return LFile{}, err
		}
	}
	//尝试获取重定向后的url
	endUrl2, err := getRedirectUrl(endUrl)
	if err != nil {
		//return endUrl, fileName_web, nil
		return LFile{endUrl, fileName_web}, nil
	}
	return LFile{endUrl2, fileName_web}, nil
}

// 获取重定向后的url
func getRedirectUrl(url string) (string, error) {
	request, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.102 Safari/537.36 Edg/104.0.1293.70")
	request.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	request.Header.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("获取重定向后的url失败,err:%w", err)
	}
	defer resp.Body.Close()
	return resp.Request.URL.String(), nil
}

func postPwdToGetJsonData(homeUrl string, postUrl string, parame url.Values, filename string, IsSingleFile bool) (lanzouyPostRes, lanzouyPostRes2, error) {
	data := parame.Encode()
	request, err := http.NewRequest("POST", postUrl, strings.NewReader(data))
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	request.Header.Set("referer", homeUrl)
	request.Header.Set("content-type", `application/x-www-form-urlencoded`)
	request.Header.Set("accept", `application/json, text/javascript, */*`)
	if err != nil {
		return lanzouyPostRes{}, lanzouyPostRes2{}, err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return lanzouyPostRes{}, lanzouyPostRes2{}, err
	}
	defer resp.Body.Close()
	bodycontent, err := io.ReadAll(resp.Body)
	if err != nil {
		return lanzouyPostRes{}, lanzouyPostRes2{}, err
	}
	if IsSingleFile {
		var filedata = lanzouyPostRes2{}
		err = json.Unmarshal(bodycontent, &filedata)
		if err != nil {
			bodycontent = unicodeToUtf8(bodycontent)
			re := regexp.MustCompile(`"inf"[ ]*:"([^"]*)"`)
			info := re.FindStringSubmatch(string(bodycontent))
			if len(info) > 0 {
				return lanzouyPostRes{}, lanzouyPostRes2{}, errors.New(info[1])
			}
			return lanzouyPostRes{}, lanzouyPostRes2{}, fmt.Errorf("json解析错误:%w", err)
		}
		return lanzouyPostRes{}, filedata, nil
	}
	var filedata = lanzouyPostRes{}
	err = json.Unmarshal(bodycontent, &filedata)
	if err != nil {
		bodycontent = unicodeToUtf8(bodycontent)
		re := regexp.MustCompile(`"info"[ ]*:"([^"]*)"`)
		info := re.FindStringSubmatch(string(bodycontent))
		if len(info) > 0 {
			return lanzouyPostRes{}, lanzouyPostRes2{}, errors.New(info[1])
		}
		return lanzouyPostRes{}, lanzouyPostRes2{}, fmt.Errorf("json解析错误:%w", err)
	}
	return filedata, lanzouyPostRes2{}, nil
}

// 对Unicode进行转码
func unicodeToUtf8(str []byte) []byte {
	reg := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	return reg.ReplaceAllFunc(str, func(s []byte) []byte {
		r, _ := strconv.ParseInt(string(s[2:]), 16, 32)
		return []byte(string(rune(r)))
	})
}

func findFile(filedata lanzouyPostRes, filename string, homeUrl string) (string, error) {
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
	bodycontent, err := io.ReadAll(resp.Body)
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
	bodycontent, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var urlInfo fileDirectUrl
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
		re2 := regexp2.MustCompile(`(?<!\w)`+match.GroupByNumber(2).String()+`[ ]*=[ ]*'([^']*)'`, 0)
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
	bodycontent, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodycontent), nil
}

func mathcVariables(content string) (map[string]string, error) {
	re := regexp2.MustCompile(`(?<![\w.])([a-zA-Z0-9_]+)[ ]*=[ ]*[']?([a-zA-Z0-9_]+)[']?(?![\w.]+)`, 0)
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

func getPostField(content string) (urlpath string, parame url.Values, IsSingleFile bool, err error) {
	IsSingleFile = false
	urlpath, parame, err1 := getPostField1(content)
	if err1 != nil {
		var err2 error
		urlpath, parame, err2 = getPostField2(content)
		if err != nil {
			return "", nil, false, errors.Join(err1, err2)
		}
		IsSingleFile = true
	}
	return urlpath, parame, IsSingleFile, nil
}

func getPostField1(content string) (urlpath string, parame url.Values, err error) {
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

// 情况2，第二种网页结构(单文件)
func getPostField2(content string) (urlpath string, parame url.Values, err error) {
	re, err := regexp2.Compile(`(?<![//a-z ]+)data[\s]*:[\s]*['"]([a-zA-Z0-9_&=]+)['"]\+pwd`, 0)
	if err != nil {
		return "", nil, err
	}
	rematch, err := re.FindStringMatch(content)
	if err != nil {
		return "", nil, err
	}
	if rematch == nil {
		return "", nil, errors.New("匹配网页结构错误")
	}
	//解析query字符串 为url.Values
	parame, err = url.ParseQuery(rematch.GroupByNumber(1).String())
	if err != nil {
		return "", nil, err
	}
	re = regexp2.MustCompile(`url[ ]*:[ ]*'([^']+)'`, 0)
	rematch, err = re.FindStringMatch(content)
	if err != nil {
		return "", nil, err
	}
	if rematch == nil {
		return "", nil, errors.New("匹配网页结构错误")
	}
	urlpath = rematch.GroupByNumber(1).String()
	//fmt.Println(urlpath, parame.Encode()) //
	return urlpath, parame, nil
}

// 第二个string是文件名,如果不为空,则此链接是单文件的下载页
func accessHomepage(url string) (string, string, error) {
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.110 Safari/537.36")
	if err != nil {
		return "", "", fmt.Errorf("构造请求失败,err:%w", err)
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", "", fmt.Errorf("访问url失败,err:%w", err)
	}
	defer resp.Body.Close()
	bodycontent, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取网页内容失败,err:%w", err)
	}

	var filenameRetrieved string
	//获取文件或文件夹名称
	re := regexp2.MustCompile(`(?<![// ])<title>(.*)</title>`, 0)
	rematch, err := re.FindStringMatch(string(bodycontent))
	if err == nil && rematch != nil {
		if rematch.GroupByNumber(1).String() == "" {
			return "", "", errors.New("来晚啦...文件取消分享了")
		}
		filenameRetrieved = rematch.GroupByNumber(1).String()
	}

	re = regexp2.MustCompile(`src[ ]*=[ ]*"(/fn[^"]{20,})`, 0)
	rematch, err = re.FindStringMatch(string(bodycontent))
	if err == nil && rematch != nil {
		//此链接是单文件的下载页
		return "", filenameRetrieved, nil
	}

	// 此链接是文件夹，或单文件但有密码
	re = regexp2.MustCompile(`<script type="[a-z/_A-Z0-9]+[^" ]+">([\s\S]+?)dataType`, 0)
	rematch, err = re.FindStringMatch(string(bodycontent))
	if err != nil {
		return "", "", err
	}
	if rematch == nil {
		return "", "", errors.New("匹配网页结构错误")
	}
	return rematch.Capture.String(), "", nil
}
