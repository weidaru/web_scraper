package strategy_imp

import(
	"github.com/weidaru/web_scraper/scraper"
	"strings"
	"sync"
)

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
	dump := scraper.CreateDumpCallback("log.xml", max_count)
	strategy.Callbacks = append(strategy.Callbacks, dump)
	
	return strategy
}







