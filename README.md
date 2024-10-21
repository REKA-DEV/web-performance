# Simple Web Performance Test Tool

```text
Usage of web-performance.exe:
  -asset-host string
    	Hosts for assets
  -clients int
    	Number of clients to run (default 1)
  -connect-timeout int
    	<seconds> Maximum time allowed for connection
  -data string
    	HTTP POST data
  -delay int
    	Delay between requests
  -header value
    	<header> Pass custom header(s) to server
  -insecure
    	Allow insecure server connections when using SSL
  -iterations int
    	Number of iterations to run (default 1)
  -out string
    	Output file name (default "performance.html")
  -request string
    	<command> Specify request command to use (default "GET")
  -url string
    	URL
  -verify-body string
    	Verify request body
```

```shell
    ./web-performance -connect-timeout 1 -clients 100 -iterations 100 --verify-body "1" --request GET --url "http://testserver"
```
![performance](https://github.com/REKA-DEV/web-performance/blob/master/performance.png?raw=true)
