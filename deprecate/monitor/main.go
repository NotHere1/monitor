package main

import (
	tm "github.com/buger/goterm"
	"github.com/NotHere1/monitor/utils"
	"container/list"
	"time"
	"fmt"
	"regexp"
	"os"
	"bufio"
	"strings"
	"flag"
)

type CommonLog struct {
	host string
	rfc931 string
	username string
	datetimezone string
	requestMethod string
	requestResource string
	requestProtocol string
	statusCode string
	bytes string
}

func aggregateLogs(logs <-chan CommonLog, results chan<- [3]map[string]int, timeout time.Duration, bufferSize int) {

	buffer := 0
	sections := make(map[string]int)
	statusCodes := make(map[string]int)
	httpMethods := make(map[string]int)
	var log CommonLog
	ok := true

	// collect logs line by line
	// accumulate them into map[string]int for statistics
	// flush data to summarizeLogs when either timed out or buffer max reached
	for ok {
		select {
			case <- time.After(timeout): // reached timeout -> flush
				results <- [3]map[string]int{sections,statusCodes,httpMethods}
				// reset
				buffer = 0
				sections = make(map[string]int)
				statusCodes = make(map[string]int)
				httpMethods = make(map[string]int)
			case log, ok = <- logs:
				// accumulate each line into groups
				buffer += 1
				sections[utils.ParseSection(log.requestResource)] += 1
				statusCodes[log.statusCode] += 1
				httpMethods[log.requestMethod] += 1

				// reached buffer capacity -> flush
				if buffer == bufferSize {
					results <- [3]map[string]int{sections,statusCodes,httpMethods}
					// reset
					buffer = 0
					sections = make(map[string]int)
					statusCodes = make(map[string]int)
					httpMethods = make(map[string]int)					
				}
		}
	}
}

func summarizeAggregatedLogs(aggregates <-chan [3]map[string]int, alerts <-chan string, throughputs <-chan int, refreshScreenTick time.Ticker, refreshScreenSec int) {

	ok := true
	accuSections := make(map[string]int)
	accuStatusCodes := make(map[string]int)
	accuHttpMethods := make(map[string]int)
	accuThroughputs := 0
	var aggregate [3]map[string]int
	var alertStrBuilder strings.Builder // unbounded. risky. but make persisting alerts easy.
	var strBuilder strings.Builder
	var alert string
	var throughput int

	for ok {
		select {
			case <- refreshScreenTick.C:
				// when ticked...print updated result to stdout
				// summarize aggregated results
				strBuilder = utils.BuildThroughputSummary(strBuilder, accuThroughputs, refreshScreenSec)
				strBuilder = utils.BuildRequestMethodSummary(strBuilder, accuHttpMethods)
				strBuilder = utils.BuildStatusCodeSummary(strBuilder, accuStatusCodes)
				strBuilder = utils.BuildSectionSummary(strBuilder, accuSections)

				// open source library that makes refreshing stdout screen easy
				tm.Clear()
				tm.MoveCursor(1,1)
				tm.Println(alertStrBuilder.String()) // print all historic alerts
				tm.Println(strBuilder.String())      // print summary since last screen update
				tm.Flush()

				// reset
				strBuilder.Reset()
				accuSections = make(map[string]int)
				accuStatusCodes = make(map[string]int)
				accuHttpMethods = make(map[string]int)
				accuThroughputs = 0

			case alert, ok = <- alerts:
				// receive throughput alerts
				alertStrBuilder.WriteString(alert)

			case throughput, ok = <- throughputs:
				// receive throughput counts
				accuThroughputs += throughput

			case aggregate, ok = <- aggregates:
				// receive aggregated logs
				// accumulate them for the next screen update 
				utils.AccuMap(aggregate[0], accuSections)
				utils.AccuMap(aggregate[1], accuStatusCodes)
				utils.AccuMap(aggregate[2], accuHttpMethods)
		}
	}
}

func checkThroughput(requests <-chan int, throughputs chan<- int, alerts chan<- string, alertDurationSecond int, alertThreshold int) {

	// use a linked list as bucket to hold each second request count
	// at each tick, deque oldest, and enque newest count
	// to preserve window len
	queue := list.New()
	tick := time.NewTicker(1 * time.Second) 
	ok := true
	count := 0
	running_total := 0
	alertMode := false
	var lastAlertTime time.Time
	var request int

	for ok {
		select {
			case <- tick.C: // tick once per second

				// add # of requests received past second
				queue.PushBack(count) 
				running_total += count

				// if reached capacity, remove oldest node
				// remove oldest count value from running total 
				if queue.Len() > alertDurationSecond {
					f := queue.Front()
					v, _ := f.Value.(int)
					running_total -= v
					queue.Remove(f)
				}

				// if running_total exceed threshold, print alert
				if !alertMode && running_total > alertThreshold {
					alertMode = true
					lastAlertTime = time.Now()
					alerts <- fmt.Sprintf("High traffic generated an alert - hits = %d (threshold = %d, window_period = %d sec), triggered at %s\n", running_total, alertThreshold, alertDurationSecond, lastAlertTime.Format(time.RFC3339))
				} else if alertMode && running_total < alertThreshold {
					alertMode = false
					alerts <- fmt.Sprintf("High traffic alert resolved. Begin = %s, End = %s\n", lastAlertTime.Format(time.RFC3339), time.Now().Format(time.RFC3339))
				}

				// send count data to summarizeAggregatedLogs
				throughputs <- count

				// reset
				count = 0

			case request, ok = <- requests:
				count += request
		}
	}
}

func main() {

	var alertThreshold int
	var alertWindowSecond int
	var screenRefreshSecond int
	flag.IntVar(&alertThreshold, "threshold", 50, "Hits alert threshold")
	flag.IntVar(&alertWindowSecond,"window", 15, "Access log retention window second for alert threshold")
	flag.IntVar(&screenRefreshSecond, "refresh", 10, "Screen refresh interval second")
	flag.Parse()

	aggregateTimeoutSecond := 5
	alertDurationSecond := 15
	aggregateTimeout := time.Duration(aggregateTimeoutSecond) * time.Second
	aggregateBuffer := 10

	if screenRefreshSecond <= aggregateTimeoutSecond {
		fmt.Fprintln(os.Stderr, "error: screenRefreshSecond cannot be lower than ", aggregateTimeoutSecond)
		os.Exit(1)
	}

	// init channels
	logs := make(chan CommonLog)
	logResults := make(chan [3]map[string]int)
	requestCounts := make(chan int)
	alerts := make(chan string)
	throughputs := make(chan int)

	// set stdout screen reset time in second
	refreshTick := time.NewTicker(time.Duration(screenRefreshSecond) * time.Second)
	reg := regexp.MustCompile(`^(\S+) (\S+) (\S+) \[(.*)\] "(.*)" (\d{3}) (\d{1,})`)

	// execute coroutines
	go aggregateLogs(logs, logResults, aggregateTimeout, aggregateBuffer)
	go summarizeAggregatedLogs(logResults, alerts, throughputs, *refreshTick, screenRefreshSecond)
	go checkThroughput(requestCounts, throughputs, alerts, alertDurationSecond, alertThreshold)

	// listen to stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		requestCounts <- 1
		m := reg.FindStringSubmatch(line)
		if len(m) == 8 {
			req := strings.Fields(m[5])
			cl := CommonLog{m[1],m[2],m[3],m[4],req[0],req[1],req[2],m[6],m[7]}
			logs <- cl
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
} 
