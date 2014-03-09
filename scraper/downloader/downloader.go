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
	"path/filepath"
	"errors"
)

const Version string = "0.1"
const TempFileName string = "660115.651220"
const TempSuffix string = ".notdone"

type LogItem struct {
	URL string `xml:"url"`
	LogTime time.Time `xml:"time"`
}

type Log struct {
	XMLName xml.Name `xml:"download_log"`
	Version string `xml:"version,attr"`
	Items []LogItem `xml:"item"`
}

type Downloader struct {
	urls []string
	log_filepath string
	log_map map[string]LogItem
}

func New(log_filepath string) *Downloader {
	d := new(Downloader)
	d.log_filepath = log_filepath
	d.log_map = make(map[string]LogItem)
	
	var open_flag int
	if _, err := os.Stat(log_filepath); os.IsNotExist(err) {
		open_flag = os.O_RDONLY  | os.O_CREATE
	} else {
		open_flag = os.O_RDONLY 
	}
	
	log_file,err := os.OpenFile(log_filepath, open_flag, 0666)
	if err != nil {
		log.Println(err)
		log_file = nil
	} else {
		if open_flag & os.O_CREATE == 0 {
			d.LoadLog(log_file)
		}
	}
	log_file.Close()
	d.SaveLog()
	
	return d
}

func (d *Downloader) LoadLog(log_file *os.File) {
	if log_file == nil {
		log.Println("Cannot load log when log file is not open.")
		return
	}
	logxml := &Log{}
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
	if logxml.Version != Version {
		log.Println("Version does not match, expect " + Version + ", get " + logxml.Version)
	}
	d.Clear()
	for _,item:=range logxml.Items {
		d.log_map[item.URL] = item
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
	logxml := &Log{Version:Version}
	for _,item := range d.log_map {
		logxml.Items = append(logxml.Items, item)
	}
	output, err := xml.MarshalIndent(logxml, "  ", "    ")
	if err != nil {
		log.Println(err)
		return
	}
	temp_filepath := ReplaceFilePath(d.log_filepath, TempFileName)
	
	log_file, err := os.OpenFile(temp_filepath, os.O_WRONLY | os.O_CREATE, 0666)
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
	if _, err := os.Stat(d.log_filepath); os.IsNotExist(err)==false {
		os.Remove(d.log_filepath)									//I suppose we can remove this line if we are working on linux. Need test.
	} 
	err = os.Rename(temp_filepath, d.log_filepath)		//Hopefully this is atomic
	if err != nil {
		log.Println(err)
	}
	os.Remove(temp_filepath)
}

func (d *Downloader) Request(url string)  {
	d.urls = append(d.urls, url)
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
/**
 *	Note: this function is revised from golang source code, io.Copy
 */
var ErrShortWrite = errors.New("short write")
var EOF = errors.New("EOF")
func Copy(dst io.Writer, src io.Reader, buffer_size int) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}
	buf := make([]byte, buffer_size)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = ErrShortWrite
				break
			}
		}
		if er == EOF {
			break
		}
		if er != nil {
			err = er
			break
		}
	}
	return written, err
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
	for _,item:=range d.urls {
		log.Println("Processing ", item)
		if _,ok := d.log_map[item]; ok {
			log.Println("Already Downloaded.")
			continue
		}
		response, err := http.Get(item)
		if err != nil {
			log.Println(err)
			continue
		}
		if response_check(response) {
			d.HandleDownload(dest_dir, item, response)
		}else {
			log.Println("Discard")
		}
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	}
}

func (d *Downloader) HandleDownload(dest_dir string, url string, response *http.Response) {
	file_path := GetLast(response.Request.URL.String())
	file_path = filepath.Join(dest_dir, file_path)
	temp_path := file_path + TempSuffix
	save_file, err:= os.Create(temp_path)
	log.Println("Save to ", file_path)
	if err!=nil {
		log.Println("Error")
		log.Println(err)
	} else {
		_,err:=Copy(save_file, response.Body, 512*1024)		//512k cache.
		response.Body.Close()
		save_file.Close()
		if _, err := os.Stat(file_path); os.IsNotExist(err)==false {
			os.Remove(file_path)									//I suppose we can remove this line if we are working on linux. Need test.
		} 
		err=os.Rename(temp_path, file_path)
		if err != nil {
			log.Println("Error ", err)
			return
		} else {
			d.log_map[url] = LogItem{url, time.Now()}
			d.SaveLog()
			log.Println("Done")
		}
	}
}

//Clear the log and download everything from start to end
func (d *Downloader) Clear() {
	d.log_map = make(map[string]LogItem)
}












