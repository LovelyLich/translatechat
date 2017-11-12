package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
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
			FromUser:   "8618100805249",
			ToUser:     "18358183215",
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
			FromUser:   "8618100805249",
			ToUser:     "18358183215",
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

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
}

func main() {
	server := flag.String("server", "tcp://27.155.100.158:1883", "The full URL of the MQTT server to connect to")
	isText := flag.Bool("isText", false, "是文本翻译")
	topic := flag.String("topic", "/8618100805249/18358183215/messages", "Topic to publish the messages on")
	username := flag.String("username", "8618100805249", "A username to authenticate to the MQTT server")
	password := flag.String("password", "P6vdnfjlMTBlZ1p", "Password to match username")
	flag.Parse()

	connOpts := MQTT.NewClientOptions().AddBroker(*server).SetClientID("testclientA").SetCleanSession(false)
	if *username != "" {
		connOpts.SetUsername(*username)
		if *password != "" {
			connOpts.SetPassword(*password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	client := MQTT.NewClient(connOpts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return
	}
	fmt.Printf("Connected to %s\n", *server)

	//订阅消息
	client.Subscribe(*topic, 2, onMessageReceived)
	//构造Message, 并Publish
	var msg *ChatMsgJson
	var err error

	msg, err = getChatMsg(*isText)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("getChatMsg: %#v\n", *msg)

	var buff []byte
	buff, err = json.Marshal(*msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	client.Publish(*topic, 2, false, buff)
	fmt.Println("Published Chat Message")
	time.Sleep(time.Second * 1000)
}
