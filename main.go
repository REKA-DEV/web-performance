package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type WebResult struct {
	err      error
	verify   bool
	duration time.Duration
}

type arrayFlag []string

func (af *arrayFlag) String() string {
	return fmt.Sprintf("%v", *af)
}

func (af *arrayFlag) Set(value string) error {
	*af = append(*af, value)
	return nil
}

func main() {
	timeout := flag.Int("connect-timeout", 0, "<seconds> Maximum time allowed for connection")
	data := flag.String("data", "", "HTTP POST data")
	insecure := flag.Bool("insecure", false, "Allow insecure server connections when using SSL")
	request := flag.String("request", "GET", "<command> Specify request command to use")

	clients := flag.Int("clients", 1, "Number of clients to run")
	iterations := flag.Int("iterations", 1, "Number of iterations to run")
	delay := flag.Int("delay", 0, "Delay between requests")

	verifyBody := flag.String("verify-body", "", "Verify request body")

	url := flag.String("url", "", "URL")
	out := flag.String("out", "performance.html", "Output file name")

	assetHost := flag.String("asset-host", "", "Hosts for assets")

	var headers arrayFlag
	flag.Var(&headers, "header", "<header> Pass custom header(s) to server")

	flag.Parse()

	client := &http.Client{
		Timeout: time.Duration(*timeout) * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: *insecure,
			},
		},
	}

	arr := make([]WebResult, (*clients)*(*iterations))

	for i := 0; i < (*clients)*(*iterations); i++ {
		arr[i].err = nil
		arr[i].verify = false
		arr[i].duration = time.Duration(0)
	}

	wg := sync.WaitGroup{}
	wg.Add(*clients)

	for i := 0; i < *clients; i++ {
		go func(ii int) {
			for j := 0; j < *iterations; j++ {
				var idx = ii*(*iterations) + j

				var httpRequest *http.Request
				if len(*data) == 0 {
					httpRequest, _ = http.NewRequest(*request, *url, nil)
				} else {
					httpRequest, _ = http.NewRequest(*request, *url, bytes.NewBufferString(*data))
				}

				for _, header := range headers {
					idx := strings.Index(header, ":")
					httpRequest.Header.Set(strings.TrimSpace(header[:idx]), strings.TrimSpace(header[idx+1:]))
				}

				start := time.Now()

				response, err := client.Do(httpRequest)
				if err != nil {
					arr[idx].err = err
					log.Printf("%d %d >> %e %t %dms\n", ii, j, arr[idx].err, arr[idx].verify, arr[idx].duration.Milliseconds())
					continue
				}

				body, err := io.ReadAll(response.Body)
				_ = response.Body.Close()
				if err != nil {
					arr[idx].err = err
					log.Printf("%d %d >> %e %t %dms\n", ii, j, arr[idx].err, arr[idx].verify, arr[idx].duration.Milliseconds())
					continue
				}

				arr[idx].verify = (len(*verifyBody) == 0) || (strings.Compare(strings.TrimSpace(string(body)), strings.TrimSpace(*verifyBody)) == 0)
				arr[idx].duration = time.Since(start)

				if !arr[idx].verify {
					log.Printf("%s\n", body)
				}

				log.Printf("%d %d >> %e %t %dms\n", ii, j, arr[idx].err, arr[idx].verify, arr[idx].duration.Milliseconds())

				time.Sleep(time.Duration(*delay) * time.Millisecond)
			}
			wg.Done()
		}(i)
	}

	wg.Wait()

	page := components.NewPage()
	dc := barChart(clients, iterations, arr)
	dc.Overlap(durationChart(clients, iterations, arr))
	page.AddCharts(dc)
	if 0 < len(*assetHost) {
		page.AssetsHost = *assetHost
	}

	f, err := os.Create(*out)
	if err != nil {
		panic(err)
	}

	err = page.Render(io.MultiWriter(f))
	if err != nil {
		panic(err)
	}
}

func barChart(clients *int, iterations *int, arr []WebResult) *charts.Bar {
	var durationMax int64 = 0

	for i := 0; i < (*clients)*(*iterations); i++ {
		if (arr[i].err == nil) && (durationMax < arr[i].duration.Milliseconds()) {
			durationMax = arr[i].duration.Milliseconds()
		}
	}

	chart := charts.NewBar()

	chart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			BackgroundColor: "white",
		}),
		charts.WithTitleOpts(opts.Title{
			Title: fmt.Sprintf("Clients: %d, Iterations: %d", *clients, *iterations),
		}),
	)

	chart.ExtendYAxis(
		opts.YAxis{
			Position: "right",
			Max:      *clients,
			Show:     opts.Bool(false),
		},
		opts.YAxis{
			Name:     "Response Time(ms)",
			Max:      durationMax,
			Position: "left",
		},
	)

	axis := make([]int, *iterations)
	for j := 0; j < *iterations; j++ {
		axis[j] = j
	}
	chart.SetXAxis(axis)

	failList := make([]opts.BarData, *iterations)
	errList := make([]opts.BarData, *iterations)
	for j := 0; j < *iterations; j++ {
		var fail int64 = 0
		var err int64 = 0
		for i := 0; i < *clients; i++ {
			if !arr[i*(*iterations)+j].verify {
				fail = fail + 1
			}
			if arr[i*(*iterations)+j].err != nil {
				err = err + 1
			}
		}
		failList[j].Value = []int64{int64(j), fail}
		failList[j].ItemStyle = &opts.ItemStyle{Color: "#FAC858FF"}
		errList[j].Value = []int64{int64(j), err}
		errList[j].ItemStyle = &opts.ItemStyle{Color: "#EE6666FF"}
	}

	chart.AddSeries("", errList,
		charts.WithBarChartOpts(opts.BarChart{
			YAxisIndex: 1,
			Stack:      "total",
		}),
	)

	chart.AddSeries("", failList,
		charts.WithBarChartOpts(opts.BarChart{
			YAxisIndex: 1,
			Stack:      "total",
		}),
	)

	return chart
}

func durationChart(clients *int, iterations *int, arr []WebResult) *charts.Line {
	chart := charts.NewLine()
	chart.BackgroundColor = "white"

	axis := make([]int, *iterations)
	for j := 0; j < *iterations; j++ {
		axis[j] = j
	}
	chart.SetXAxis(axis)

	for i := 0; i < *clients; i++ {
		dataList := make([]opts.LineData, *iterations)
		for j := 0; j < *iterations; j++ {
			dataList[j].Value = []int64{int64(j), arr[i*(*iterations)+j].duration.Milliseconds()}
		}
		chart.AddSeries("", dataList,
			charts.WithLineChartOpts(opts.LineChart{
				YAxisIndex: 2,
				Smooth:     opts.Bool(true),
				ShowSymbol: opts.Bool(false),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Width: 0.3,
			}),
		)
	}

	avgList := make([]opts.LineData, *iterations)
	for j := 0; j < *iterations; j++ {
		var sum int64 = 0
		var div = 0
		for i := 0; i < *clients; i++ {
			if arr[i*(*iterations)+j].verify {
				sum = sum + arr[i*(*iterations)+j].duration.Milliseconds()
				div = div + 1
			}
		}
		if div == 0 {
			avgList[j].Value = []float64{float64(j), 0}
		} else {
			avgList[j].Value = []float64{float64(j), float64(sum) / float64(div)}
		}
	}

	chart.AddSeries("", avgList,
		charts.WithLineChartOpts(opts.LineChart{
			YAxisIndex: 2,
			Smooth:     opts.Bool(true),
			ShowSymbol: opts.Bool(false),
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Width: 2,
		}),
	)

	return chart
}
