package main

import (
	"github.com/weidaru/web_scraper/scraper/downloader"
	"github.com/weidaru/web_scraper/scraper"
	"log"
	"net/http"
	"encoding/xml"
	"os"
	"io/ioutil"
)

func main() {
	log.SetFlags(log.Lshortfile)
	
	//Check arguments.
	if(len(os.Args) < 3) {
		log.Printf("usage: %s INPUT_FILE OUTPUT_DIR \n", os.Args[0])
		return
	}
	filename := os.Args[1]
	outdir := os.Args[2]
	
	//Read in xml and download
	d := downloader.New("download.xml")
	
	logxml := &scraper.Log{}
	log_file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	data,err := ioutil.ReadAll(log_file)
	if err != nil {
		log.Println(err)
		return
	}
	err = xml.Unmarshal(data, logxml)
	if err != nil {
		log.Println(err)
		return
	}
	for _, item := range logxml.LogItems {
		d.Request(item.URL)
	}

	d.Start(outdir, func(response *http.Response) bool {
		if response.StatusCode == 200 && 
			response.Header.Get("Content-Type") == "application/vnd.android.package-archive" {
			return true
		}else {
			return false
		}
	})
}


