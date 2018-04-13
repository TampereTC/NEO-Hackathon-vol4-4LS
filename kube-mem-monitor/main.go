package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	chart "github.com/wcharczuk/go-chart"
)

type memMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
			} `json:"metric"`
			Value []interface{} `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type countMetric struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Name     string `json:"__name__"`
				Instance string `json:"instance"`
				Job      string `json:"job"`
			} `json:"metric"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func drawMem(res http.ResponseWriter, req *http.Request) {
	buffers := getMetric("sum(node_memory_Buffers)")
	free := getMetric("sum(node_memory_MemFree)")
	cached := getMetric("sum(node_memory_Cached)")
	used := getMetric("sum(node_memory_MemTotal)-sum(node_memory_MemFree)-sum(node_memory_Buffers)-sum(node_memory_Cached)")
	pie := chart.PieChart{
		Width:      400,
		Height:     400,
		Title:      "RAM usage",
		TitleStyle: chart.StyleShow(),
		Values: []chart.Value{
			{Value: used, Label: "Used " + toMib(used)},
			{Value: buffers, Label: "Buffers " + toMib(buffers)},
			{Value: free, Label: "Free " + toMib(free)},
			{Value: cached, Label: "Cached " + toMib(cached)},
		},
	}

	res.Header().Set("Content-Type", "image/png")
	err := pie.Render(chart.PNG, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering pie chart: %v\n", err)
	}
}
func toMib(x float64) string {
	return fmt.Sprintf("%d", int(x)/1024/1024) + " MiB"
}

func drawCount(res http.ResponseWriter, req *http.Request) {
	times, counts := getMetrics("kubelet_running_pod_count", 3600000000000)
	fmt.Fprintln(os.Stderr, times)
	fmt.Fprintln(os.Stderr, counts)
	res.Header().Set("Content-Type", "image/png")

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
			ValueFormatter: chart.TimeMinuteValueFormatter,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		Height:     200,
		Width:      600,
		Title:      "Pod count",
		TitleStyle: chart.StyleShow(),
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: times,
				YValues: counts,
			},
		},
	}
	err := graph.Render(chart.PNG, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering pie chart: %v\n", err)
	}
}

func drawDiskio(res http.ResponseWriter, req *http.Request) {
	times, counts := getMetrics("sum(rate(node_disk_bytes_read[5m]))", 3600000000000)
	times2, counts2 := getMetrics("sum(rate(node_disk_bytes_written[5m]))", 3600000000000)
	fmt.Fprintln(os.Stderr, times)
	fmt.Fprintln(os.Stderr, counts)
	res.Header().Set("Content-Type", "image/png")

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
			ValueFormatter: chart.TimeMinuteValueFormatter,
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		Height:     200,
		Width:      600,
		Title:      "Disk",
		TitleStyle: chart.StyleShow(),
		Series: []chart.Series{
			chart.TimeSeries{
				XValues: times,
				YValues: counts,
			},
			chart.TimeSeries{
				XValues: times2,
				YValues: counts2,
			},
		},
	}
	err := graph.Render(chart.PNG, res)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error rendering pie chart: %v\n", err)
	}
}

func getMetric(metric string) float64 {
	q := "http://10.105.128.93:9090/api/v1/query?query=" + metric
	fmt.Fprintln(os.Stderr, q)
	res, _ := http.Get(q)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Fprintln(os.Stderr, string(body))

	var mmetric memMetric
	err := json.Unmarshal(body, &mmetric)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	value := mmetric.Data.Result[0].Value[1].(string)
	fmt.Fprintf(os.Stderr, "%v:%v\n", metric, value)
	numval, _ := strconv.ParseFloat(value, 64)
	return numval
}

func getMetrics(metric string, timeRange time.Duration) ([]time.Time, []float64) {
	fmt.Fprintln(os.Stderr, timeRange)
	timeNow := time.Now().UTC()
	startTime := timeNow.Add(-timeRange).Format(time.RFC3339)
	endTime := timeNow.Format(time.RFC3339)
	fmt.Fprintln(os.Stderr, startTime)
	fmt.Fprintln(os.Stderr, endTime)
	q := "http://192.168.99.100:31791/api/v1/query_range?query=" + metric +
		"&start=" + startTime +
		"&end=" + endTime +
		"&step=1s"
	fmt.Fprintln(os.Stderr, q)

	res, _ := http.Get(q)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Fprintln(os.Stderr, string(body))

	var cmetric countMetric
	err := json.Unmarshal(body, &cmetric)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}

	data := cmetric.Data.Result[0].Values
	fmt.Fprintf(os.Stderr, "\n%T:%v\n", data, data)
	var times []time.Time
	var values []float64
	for _, x := range data {
		fmt.Fprintln(os.Stderr, x[0])
		times = append(times, time.Unix(int64(x[0].(float64)), 0))
		fmt.Fprintf(os.Stderr, "%T:%v\n", x[1], x[1])
		numval, _ := strconv.ParseFloat(x[1].(string), 64)
		values = append(values, numval)
	}
	return times, values
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(os.Stderr, "index\n")
	fmt.Fprintln(w, "Welcome!")
}

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", index)
	router.HandleFunc("/memory", drawMem)
	router.HandleFunc("/count", drawCount)
	router.HandleFunc("/diskio", drawDiskio)
	// http.HandleFunc("/", drawMem)
	log.Fatal(http.ListenAndServe(":8080", router))
}
