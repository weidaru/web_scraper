package main

import (
	"github.com/weidaru/web_scraper/scraper"
	"log"
	"./strategy_imp"
	"runtime"
)

func main() {
	log.SetFlags( log.Ldate | log.Ltime | log.Lshortfile)
	
	strategy := strategy_imp.CreateStrategy4wandoujia(1000)
	
	runtime.GOMAXPROCS(runtime.NumCPU())
	context := scraper.New(runtime.NumCPU())
	context.Add("http://www.wandoujia.com/apps", strategy)
	context.Run()
}




