package main

import (
	"github.com/weidaru/web_scraper/scraper"
	"code.google.com/p/go.net/html"
	"runtime"
	"log"
	"os"
	"strconv"
)


func createStrategy(max_count int) scraper.Strategy {
	ext_func := func(url string, node *html.Node) (interface{}, bool) {
		return node.FirstChild.Data, false
	}
	ext_func = scraper.ExtractCountDecorator(max_count, ext_func)
	
	
	crawl_func := func(url string, node *html.Node) string {
		for _,attr := range node.Attr {
			if attr.Key == "href"  {	
				new_url := scraper.SolveURL(url, attr.Val)
				return new_url
			}
		}
		return ""
	}
	
	strategy := scraper.CreateSelectorBasedStrategy("title", ext_func, "[href]", crawl_func)
	return strategy
}

/*
 * Create a simple scraper which get title of website. Duplication of crawling is handled by the framework.
 */
func main() {
	log.SetFlags(log.Lshortfile)
	
	//Check arguments.
	if(len(os.Args) < 3) {
		log.Printf("usage: %s URL MAX_COUNT \n", os.Args[0])
		return
	}
	url := os.Args[1]
	max_count,err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Println(err)
		return
	}
	
	strategy := createStrategy(max_count)
	runtime.GOMAXPROCS(runtime.NumCPU())
	context := scraper.New(runtime.NumCPU())
	context.Add(url, strategy)
	context.Run()
}










