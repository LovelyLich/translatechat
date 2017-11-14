package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func onMessageReceived(client MQTT.Client, message MQTT.Message) {
	fmt.Printf("Received message on topic: %s\nMessage: %s\n", message.Topic(), message.Payload())
}

func main() {
	server := "tcp://27.155.100.158:1883"
	topic := "/sys/8618100805249/message"
	username := "8618100805257"
	password := "XqrCtvxCzZh7Z8x"
	addFriendUrl := "http://27.155.100.158:3389/friends/add_friend?phoneNo=8618100805249"

	connOpts := MQTT.NewClientOptions().AddBroker(server).SetClientID("testclientA").SetCleanSession(false)
	connOpts.SetUsername(username)
	connOpts.SetPassword(password)
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert}
	connOpts.SetTLSConfig(tlsConfig)

	mqttClient := MQTT.NewClient(connOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		return
	}
	fmt.Printf("Connected to %s\n", server)

	//订阅系统消息
	mqttClient.Subscribe(topic, 2, onMessageReceived)
	//添加朋友关系
	//curl -H "Content-Type: application/json" -H "Auth: 18100805251_qQUXbv3guQRlPEr_356696080597029" http://127.0.0.1:8080/friends/add_friend?phoneNo=18358183215
	authStr := `8618100805257_XqrCtvxCzZh7Z8x_356696080597022`
	var resp *http.Response
	client := &http.Client{}
	req, err := http.NewRequest("GET", addFriendUrl, nil)
	req.Header.Add("Auth", authStr)
	resp, err = client.Do(req)
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

	time.Sleep(time.Second * 1000)
}
