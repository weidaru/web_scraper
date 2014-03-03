package scraper

import (
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go.net/html"
	"net/http"
	"log"
	"sync"
	"sync/atomic"
	"math/rand"
	"time"
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
	StopFlag int32				//StopFlag is non-0 if stop is requested
}

func New(num_thread int) *Scraper {
	result := new(Scraper)
	result.work = make(chan bool, num_thread)
	result.StopFlag = 0
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
		
		//rand.XXX() uses a global object with a mutex lock which may be slow, so we create a new generator here for each Execution(thread)
		source := rand.NewSource(time.Now().UnixNano())		
		generator := rand.New(source)
		
		//Start get Body of the html if it really is
		response, err := http.Get(url)
		if err != nil {
			log.Println(err)
			<-s.work
			return 
		}
		if response.StatusCode != 200 && response.Header.Get("Content-Type") != "text/html" {
			<-s.work
			return
		}
		tree, err := h5.New(response.Body)
		if err != nil {
			log.Println(err)
			<-s.work
			return
		}
		
		root:=tree.Top()
		extracts, should_stop := strategy.Extract(url, root)
		for _,v := range extracts {
			for _,cb := range strategy.Callbacks {
				cb(v)
			}
		}
		if should_stop {
			atomic.AddInt32(&s.StopFlag, 1)
			<-s.work
			return
		}
		
		state.Update()
		if(state.CheckStop()) {
			<-s.work
			return
		}
		
		new_urls := strategy.Crawl(url, root)
		
		for i:=0; i<10; i++ {
			new_url := new_urls[generator.Int() % len(new_urls)]
			
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
		if atomic.LoadInt32(&s.StopFlag) != 0 {
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
















