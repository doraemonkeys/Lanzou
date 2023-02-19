package lanzou

//可能会因为蓝奏云的变化而发生错误

type FileDirectUrl struct {
	//Zt  int    `json:"zt"`
	Dom string `json:"dom"`
	URL string `json:"url"`
	//Inf int    `json:"inf"`
}

//单文件post返回的json
type LanzouyPostRes2 struct {
	//Zt int `json:"zt"`
	Dom string `json:"dom"`
	URL string `json:"url"`
	Inf string `json:"inf"`
}

//文件夹post返回的json
type LanzouyPostRes struct {
	//Zt   int    `json:"zt"`
	Info string `json:"info"`
	Text []Text `json:"text"`
}
type Text struct {
	//Icon    string `json:"icon"`
	//T       int    `json:"t"`
	ID      string `json:"id"`
	NameAll string `json:"name_all"`
	//Size    string `json:"size"`
	//Time string `json:"time"`
	//Duan    string `json:"duan"`
	//PIco    int    `json:"p_ico"`
}

type LFile struct {
	DirectUrl string
	//Size        int64
	Name string
	//UploadTime  string
	//Description string
	//FileType    string
}
