package main

import (
	"io/ioutil"
	"log"

	"github.com/holys/initials-avatar"
)

func main() {
	a := avatar.New(`./Hiragino_Sans_GB_W3.ttf`)
	//a := avatar.New("/System/Library/Fonts/Monaco.dfont")
	b, err := a.DrawToBytes("李易", 148)
	if err != nil {
		log.Println(err)
	}

	err = ioutil.WriteFile("./myavatar.png", b, 0644)
	if err != nil {
		log.Println(err)
	}
}
