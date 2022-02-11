package main

//goland:noinspection ALL
import (
	"fmt"
	colly "github.com/gocolly/colly/v2"
	"github.com/vijayviji/executor"
	"log"
	"strconv"
)

type Proxy struct {
	IP   string
	port uint16
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

func main() {
	var proxies = extractProxies()

	fmt.Println(len(proxies))
	/*
		scraper := twitterscraper.New()
		scraper.SetSearchMode(twitterscraper.SearchLatest)

		var userTweets []twitterscraper.Tweet
		for tweet := range scraper.GetTweets(context.Background(), "Twitter", 10) {
			if tweet.Error != nil {
				panic(tweet.Error)
			}
			userTweets = append(userTweets, tweet.Tweet)
		}

		jsonString, err := json.Marshal(userTweets)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = ioutil.WriteFile("big_marhsall.json", jsonString, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return
		}
	*/
}
