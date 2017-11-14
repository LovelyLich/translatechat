package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/VolantMQ/volantmq/packet"
	"github.com/VolantMQ/volantmq/topics/types"
	nsq "github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

var (
	logger *zap.Logger
	token  string
)

//{"from":"zh","to":"en","trans_result":[{"src":"\u767e\u5ea6 \u4f60\u597d","dst":"Hello, Baidu"}]}
type TransResp struct {
	From        string `json:"from"`
	To          string `json:"to"`
	TransResult []struct {
		Src string `json:"src"`
		Dst string `json:"dst"`
	} `json:"trans_result"`
}
type TokenResult struct {
	AccessToken string `json:"access_token"`
}

type Audio2Text struct {
	Format  string `json:"format"`
	Rate    int    `json:"rate"`
	Channel int    `json:"channel"`
	Cuid    string `json:"cuid"`
	Token   string `json:"token"`
	Speech  string `json:"speech"`
	Len     int    `json:"len"`
}
type Audio2TextResult struct {
	ErrNo  int      `json:"err_no"`
	ErrMsg string   `json:"err_msg"`
	Result []string `json:"result"`
}

type ChatMsgJson struct {
	Catalog  string
	Time     string
	FromUser string
	ToUser   string

	FromLang string
	ToLang   string

	FromText  string
	FromAudio string // base64编码的Mp3文件

	ToText     string
	ToAudioUrl string // 目标mp3文件下载地址
}

func translateText(FromLang, ToLang, FromText string) (string, error) {
	//文本翻译,翻译源为FromText
	appId := "20170714000064493"
	SecretKey := "1F0H8Oh1YZX2U9pqdIMP"

	queryText := FromText
	queryTextEncoded := url.QueryEscape(queryText)
	srcLang := FromLang
	dstLang := ToLang
	salt := strconv.FormatInt(time.Now().Unix(), 10)
	signStr := appId + queryText + salt + SecretKey
	sign := fmt.Sprintf("%x", md5.Sum([]byte(signStr)))
	translateUrl := "http://api.fanyi.baidu.com/api/trans/vip/translate?q=" + queryTextEncoded + "&from=" + srcLang + "&to=" + dstLang + "&appid=" + appId + "&salt=" + salt + "&sign=" + sign

	fmt.Printf("signStr=[%s], translateUrl=[%s]\n", signStr, translateUrl)
	res, err := http.Get(translateUrl)
	if err != nil {
		log.Println(err)
		return "", nil
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Println(err)
		return "", nil
	}
	var transResp TransResp
	err = json.Unmarshal(body, &transResp)
	if err != nil {
		log.Println(err)
		return "", nil
	}
	fmt.Printf("translate result: %s, text result: %s\n", string(body), transResp.TransResult[0].Dst)
	return transResp.TransResult[0].Dst, nil
}

func convertAudioFile(beforeFile, afterAmrFile string) error {
	cmdStr := "ffmpeg -y -i " + beforeFile + " -acodec amr_wb -ac 1 -ar 16000 -ab 23850 " + afterAmrFile
	_, err := exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		log.Println("convert file: ", err)
		return err
	}
	if err = os.Remove(beforeFile); err != nil {
		log.Println("convert file: ", err)
		return err
	}
	return nil
}

