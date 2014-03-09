package main

import (
	"github.com/weidaru/web_scraper/scraper/downloader"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.Lshortfile)
	
	d := downloader.New("download.xml")
	d.Request("http://apps.wandoujia.com/apps/com.zhimahu/download")

	d.Start("d:\\temp", func(response *http.Response) bool {
		if response.StatusCode == 200 && 
			response.Header.Get("Content-Type") == "application/vnd.android.package-archive" {
			return true
		}else {
			return false
		}
	})
}


