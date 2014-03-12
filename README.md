web_scraper  
===========  
  
Dependencies:  
1. lfreequeue: go get github.com/scryner/lfreequeue  
2. go-html-transform: go get code.google.com/p/go-html-transform/h5  

Test:  
Try test by running ./test/main.go  
something go run ./test/main.go http://www.google.com 100  
This will pull 100 website titles starting from http://www.google.com  
  
TODO:  
Add makefile  