func translate(msg *ChatMsgJson) error {
	if msg.Catalog == "text" {
		//构造会话目录
		dir := "../tcpBoltDB/" + "upload/" + msg.FromUser + "/" + msg.ToUser + "/"
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Println(err)
			return err
		}

		to, err := translateText(msg.FromLang, msg.ToLang, msg.FromText)
		if err != nil {
			log.Println(err)
			return err
		}
		msg.ToText = to

		nowStr := strconv.FormatInt(time.Now().Unix(), 10)
		//文字转语音
		audioResultMp3File := dir + nowStr + "_result.mp3"
		audioResultAmrFile := dir + nowStr + "_result.amr"
		audioDownloadAmrFile := "download/" + msg.FromUser + "/" + msg.ToUser + "/" + nowStr + "_result.amr"
		err = text2audio(msg.ToText, audioResultMp3File, audioResultAmrFile)
		if err != nil {
			log.Println(err)
			return err
		}
		msg.ToAudioUrl = audioDownloadAmrFile
	} else {
		//语音翻译, Catalog == "audio"
		//构造会话目录
		dir := "../tcpBoltDB/" + "upload/" + msg.FromUser + "/" + msg.ToUser + "/"
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Println(err)
			return err
		}
		//语音生成amr文件到会话目录
		decoded, err := base64.StdEncoding.DecodeString(msg.FromAudio)
		if err != nil {
			log.Println("decode error:", err)
			return err
		}
		nowStr := strconv.FormatInt(time.Now().Unix(), 10)
		audioCvtBefFile := dir + nowStr + "_cvtbef.amr"
		audioAmrFile := dir + nowStr + ".amr"

		err = ioutil.WriteFile(audioCvtBefFile, decoded, 0644)
		if err != nil {
			log.Println(err)
			return err
		}
		//语音文件格式适配
		if err = convertAudioFile(audioCvtBefFile, audioAmrFile); err != nil {
			log.Printf("convert audio file format failed, %s", err.Error())
			return err
		}

		//语音转文字
		var text string
		text, err = audio2text(audioAmrFile)
		if err != nil {
			log.Println(err)
			return err
		}
		msg.FromText = text
		//文字翻译
		to, err := translateText(msg.FromLang, msg.ToLang, msg.FromText)
		if err != nil {
			log.Println(err)
			return err
		}
		msg.ToText = to
		//文字转语音
		audioResultMp3File := dir + nowStr + "_result.mp3"
		audioResultAmrFile := dir + nowStr + "_result.amr"
		audioDownloadAmrFile := "download/" + msg.FromUser + "/" + msg.ToUser + "/" + nowStr + "_result.amr"
		err = text2audio(msg.ToText, audioResultMp3File, audioResultAmrFile)
		if err != nil {
			log.Println(err)
			return err
		}
		msg.ToAudioUrl = audioDownloadAmrFile
		msg.FromAudio = ""
	}
	return nil
}

func getToken() (string, error) {
	//获取Access Token
	apiKey := "WHdlotz6Il2bnGPtHvtDw3Wa"
	secretKey := "LwjawyMjWD15oHgKCgcWhH4kI6PCHXxe"
	urlToken := "https://openapi.baidu.com/oauth/2.0/token?grant_type=client_credentials&client_id=" + apiKey + "&client_secret=" + secretKey
	res, err := http.Get(urlToken)
	if err != nil {
		log.Println(err)
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Println(err)
		return "", err
	}
	var t TokenResult
	err = json.Unmarshal(body, &t)
	if err != nil {
		log.Println(err)
		return "", err
	}
	fmt.Printf("token result: %s\n", t.AccessToken)
	return t.AccessToken, nil
}

func audio2text(audioAmrFile string) (string, error) {
	cuid := "TranslateChat"
	log.Printf("Start audio2text for file %s\n", audioAmrFile)
	//语音到文本
	var file *os.File
	var err error
	file, err = os.Open(audioAmrFile)
	if err != nil {
		log.Println(err)
		return "", err
	}
	urlAudio2Text := "http://vop.baidu.com/server_api?lan=zh&cuid=" + cuid + "&token=" + token
	var resp *http.Response
	resp, err = http.Post(urlAudio2Text, "audio/amr;rate=16000", file)
	if err != nil {
		log.Println(err)
		return "", err
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Println(err)
		return "", err
	}
	var a2tRet Audio2TextResult
	err = json.Unmarshal(body, &a2tRet)
	if err != nil {
		log.Println(err)
		return "", err
	}
	if a2tRet.ErrNo != 0 {
		err = fmt.Errorf("Audio2Text failed, %s", a2tRet.ErrMsg)
		log.Printf("Audio2Text failed, url: %s, result from server %s, %#v\n", urlAudio2Text, string(body), a2tRet)
		return "", err
	}

	log.Printf("Audio2Text result: %s\n", a2tRet.Result[0])
	return a2tRet.Result[0], nil
}

