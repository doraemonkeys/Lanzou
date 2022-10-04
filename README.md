# lanzou
## Overview

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
	durl, err := lanzou.GetDownloadUrl("https://XXXXX", "pwd", "filename")
	if err != nil {
		fmt.Println(err)
		return
	}
	lanzou.Download(durl, "./filename")
}
```



