package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/mux"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/sclevine/agouti"
	"github.com/unrolled/render"
	"gopkg.in/go-playground/validator.v9"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var rendering = render.New(render.Options{})
var validate = validator.New()
var client = &http.Client{}
var bot, _ = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("ACCESS_TOKEN"), linebot.WithHTTPClient(client))

type Response struct {
	Message string `json:"message"`
}

type data struct {
	A []string `json:"あ"`
	I []string `json:"い"`
}

func siritori(msg *linebot.TextMessage) (string, error) {
	var indexWords data
	jsonFile, err := os.Open(`data.json`)
	if err != nil {
		return "", err
	}
	defer jsonFile.Close()
	if err := json.NewDecoder(jsonFile).Decode(&indexWords); err != nil {
		return "", err
	}
	var response *string
	message := msg.Text
	runes := []rune(message)
	lastChar := string(runes[len(runes)-1])
	switch lastChar {
	case "あ":
		response = &indexWords.A[ rand.Intn(4)]
	case "い":
		response = &indexWords.I[rand.Intn(4)]
	default:
		response = nil
	}
	return *response, nil
	}


func replyMessage(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		rendering.JSON(w, http.StatusInternalServerError, &Response{
			Message: "Error",
		})
	}
	for _, event := range events {
		replyToken := event.ReplyToken
		if event.Type == linebot.EventTypeMessage {
			switch msg := event.Message.(type) {
			case *linebot.TextMessage:
				response, err := siritori(msg)
				if err != nil {
					log.Fatal(err)
				}
				if _, err := bot.ReplyMessage(replyToken, linebot.NewTextMessage(response)).Do(); err != nil {
					log.Fatal(err)
				}
			default:
				log.Print("No Text Message")
			}
		}
	}
}

func makeJsonFile() {
	var index []int
	var content []byte
	for i := 2; i < 5; i++ {
		index = append(index, i)
	}
	for i := range index {
		url := "http://siritori.net/tail/item/%E3%82%8A/page:" + strconv.Itoa(i)
		driver := agouti.ChromeDriver()
		err := driver.Start()
		if err != nil {
			log.Printf("Failed to start driver: %v", err)
		}
		defer driver.Stop()
		page, err := driver.NewPage(agouti.Browser("chrome"))
		if err != nil {
			log.Printf("Failed to open page: %v", err)
		}
		err = page.Navigate(url)
		if err != nil {
			log.Printf("Failed to navigate: %v", err)
		}
		curContentsDom, err := page.HTML()
		if err != nil {
			log.Printf("Failed to get html: %v", err)
		}

		var mapData = map[string][]string{}
		readerCurContents := strings.NewReader(curContentsDom)
		contentsDom, err := goquery.NewDocumentFromReader(readerCurContents) // Get page DOM from opening browser
		if err != nil {
			log.Printf("Failed to generate document: %v", err)
		}
		contentsDom.Find("html > body > div#main > div.pages.archive > ul.linkClound.clear > li").Each(func(_ int, s *goquery.Selection){
			data := s.Find("a").Text() // Get text element
			runes := []rune(data)
			firstChar := string(runes[0])
			mapData[firstChar] = append(mapData[firstChar], data)
		})
		dataJson, err := json.Marshal(mapData)
		if err != nil {
			log.Printf("Failed to encode data: %v", err)
		}
		content = append(content, dataJson...)
	}
	ioutil.WriteFile("data.json", content, os.ModePerm) // Make json file based on scraping data
}

func main() {
	makeJsonFile()
	router := mux.NewRouter()
	router.Path("/health_check").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK)},
		).Methods(http.MethodGet)

	router.Path("/callback").HandlerFunc(replyMessage).Methods(http.MethodPost)

	s := http.Server{
		Addr:    ":8080",
		Handler: router,
	}
	log.Println("=== Start ===")
	log.Fatal(s.ListenAndServe())
}