func text2audio(text, mp3File, saveAudioFile string) error {
	cuid := "TranslateChat"
	var err error
	//文本到语音
	message := text
	urlMsg := url.QueryEscape(message)
	urlText2Audio := "http://tsn.baidu.com/text2audio?tex=" + urlMsg + "&lan=zh&cuid=" + cuid + "&ctp=1&tok=" + token
	var res *http.Response
	res, err = http.Get(urlText2Audio)
	if err != nil {
		log.Println(err)
		return err
	}

	var body []byte
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		return err
	}
	defer res.Body.Close()

	if res.Header.Get("Content-type") != "audio/mp3" {
		log.Printf("text2audio failed: %s\n", string(body))
		return err
	}

	err = ioutil.WriteFile(mp3File, body, 0644)
	if err != nil {
		log.Println(err)
		return err
	}

	cmdStr := "lame " + mp3File + " " + saveAudioFile
	_, err = exec.Command("bash", "-c", cmdStr).Output()
	if err != nil {
		log.Println(err)
		return err
	}
	if err = os.Remove(mp3File); err != nil {
		log.Println(err)
	}
	return nil
}
func UpdateToken() {
	dir := "/tmp/translatechat/"
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if t, err := getToken(); err != nil {
			log.Fatal(err)
			time.Sleep(time.Second * 20)
		} else {
			token = t
			err = ioutil.WriteFile(dir+"token", []byte(token), 0644)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("Updated token %s\n", token)
			time.Sleep(time.Hour * 24 * 20)
		}
	}
}
func main() {
	var (
		err error
		p   *nsq.Producer
		c   *nsq.Consumer
	)
	go UpdateToken()
	//wait for token initialzied
	for {
		if token == "" {
			time.Sleep(time.Second * 10)
		} else {
			break
		}
	}

	//初始化Producer和Consumer
	p, err = nsq.NewProducer("127.0.0.1:4150", nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
		return
	}
	c, err = nsq.NewConsumer("translateBefore", "channelA", nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
		return
	}

	hand := func(msg *nsq.Message) error {
		//先接收翻译前的数据
		//decode
		m, _, err := packet.Decode(packet.ProtocolV311, msg.Body)
		if err != nil {
			log.Printf("decode message failed, msg: %s\n", string(msg.Body))
			return topicsTypes.ErrUnexpectedObjectType
		}

		pubPkt, ok := m.(*packet.Publish)
		if !ok {
			log.Printf("decode message converted to publish pkt failed, pkt %#v\n", m)
			return topicsTypes.ErrUnexpectedObjectType
		}

		fmt.Printf("got new message from channel translateBefore, topic: %s\n", string(pubPkt.Topic()))

		var chatMsg ChatMsgJson
		if err = json.Unmarshal(pubPkt.Payload(), &chatMsg); err != nil {
			log.Printf("unmarshal publish pkt failed, payload: %s, err: %s\n", string(pubPkt.Payload()), err.Error())
			return err
		}
		//翻译
		if err = translate(&chatMsg); err != nil {
			log.Printf("translate message failed, chatMsg: %#v, err %s\n", chatMsg, err.Error())
			return err
		}
		//返回
		var ret []byte
		ret, err = json.Marshal(chatMsg)
		if err != nil {
			log.Printf("marshal into publish pkt failed, chatMsg: %#v, err %s\n", chatMsg, err.Error())
			return err
		}
		pubPkt.SetPayload(ret)
		//encode Mqtt  Publish pkt into NSQ Message
		var buff []byte
		buff, err = packet.Encode(pubPkt)
		if err != nil {
			log.Printf("encode message into []byte failed, err %s\n", err.Error())
			return err
		}
		//发送翻译后的数据到消息队列
		err = p.Publish("translateAfter", buff)
		if err != nil {
			log.Printf("publish to translateAfter failed, err %s\n", err.Error())
			return err
		}

		return nil
	}
	c.AddHandler(nsq.HandlerFunc(hand))
	if err := c.ConnectToNSQLookupd("127.0.0.1:4161"); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Entering sleep...")
	blockForever()
}
func blockForever() {
	c := make(chan struct{})
	<-c
}
