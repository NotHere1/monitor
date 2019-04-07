package utils

import str "strings"
import "fmt"
import "sort"
import "strconv"
import "time"
import "os"

var sprint = fmt.Sprint
var sprintf = fmt.Sprintf

type kv struct {
    Key   string
    Value int
}

func ParseEpochString(epoch string) (time.Time, error) {
	i, err := strconv.ParseInt(epoch, 10, 64)
    if err != nil {
        return time.Now(), err
    }
    tm := time.Unix(i, 0)
    return tm, nil
}

func AccuMap(mp map[string]int, accu map[string]int) map[string]int {
	for k, v := range mp {
		accu[k] += v
	}
	return accu
}

func Check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func PrintMsgExit(err string) {
	fmt.Fprintln(os.Stderr, err)	
	os.Exit(1)
}

func ValidateFilePath(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		PrintMsgExit(fmt.Sprintf("file not found: %s", err))
	}
}

func ParseSection(resource string) string {
	split := str.Split(str.TrimPrefix(resource, "/"), "/")
	first := split[0]
	if len(split) > 1 {
		first = split[1]
	} else if len(split) == 1 {
		if first == "" {
			first = "/"
		}
	}
	return first
}

func SortMap(summary map[string]int) []kv {
	var arr []kv
    for k, v := range summary {
        arr = append(arr, kv{k, v})
    }

    sort.Slice(arr, func(i, j int) bool {
        return arr[i].Value > arr[j].Value
    })

    return arr
}

func BuildWindowStat(sb str.Builder, https map[string]int, statusCodes map[string]int, sections map[string]int, windowLen int, start int, end int) str.Builder {
	sb = BuildTitle(sb)
	sb = BuildThroughputSummary(sb, windowLen, start, end)
	sb = BuildRequestMethodSummary(sb, https)
	sb = BuildStatusCodeSummary(sb, statusCodes)
	sb = BuildSectionSummary(sb, sections)
	return sb
}

func BuildTitle(sb str.Builder) str.Builder {
	sb.WriteString(fmt.Sprintf("[10 seconds window stats]"))
	sb.WriteString("\n")
	return sb
}

func BuildThroughputSummary(sb str.Builder, accuThroughputs int, startSecond int, endSecond int) str.Builder {
	elapseSeconds := endSecond - startSecond
	throughput := float64(accuThroughputs / elapseSeconds)
	if elapseSeconds > 0 {
		sb.WriteString(fmt.Sprintf("[start: %d, end %d, elapse: %d, throughputs: %.2f req/s]", startSecond, endSecond, elapseSeconds, throughput))
		sb.WriteString("\n")
	}
	return sb
}

func BuildStatusCodeSummary(sb str.Builder, statusCodes map[string]int) str.Builder {
	all_status := map[string]int{"2XX": 0,"3XX": 0,"4XX": 0,"5XX": 0}
	for k, v := range statusCodes {
		all_status[sprint(k[:1], "XX")] += v 
	}
	for i := 2; i <= 5; i++ {
		s := sprint(i, "XX")
		sb.WriteString(sprintf("%-8s%-6d", s, all_status[s]))
	}
	sb.WriteString("\n")
	return sb
}

func BuildRequestMethodSummary(sb str.Builder, requestMethods map[string]int) str.Builder {
	all_methods := [8]string{"GET","HEAD","POST","PUT","DELETE","CONNECT","OPTIONS","TRACE"}
	for _, v := range all_methods {
		_, prs := requestMethods[v]
		if !prs {
			requestMethods[v] = 0
		}
	}
	for _, v := range all_methods {
		sb.WriteString(sprintf("%-8s%-6d", v, requestMethods[v]))
	}
	sb.WriteString("\n")
	return sb
}

func BuildSectionSummary(sb str.Builder, sections map[string]int) str.Builder {
	sortedSections := SortMap(sections)
	sb.WriteString("\n")
	sb.WriteString(sprintf("%-5s%-10s\n", "reqs", "section"))
	sb.WriteString("\n")
	for idx, obj := range sortedSections {
		if idx == 3 {
			break
		}
		sb.WriteString(sprintf("%-5d%-10s\n", obj.Value, obj.Key))
	}
	sb.WriteString("\n")
	return sb
}

// func main() {

// 	var sb str.Builder
// 	statusCodes := map[string]int{"400": 200, "201": 3817, "244": 101833, "301": 5}
// 	requestMethods := map[string]int{"GET": 2, "POST": 3, "HEAD": 1, "DELETE": 5}
// 	sections := map[string]int{"img": 1810, "page": 41828, "/": 20, "yuna": 31}
// 	sb = buildRequestCountSummary(sb, 1244505, 10)
// 	sb = buildRequestMethodSummary(sb, requestMethods)
// 	sb = buildStatusCodeSummary(sb, statusCodes)
// 	sb = buildSectionSummary(sb, sections)

// 	fmt.Println(sb.String())
// }