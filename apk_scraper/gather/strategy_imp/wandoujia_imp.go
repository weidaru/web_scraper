package strategy_imp

import(
	"github.com/weidaru/web_scraper/scraper"
	"strings"
	"sync"
	"os"
	"log"
	"encoding/xml"
)

type Log struct {
	XMLName xml.Name `xml:"log"`
    Version string   `xml:"version,attr"`
    Items []Item `xml:"item"`
}

type Item struct {
	URL string `xml:"url,attr"`
}

func CreateDumpCallback(filename string, max_count int) scraper.ExtractCallback {
	file,err := os.OpenFile(filename, os.O_RDWR | os.O_CREATE, 0666)
	logxml := &Log{Version:"0.1"}
	count := 0
	var mutex sync.Mutex
	if err != nil {
		log.Println(err)
		return nil
	}
	dump := func(input interface{}) {
		mutex.Lock()
		count++
		if(count >= max_count) {
			output, err := xml.MarshalIndent(logxml, "  ", "    ")
			if err != nil {
				log.Println(err)
			}else {
				file.Write(output)
				file.Close()
			}
		}else {
			url := input.(string)
			logxml.Items = append(logxml.Items, Item{URL:url})
		}
		mutex.Unlock()
	}
	return dump
}

func CreateStrategy4wandoujia(max_count int) scraper.Strategy  {
	url_map := map[string]bool{}
	var mutex sync.Mutex
	extract_ref := func(new_url string) bool {
		mutex.Lock()
		result := false
		if strings.Index(new_url, "download") != -1 && 
			url_map[new_url]==false{
			result = true
			url_map[new_url]=true
		}
		mutex.Unlock()
		return result
	}
	crawl_href := func(new_url string) bool {
		if strings.Index(new_url, "www.wandoujia.com") != -1 &&
		   strings.Index(new_url, "comment") == -1{
			return true
		} else {
			return false
		}
	}
	strategy := scraper.CreateHrefBasedStrategy(max_count, extract_ref, crawl_href)
	
	//Add dump to callback.
	dump := CreateDumpCallback("log.xml", max_count)
	strategy.Callbacks = append(strategy.Callbacks, dump)
	
	return strategy
}







