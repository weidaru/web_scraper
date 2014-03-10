package main

import (
	"github.com/weidaru/web_scraper/scraper"
	"log"
	"./strategy_imp"
)

func main() {
	log.SetFlags( log.Ldate | log.Ltime | log.Lshortfile)
	
	strategy := strategy_imp.CreateStrategy4wandoujia(5)
	
	context := scraper.New(4)
	context.Add("http://www.wandoujia.com/apps", strategy)
	context.Run()
}




