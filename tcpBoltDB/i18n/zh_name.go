package main

import (
	"fmt"
	"log"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func ExampleScrape() {
	f, err := os.OpenFile("./86_name.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	doc, err := goquery.NewDocument("http://www.qmsjmfb.com/")
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		name := s.Text()
		nameRune := []rune(name)

		var text string
		if len(nameRune) == 3 {
			firstname := nameRune[0]
			secondname := nameRune[1]
			lastname := nameRune[2]

			text = fmt.Sprintf("%s %s %s\n", string(firstname), string(secondname), string(lastname))
		} else if len(nameRune) == 4 {
			firstname := nameRune[:2]
			secondname := nameRune[2]
			lastname := nameRune[3]

			text = fmt.Sprintf("%s %s %s\n", string(firstname), string(secondname), string(lastname))
		}

		if _, err = f.WriteString(text); err != nil {
			panic(err)
		}
	})
}

func main() {
	ExampleScrape()
}
