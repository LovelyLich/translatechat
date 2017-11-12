package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

type ChatMsgJson struct {
	Catalog  string
	Time     string
	FromUser string
	ToUser   string

	FromLang string
	ToLang   string

	FromText  string
	FromAudio string // base64编码的amr文件

	ToText     string
	ToAudioUrl string // 目标amr文件下载地址
}

func getChatMsg(isText bool) (*ChatMsgJson, error) {
	var msg *ChatMsgJson
	if !isText {
		amrFile := "../tcpBoltDB/test.amr"
		content, err := ioutil.ReadFile(amrFile)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		str := base64.StdEncoding.EncodeToString(content)
		msg = &ChatMsgJson{
			Catalog:    "audio",
			Time:       strconv.FormatInt(time.Now().Unix(), 10),
			FromLang:   "zh",
			ToLang:     "en",
			FromText:   "",
			FromAudio:  str,
			ToText:     "",
			ToAudioUrl: "",
		}
	} else {
		msg = &ChatMsgJson{
			Catalog:    "text",
			Time:       strconv.FormatInt(time.Now().Unix(), 10),
			FromLang:   "zh",
			ToLang:     "en",
			FromText:   "你好,很高兴见到你",
			FromAudio:  "",
			ToText:     "",
			ToAudioUrl: "",
		}
	}
	return msg, nil
}

func main() {
	isText := flag.Bool("isText", false, "是文本翻译")
	flag.Parse()
	//构造Message, 并Post
	url := "http://27.155.100.158:3389/translate"
	msg, err := getChatMsg(*isText)
	if err != nil {
		fmt.Println(err)
		return
	}

	var buff []byte
	buff, err = json.Marshal(*msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	var resp *http.Response
	resp, err = http.Post(url, "application/json", bytes.NewReader(buff))
	if err != nil {
		fmt.Println(err)
		return
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	fmt.Printf("response: %#v\n", string(body))
}
