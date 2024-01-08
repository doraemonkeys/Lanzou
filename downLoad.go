package lanzou

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// Download可用于下载未重定向的原始蓝奏云直链,filename为文件存储的路径(可省略)+文件名
func Download(url string, filename string) error {
	request, err := http.NewRequest("GET", url, nil)
	request.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.102 Safari/537.36 Edg/104.0.1293.70")
	request.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	request.Header.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("访问url失败,err:%w", err)
	}
	defer resp.Body.Close()
	// 创建一个文件用于保存
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
