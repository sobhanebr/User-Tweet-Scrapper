package main

//goland:noinspection ALL
import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	colly "github.com/gocolly/colly/v2"
	twitterscraper "github.com/n0madic/twitter-scraper"
	"github.com/vijayviji/executor"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

const MaxConcurrency uint32 = 50
const MaxTweetsPerUser = 100
const UserProcessPerProxy uint32 = 100
const OutputPath string = "tweets"

var counter uint32 = 0
var currentProxyIndex = 0

var proxies = extractProxies()

type Proxy struct {
	IP    string
	port  uint16
	https bool
}

type User struct {
	Id         uint64
	screenName string
}

func createUserList(data [][]string) []User {
	var userList []User
	for i, line := range data {
		if i > 0 { // skip header line
			var rec User
			for j, field := range line {
				if j == 0 {
					rec.Id, _ = strconv.ParseUint(field, 10, 64)
				} else if j == 1 {
					rec.screenName = field
				}
			}
			userList = append(userList, rec)
		}
	}
	return userList
}

func extractProxies() []Proxy {
	const proxyServer string = "https://free-proxy-list.net/"

	var proxies []Proxy

	collyCollector := colly.NewCollector(
		colly.Async(),
		colly.CacheDir("./proxy_list_cache"),
	)

	err := collyCollector.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 4})
	if err != nil {
		log.Fatal(err)
		return nil
	}

	collyCollector.OnHTML("table[class=\"table table-striped table-bordered\"] tbody", func(e *colly.HTMLElement) {
		e.ForEach("tr", func(_ int, row *colly.HTMLElement) {
			proxy := Proxy{}
			row.ForEach("td", func(_ int, el *colly.HTMLElement) {
				switch el.Index {
				case 0:
					proxy.IP = el.Text
				case 1:
					var port, err = strconv.Atoi(el.Text)
					if err != nil {
						log.Fatal(err)
					}
					proxy.port = uint16(port)
				case 6:
					proxy.https = el.Text == "yes"
				}
			})
			proxies = append(proxies, proxy)
		})
	})

	err2 := collyCollector.Visit(proxyServer)
	if err2 != nil {
		log.Fatal(err2)
		return nil
	}
	collyCollector.Wait()

	return proxies
}

func spreadTasks(userList []User) (future []executor.Future) {
	var futures []executor.Future
	createOutputDir()
	twitterscraper.SetSearchMode(twitterscraper.SearchLatest)
	ex := executor.NewFixedThreadPool("executorName", MaxConcurrency, 2000)
	for _, user := range userList {
		userFileName := fmt.Sprintf("%s/%d.json", OutputPath, user.Id)
		if _, err := os.Stat(userFileName); err == nil {
			log.Println(userFileName + " already exists, skipping")
			continue
		}
		var future = ex.Submit(func(taskData interface{}, threadName string, taskID uint64) interface{} {
			taskUser := taskData.(User)
			if counter == 0 {
				counter = UserProcessPerProxy
				err := twitterscraper.SetProxy(enrollProxy())
				if err != nil {
					log.Fatal(err)
				}
				counter--
			}
			var userTweets []twitterscraper.Tweet
			for tweet := range twitterscraper.WithDelay(5).GetTweets(context.Background(), taskUser.screenName, MaxTweetsPerUser) {
				if tweet.Error != nil {
					log.Println(tweet.Error)
				}
				userTweets = append(userTweets, tweet.Tweet)
			}

			jsonString, err := json.Marshal(userTweets)
			if err != nil {
				log.Println(err)
			}
			err = ioutil.WriteFile(userFileName, jsonString, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			return fmt.Sprintf("Tweets of user with ID %d was wrote in %s", user.Id, userFileName)
		}, user)
		futures = append(futures, future)
	}
	return
}

func enrollProxy() string {
	if currentProxyIndex == len(proxies) {
		currentProxyIndex = 0
	}
	var proxy = proxies[currentProxyIndex]
	currentProxyIndex++
	var prefix string
	if proxy.https {
		prefix = "https"
	} else {
		prefix = "http"
	}
	return fmt.Sprintf("%s://%s:%d", prefix, proxy.IP, proxy.port)
}

func createOutputDir() {
	if _, err := os.Stat(OutputPath); os.IsNotExist(err) {
		err := os.Mkdir(OutputPath, 0775)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func main() {

	// load users
	userList := loadUserList()
	futures := spreadTasks(userList)

	for _, future := range futures {
		result := future.Get()
		log.Println(result)
	}
}

func loadUserList() []User {
	file, err := os.Open("nonLocatedUserDetails.csv")
	if err != nil {
		log.Fatal(err)
	}

	// close the file at the end of the program
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)

	// read csv values using csv.Reader
	csvReader := csv.NewReader(file)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// convert records to array of structs
	return createUserList(data)
}
