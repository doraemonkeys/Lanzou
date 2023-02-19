# lanzou

提取蓝奏云下载直链



## QuickStart

`go get -u github.com/Doraemonkeys/lanzou`



```go
package main

import (
	"fmt"
	"github.com/Doraemonkeys/lanzou"
)

func main() {
	file, err := lanzou.GetDownloadUrl("https://XXXXX", "pwd", "desiredFileName")
	if err != nil {
		fmt.Println(err)
		return
	}
    fmt.Println(file.Name)
	fmt.Println(file.DirectUrl)
	lanzou.Download(file.DirectUrl, "./"+file.Name)
}
```



