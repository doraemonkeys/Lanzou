package lanzou

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

//filename为文件存储的路径(可省略)和文件名,记得校验md5
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
	//io.Copy() 方法将副本从 src 复制到 dst ，直到 src 达到文件末尾 ( EOF ) 或发生错误，
	//然后返回复制的字节数和复制时遇到的第一个错误(如果有)。
	//将响应流和文件流对接起来
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
