package scraper

import(
	"log"
	"math/rand"
	"sync/atomic"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go-html-transform/css/selector"
	urllib "net/url"
	"encoding/xml"
	"os"
	"sync"
)

type Log struct {
	XMLName xml.Name `xml:"log"`
    Version string `xml:"version,attr"`
    LogItems []LogItem `xml:"log_item"`
}

type LogItem struct {
	URL string `xml:"url,attr"`
}

func CreateDumpCallback(filename string, max_count int) ExtractCallback {
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
				_,err := file.Write(output)
				if err != nil {
					log.Println(err)
				}
				file.Close()
			}
		}else {
			url := input.(string)
			logxml.LogItems = append(logxml.LogItems, LogItem{url})
		}
		mutex.Unlock()
	}
	return dump
}

func CreateDebugCallbacks() ExtractCallback {
	var count int32
	return func(input interface{}) {
		//log is thread safe 
		log.Println(atomic.AddInt32(&count, 1), "  ", input)
	}
}

func SolveURL(old string, part string) string {
	_url,err := urllib.Parse(old)
	if err!=nil {
		return ""
	}
	new_url,err := _url.Parse(part)
	if err!= nil {
		return ""
	}
	return new_url.String()
}

type ExtractHrefFunc func(new_url string) bool
type CrawlHrefFunc func(new_url string) bool

type ExtractNodeFunc func(url string, node *html.Node) (interface{}, bool)
type CrawlNodeFunc func(url string, node *html.Node) string

func CreateSelectorBasedStrategy(ext_sel string, ext_func ExtractNodeFunc, crawl_sel string, crawl_func CrawlNodeFunc) Strategy {
	strategy := Strategy{}
	
	strategy.Extract = func(url string, node *html.Node) ([]interface{}, bool) {
		result := make([]interface{}, 0)
		should_stop := false
		sel,err := selector.Selector(ext_sel)
		if err!=nil {
			log.Println(err)
			return nil, should_stop
		}
		nodes := sel.Find(node)
		
		for _,v := range nodes {
			var res interface{}
			res,should_stop = ext_func(url, v)
			if should_stop {
				break
			}
			if res != nil {
				result = append(result, res)
			}
		}
		
		return result, should_stop
	}
	
	strategy.Crawl = func(url string, node *html.Node) []string {
		result := make([]string, 0)	
		sel,err := selector.Selector(crawl_sel)
		if err!=nil {
			log.Println(err)
			return nil
		}
		href_nodes := sel.Find(node)
		
		for i:=0; i<15; i++ {
			index := rand.Int() % len(href_nodes)
			v := href_nodes[index]
			res := crawl_func(url, v)
			if res != "" {
				result = append(result, res)
			}
		}
		return result
	}
	
	cbs := make([]ExtractCallback, 0)
	cbs = append(cbs, CreateDebugCallbacks())
	strategy.Callbacks = cbs
	
	return strategy
}

func ExtractCountDecorator(max_count int, ext_func ExtractNodeFunc) ExtractNodeFunc {
	var count int32
	count = 0
	
	return func(url string, node *html.Node) (interface{}, bool) {
		res,should_stop := ext_func(url, node)
		if res == nil {
			return res, should_stop
		}
		if(atomic.AddInt32(&count, 1) > int32(max_count)) {
			should_stop = true
		}
		return res, should_stop
	}
}

func CreateHrefBasedStrategy(max_count int, ext_href ExtractHrefFunc, crawl_href CrawlHrefFunc ) Strategy {
	ext_func := func(url string, node *html.Node) (interface{}, bool) {
		for _,attr := range node.Attr {
			if(attr.Key == "href") {
				var new_url string
				new_url = SolveURL(url, attr.Val)
				should_add := ext_href(new_url)
				if should_add {
					return new_url, false
				}
			}
		}
		return nil, false
	}
	ext_func = ExtractCountDecorator(max_count, ext_func)
	
	crawl_func := func(url string, node *html.Node) string {
		for _,attr := range node.Attr {
			if attr.Key == "href"  {	
				new_url := SolveURL(url, attr.Val)
				should_add := crawl_href(new_url)
				if should_add {
					return new_url
				}
			}
		}
		return ""
	}
	
	strategy := CreateSelectorBasedStrategy("[href]", ext_func, "[href]", crawl_func)
	
	return strategy
}










