package strategy_imp

import(
	"github.com/weidaru/web_scraper/scraper"
	"strings"
)

func CreateStrategy4wandoujia(max_count int) scraper.Strategy  {
	extract_ref := func(new_url string) bool {
		if strings.Index(new_url, "download") != -1 {
			return true
		} else {
			return false
		}
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
	
	return strategy
}







