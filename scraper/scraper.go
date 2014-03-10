package scraper

import (
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"net/http"
	"log"
	"sync"
	"sync/atomic"
	"strings"
	lf "github.com/scryner/lfreequeue"
	"time"
)

type ExtractFunc func(url string, node *html.Node) ([]interface{},bool)
type CrawlFunc func(url string, node *html.Node) []string
type ExtractCallback func(input interface{})

type Strategy struct {
	Extract ExtractFunc
	Crawl CrawlFunc
	Callbacks []ExtractCallback
}

type State interface {
	Update()
	CheckStop() bool
	Copy() State
}

type DefaultState struct {
	depth int
	maxdepth int
}

func (s *DefaultState) Update() {
	s.depth++
}

func (s *DefaultState) CheckStop() bool {
	if s.depth>=s.maxdepth {
		return true
	}else {
		return false
	}
}

func (s *DefaultState) Copy() State {
	new_state := *s
	return &new_state
}

type Item struct {
	url_ string
	strategy_ Strategy
	state_ State
}

type Scraper struct {
	items *lf.Queue				//Better to use a lock free queue
	stop_flag int32				//stop_flag is non-0 if stop is requested
	group sync.WaitGroup
	num_thread int
}

func New(num_thread int) *Scraper {
	result := new(Scraper)
	result.stop_flag = 0
	result.items = lf.NewQueue()
	result.num_thread = num_thread
	return result
}

func (s *Scraper) Add(url string, strategy Strategy) {
	state := DefaultState{depth:0, maxdepth:10}
	s.AddWithState(url, strategy, &state)
}

func  (s *Scraper) AddWithState(url string, strategy Strategy, state State) {
	item := Item{url_:url, strategy_:strategy, state_:state}
	s.items.Enqueue(item)
}

type executeFunc func(url string, strategy Strategy, state State)

func (s *Scraper) Run() {
	execution := s.createExecution();
	s.group.Add(s.num_thread)
	for i:=0; i<s.num_thread; i++ {
		go s.runWorkThread(execution)
	}

	s.group.Wait()
}

//Private

func (s *Scraper) runWorkThread(execution executeFunc) {
	defer s.group.Done()
	attempt := 0
	for {
		if atomic.LoadInt32(&s.stop_flag) != 0 || attempt > 10 {
			break
		}
		
		if i,ok := s.items.Dequeue(); ok {
			item := i.(Item)
			execution(item.url_, item.strategy_, item.state_)
			attempt = 0
		} else {
			time.Sleep(1000 * time.Millisecond)
			attempt++
		}
	}
}

func getHTMLTree(url string) *html.Node {
	//Start get Body of the html if it really is
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil
	}
	mime := response.Header.Get("Content-Type")
	if response.StatusCode != 200 || strings.Index(mime, "text/html") == -1 {
		log.Println("Bad url ", url , "StatusCode: ", response.StatusCode, "MIME: ", mime)
		return nil
	}
	tree, err := h5.New(response.Body)
	if err != nil {
		log.Println(err)
		return nil
	}
	
	
	return tree.Top()
}

func (s *Scraper) createExecution() executeFunc {
	url_map := map[string]bool{}
	var mutex sync.Mutex			//mutex to protect url_map
		
	var internal func(string, Strategy, State)
	internal = func(url string, strategy Strategy, state State) {
		mutex.Lock()
		if url_map[url] == true {
			mutex.Unlock()
			return 
		}
		
		url_map[url] = true
		mutex.Unlock()
		
		root := getHTMLTree(url)
		if root == nil {
			return
		}
		
		//Do extractions
		extracts, should_stop := strategy.Extract(url, root)
		for _,v := range extracts {
			for _,cb := range strategy.Callbacks {
				cb(v)
			}
		}
		if should_stop {
			atomic.AddInt32(&s.stop_flag, 1)
			return
		}
		
		state.Update()
		if(state.CheckStop()) {
			return
		}
		
		//Do crawling
		new_urls := strategy.Crawl(url, root)
		
		for _,new_url := range new_urls {
			mutex.Lock()
			if !url_map[new_url] {
				new_state := state.Copy()

				i := Item{url_:new_url, strategy_:strategy, state_:new_state}
				s.items.Enqueue(i)
			}
			mutex.Unlock()
		}
	}
	
	return internal
}














