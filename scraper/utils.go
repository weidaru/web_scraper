package scraper

import(
	"log"
	"sync"
	"sync/atomic"
	"code.google.com/p/go.net/html"
	"code.google.com/p/go-html-transform/css/selector"
	urllib "net/url"
)

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

func CreateHrefBasedStrategy(max_count int, ext_href ExtractHrefFunc, crawl_href CrawlHrefFunc ) Strategy {
	strategy := Strategy{}
	count := 0
	var count_mx sync.Mutex		//Protect count
	
	strategy.Extract = func(url string, node *html.Node) ([]interface{}, bool) {
		result := make([]interface{}, 0)
		should_stop := false
		sel,err := selector.Selector("[href]")
		if err!=nil {
			log.Println(err)
			return nil, should_stop
		}
		nodes := sel.Find(node)
		
	OutLoop:
		for _,v := range nodes {
			for _,attr := range v.Attr {
				if(attr.Key == "href") {
					count_mx.Lock()
					if count == max_count {
						count_mx.Unlock()
						should_stop = true
						break OutLoop
					}
					
					new_url := SolveURL(url, attr.Val)
					should_add := ext_href(new_url)
					if should_add {
						result = append(result, new_url)
						count++
					}
					count_mx.Unlock()
					break
				}
			}
		}
		return result, should_stop
	}
	
	strategy.Crawl = func(url string, node *html.Node) []string {
		result := make([]string, 0)	
		sel,err := selector.Selector("[href]")
		if err!=nil {
			log.Println(err)
			return nil
		}
		href_nodes := sel.Find(node)
	OutLoop:
		for _,v := range href_nodes {
			for _,attr := range v.Attr {
				if attr.Key == "href"  {
					count_mx.Lock()
					if count == max_count {
						count_mx.Unlock()
						break OutLoop
					}
					count_mx.Unlock()
					
					new_url := SolveURL(url, attr.Val)
					should_add := crawl_href(new_url)
					if should_add {
						result = append(result, new_url)
					}
				}
			}
		}
		return result
	}
	
	cbs := make([]ExtractCallback, 0)
	cbs = append(cbs, CreateDebugCallbacks())
	strategy.Callbacks = cbs
	
	return strategy
}