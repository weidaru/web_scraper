package scraper

import (
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"net/http"
	"log"
	"sync"
	"sync/atomic"
)

type ExtractFunc func(url string, node *html.Node) ([]interface{},bool)
type CrawlFunc func(url string, node *html.Node) []string
type ExtractCallback func(input interface{})
type ExecuteFunc func(url string, strategy Strategy, state State)

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
	Execution ExecuteFunc
}

type Scraper struct {
	items []Item				//Better to use a lock free queue
	mutex sync.Mutex			//to protect items, length of items always increase which makes synchronization easy
	work chan bool				//work and group are used for thread control
	group sync.WaitGroup
	stop_flag int32				//stop_flag is non-0 if stop is requested
}

func New(num_thread int) *Scraper {
	result := new(Scraper)
	result.work = make(chan bool, num_thread)
	result.stop_flag = 0
	return result
}

func (s *Scraper) Add(url string, strategy Strategy) {
	state := DefaultState{depth:0, maxdepth:10}
	s.AddWithState(url, strategy, &state)
}

func  (s *Scraper) AddWithState(url string, strategy Strategy, state State) {
	item := Item{url_:url, strategy_:strategy, state_:state}
	item.Execution = s.CreateExecution()
	s.mutex.Lock()
	s.items = append(s.items, item)
	s.mutex.Unlock()
}

func GetHTMLTree(url string) *html.Node {
	//Start get Body of the html if it really is
	response, err := http.Get(url)
	if err != nil {
		log.Println(err)
		return nil
	}
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/html" {
		return nil
	}
	tree, err := h5.New(response.Body)
	if err != nil {
		log.Println(err)
		return nil
	}
	
	return tree.Top()
}

func (s *Scraper) CreateExecution() ExecuteFunc {
	url_map := map[string]bool{}
	var mutex sync.Mutex			//mutex to protect url_map
		
	var internal func(string, Strategy, State)
	internal = func(url string, strategy Strategy, state State) {
		defer s.group.Done()		//Cannot defer <-s.work? Because it is a channel and can block?
		mutex.Lock()
		if url_map[url] == true {
			mutex.Unlock()
			<-s.work
			return 
		}
		
		url_map[url] = true
		mutex.Unlock()
		
		root := GetHTMLTree(url)
		if root == nil {
			<-s.work
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
			<-s.work
			return
		}
		
		state.Update()
		if(state.CheckStop()) {
			<-s.work
			return
		}
		
		//Do crawling
		new_urls := strategy.Crawl(url, root)
		
		for _,new_url := range new_urls {
			mutex.Lock()
			if !url_map[new_url] {
				new_state := state.Copy()

				i := Item{url_:new_url, strategy_:strategy, state_:new_state, Execution:internal}
				s.mutex.Lock()
				s.items = append(s.items, i)
				s.mutex.Unlock()
			}
			mutex.Unlock()
		}
		<-s.work
	}
	
	return internal
}

func (s *Scraper) Run() {
	head := 0
	var item Item
	for {
		if atomic.LoadInt32(&s.stop_flag) != 0 {
			break
		}
		var length int
		s.mutex.Lock()
		length = len(s.items)
		s.mutex.Unlock()
		if head < length {
			s.work<-true			//Try sending work to see whether there is any empty slot(thread)
			s.group.Add(1)			//New work in the group
			s.mutex.Lock()
			item = s.items[head]	//Copy the item
			s.mutex.Unlock()
			go item.Execution(item.url_, item.strategy_, item.state_)
			head++
		}
	}
	s.group.Wait()
}
















