package main

import (
	"github.com/weidaru/web_scraper/scraper"
	"log"
	"os"
	"./strategy_imp"
)

func main() {
	log.SetFlags( log.Ldate | log.Ltime | log.Lshortfile)
	file,err := os.OpenFile("log.txt", os.O_RDWR | os.O_CREATE, 0666)
	defer file.Close()
	if err != nil {
		log.Println(err)
	}
	//log.SetOutput(file)
	
	strategy := strategy_imp.CreateStrategy4wandoujia(1000)
	
	context := scraper.New(4)
	context.Add("http://www.wandoujia.com/apps", strategy)
	context.Run()
}




