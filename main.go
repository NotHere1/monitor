package main

import (
	// tm "github.com/buger/goterm"
	"github.com/NotHere1/monitor/utils"
	"container/list"
	"time"
	"fmt"
	"os"
	"bufio"
	"strings"
	"flag"
	"strconv"
	"math"
)

type CommonLog struct {
	host string
	rfc931 string
	username string
	epoch string
	requestMethod string
	requestResource string
	requestProtocol string
	statusCode string
	bytes string
}

type AggregateLogs struct {
	logs []CommonLog
	startSec int
	endSec int
}

func aggregateLogs(logs <-chan CommonLog, requests chan<- AggregateLogs, results chan<- AggregateLogs, timeout time.Duration) {

	var log CommonLog
	accuSections := make(map[string]int)
	accuStatusCodes := make(map[string]int)
	accuHttpMethods := make(map[string]int)
	accuLogs := make([]CommonLog, 0)
	curMin := 999999999999
	curMax := -1
	ok := true

	// collect logs line by line
	// flush data to summarizeLogs when either timed out or buffer max reached
	for ok {
		select {
			case <- time.After(timeout): // reached timeout -> flush
				results <- AggregateLogs{accuLogs, curMin, curMax}

				// reset
				accuLogs = make([]CommonLog, 0)
				curMin = 999999999999
				curMax = -1

			case log, ok = <- logs:

				// find 10 seconds window
				epoch, _ := strconv.Atoi(log.epoch)
				if epoch > curMax {
					curMax = epoch
				}
				if epoch < curMin {
					curMin = epoch
				}
				if curMax - curMin < 10 {
					accuLogs = append(accuLogs, log)
				} else {
					
					// flush
					// fmt.Println("ten seconds", curMax - curMin)
					for _, log := range accuLogs {
						accuSections[utils.ParseSection(log.requestResource)] += 1
						accuStatusCodes[log.statusCode] += 1
						accuHttpMethods[log.requestMethod] += 1
					}
					results <- AggregateLogs{accuLogs, curMin, curMax}
					requests <- AggregateLogs{accuLogs, curMin, curMax}

					// reset
					accuLogs = make([]CommonLog, 0)
					curMin = 999999999999
					curMax = -1
				}
		}
	}
}

func summarizeAggregatedLogs(aggregates <-chan AggregateLogs, alerts <-chan string, alertDurationSecond int, alertThreshold int) {

	ok := true 
	accuSections := make(map[string]int)
	accuStatusCodes := make(map[string]int)
	accuHttpMethods := make(map[string]int)
	var alert string
	var aggregate AggregateLogs
	var alertStrBuilder strings.Builder // unbounded. risky. but make persisting alerts easy.
	var strBuilder strings.Builder

	alertStrBuilder.WriteString(fmt.Sprintf("[High Threshold Alerts - total traffic > %d for past %d second]\n", alertThreshold, alertDurationSecond))

	for ok {
		select {
		
		case aggregate, ok = <- aggregates:

			logs := aggregate.logs
			startSecond := aggregate.startSec
			endSecond := aggregate.endSec

			for _, log := range logs {
				// accumulate all required attributes
				accuSections[utils.ParseSection(log.requestResource)] += 1
				accuStatusCodes[log.statusCode] += 1
				accuHttpMethods[log.requestMethod] += 1
			}

			// build attributes statistics
			strBuilder = utils.BuildWindowStat(strBuilder, accuHttpMethods, accuStatusCodes, accuSections, len(logs), startSecond, endSecond)
	
			fmt.Println(alertStrBuilder.String())
			fmt.Println(strBuilder.String())

			// reset
			strBuilder.Reset()
			accuSections = make(map[string]int)
			accuStatusCodes = make(map[string]int)
			accuHttpMethods = make(map[string]int)

		case alert, ok = <- alerts:
			alertStrBuilder.WriteString(alert)
		}
	}
}

func checkThroughput(aggregates <-chan AggregateLogs, alerts chan<- string, alertDurationSecond int, alertThreshold int) {

	// use a linked list as bucket to hold each second request count
	// at each tick, deque oldest, and enque newest count
	// to preserve window len
	queue := list.New()
	count := 0
	running_total := 0
	alertMode := false
	var lastAlertTime int

	for aggregate := range aggregates {

		// add # of requests received past second
		count = len(aggregate.logs)
		queue.PushBack(count)
		running_total += count

		// if reached capacity, remove oldest node
		// remove oldest count value from running total 
		if queue.Len() > alertDurationSecond / 10 {
			f := queue.Front()
			v, _ := f.Value.(int)
			running_total -= v
			queue.Remove(f)
		}

		// if running_total exceed threshold, print alert
		if !alertMode && running_total > alertThreshold {
			alertMode = true
			lastAlertTime = aggregate.startSec
			alerts <- fmt.Sprintf("High traffic generated an alert - hits = %d (threshold = %d, window_period = %d sec), triggered at %d\n", running_total, alertThreshold, alertDurationSecond, lastAlertTime)
		} else if alertMode && running_total < alertThreshold {
			alertMode = false
			alerts <- fmt.Sprintf("High traffic alert resolved. Begin = %d, End = %d\n", lastAlertTime, aggregate.endSec)
		}

	}
}

func main() {


	var readSource *os.File
	var inputFilePath string
	var alertThreshold int
	var alertWindowSecond int
	flag.StringVar(&inputFilePath, "file", "os.Stdin", "input file path")
	flag.IntVar(&alertThreshold, "threshold", 100, "Hits alert threshold.")
	flag.IntVar(&alertWindowSecond,"window", 10, "Access log retention window second. Min 10 seconds - must be multiple of 10")
	flag.Parse()

	// positional arg 1 is file path
	// if no positional arg is set
	// then default to stdin as input source
	fmt.Println(flag.NFlag())
	fmt.Println(flag.NArg())
	args := os.Args[1:]
	fmt.Println(args)

	if inputFilePath == "os.Stdin" { 
		readSource = os.Stdin
	} else {
		utils.ValidateFilePath(inputFilePath)
		f, err := os.Open(inputFilePath)
		utils.Check(err)
		readSource = f
		defer f.Close()
	}
	aggregateTimeout := 5 * time.Second

	if math.Mod(float64(alertWindowSecond), 10) != 0 {
		utils.PrintMsgExit("error: alertWindowSecond must be multiple of 10")
	}

	// init channels
	logs := make(chan CommonLog)
	logResults := make(chan AggregateLogs)
	requests := make(chan AggregateLogs)
	alerts := make(chan string)

	// execute coroutines
	go aggregateLogs(logs, requests, logResults, aggregateTimeout)
	go summarizeAggregatedLogs(logResults, alerts, alertWindowSecond, alertThreshold)
	go checkThroughput(requests, alerts, alertWindowSecond, alertThreshold)

	// listen to stdin
	scanner := bufio.NewScanner(readSource)
	for scanner.Scan() {
		line := strings.ReplaceAll(scanner.Text(), "\"", "")
		// requestCounts <- 1
		m := strings.Split(line, ",")
		if len(m) == 7 {
			req := strings.Fields(m[4])
			if len(req) == 3 {
				cl := CommonLog{m[0],m[1],m[2],m[3],req[0],req[1],req[2],m[5],m[6]}
				logs <- cl
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
} 
