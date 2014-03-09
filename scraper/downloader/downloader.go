package downloader

import (
	"time"
	"encoding/xml"
	"os"
	"io/ioutil"
	"io"
	"log"
	"regexp"
	"net/http"
)

const Version string = "0.1"
const TempFileName string = "660115.651220"

type LogItem struct {
	url string			`xml:"url"`
	log_time time.Time	`xml:"time"`
}

type Log struct {
	xml_name xml.Name	`xml:"download_log"`
	version string		`xml:"version,attr"`
	items []LogItem		`xml:"item"`
}

type Downloader struct {
	urls []string
	log_filepath string
	log_map map[string]LogItem
}

func New(log_filepath string) *Downloader {
	d := new(Downloader)
	d.log_filepath = log_filepath
	
	var open_flag int
	if _, err := os.Stat(log_filepath); os.IsNotExist(err) {
		open_flag = os.O_RDONLY  | os.O_CREATE
	} else {
		open_flag = os.O_RDONLY 
	}
	
	log_file,err := os.OpenFile(log_filepath, open_flag, 0666)
	defer log_file.Close()
	if err != nil {
		log.Println(err)
		log_file = nil
	} else {
		if open_flag & os.O_CREATE == 0 {
			d.LoadLog(log_file)
		}
	}
	
	return d
}

func (d *Downloader) LoadLog(log_file *os.File) {
	if log_file == nil {
		log.Println("Cannot load log when log file is not open.")
		return
	}
	logxml := Log{}
	data,err := ioutil.ReadAll(log_file)
	if err != nil {
		log.Println(err)
		return
	}
	err = xml.Unmarshal(data, &logxml)
	if err != nil {
		log.Println(err)
		return
	}
	if logxml.version != Version {
		log.Println("Version does not match, expect " + Version + ", get " + logxml.version)
	}
	d.Clear()
	for _,item:=range logxml.items {
		d.log_map[item.url] = item
	}
}

func ReplaceFilePath(old_path string, filename string) string {
	re := regexp.MustCompile("/|\\\\")
	match := re.FindAllStringIndex(old_path,-1)
	if len(match)==0 {
		return filename
	}
	last := match[len(match)-1] 
	return old_path[0:last[1]] + filename
}

func (d *Downloader) SaveLog() {
	logxml := &Log{version:Version}
	for _,item := range d.log_map {
		logxml.items = append(logxml.items, item)
	}
	output, err := xml.MarshalIndent(logxml, "  ", "    ")
	if err != nil {
		log.Println(err)
		return
	}
	temp_filepath := ReplaceFilePath(d.log_filepath, TempFileName)
	
	log_file, err := os.OpenFile(temp_filepath, os.O_WRONLY, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	_,err = log_file.Write(output)
	log_file.Close()
	if err != nil {
		log.Println(err)
		os.Remove(temp_filepath)
		return
	}
	os.Rename(temp_filepath, d.log_filepath)		//Hopefully this is atomic
}

func (d *Downloader) Request(url string) bool {
	if _,ok := d.log_map[url]; !ok {
		d.urls = append(d.urls, url)
		return true
	} else {
		return false
	}
}


func GetLast(url string) string {
	re := regexp.MustCompile("/")
	match := re.FindAllStringIndex(url,-1)
	if len(match)==0 {
		return ""
	}
	last := match[len(match)-1] 
	return url[last[1]:]
}

type ResponseCheckFunc func(response *http.Response) bool

/** I assume the network bandwidth is the bottleneck, thus network is 
 *  much slower than disk IO. At the same time, each apk file is relatively
 *  small, couple of mbs. So the strategy is making footprints after downloading
 *  each apk.
 */
func (d *Downloader) Start(dest_dir string, response_check ResponseCheckFunc) {
	err := os.MkdirAll(dest_dir, 0666)
	if err != nil {
		log.Println(err)
		return
	}
	d.SaveLog()
	for _,item:=range d.urls {
		log.Println("Processing ", item)
		if _,ok := d.log_map[item]; ok {
			continue
		}
		response, err := http.Get(item)
		if err != nil {
			log.Println(err)
			continue
		}
		if response_check(response) {
			log.Println("Start")
			filename := GetLast(response.Request.URL.String())

			save_file, err:= os.Create(dest_dir+filename)
			if err!=nil {
				log.Println("Error")
				log.Println(err)
			} else {
				io.Copy(save_file, response.Body)		//Does this block? Guess it should.
				response.Body.Close()
				d.log_map[item] = LogItem{item, time.Now()}
				d.SaveLog()
				log.Println("Done")
			}
		}else {
			log.Println("Discard")
		}
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	}
}

//Clear the log and download everything from start to end
func (d *Downloader) Clear() {
	d.log_map = make(map[string]LogItem)
}












