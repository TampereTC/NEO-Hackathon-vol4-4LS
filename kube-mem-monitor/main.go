package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"os"
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

func drawChart(res http.ResponseWriter, req *http.Request) {
	buffers := getMetric("sum(node_memory_Buffers)")
	free := getMetric("sum(node_memory_MemFree)")
	cached := getMetric("sum(node_memory_Cached)")
	pie := chart.PieChart{
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{Value: buffers, Label: "Buffers " + toMib(buffers)},
			{Value: free, Label: "Free " + toMib(free)},
			{Value: cached, Label: "Cached " + toMib(cached)},
		},
	}

	res.Header().Set("Content-Type", "image/png")
	err := pie.Render(chart.PNG, res)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Error rendering pie chart: %v\n", err)
	}
}

func toMib(x float64) string {
	return fmt.Sprintf("%d", int(x)/1024/1024) + " MiB"
}

func getMetric(metric string) float64 {
	q := "http://10.105.128.93:9090/api/v1/query?query=" + metric
	fmt.Fprintln(os.Stderr,q)
	res, _ := http.Get(q)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	fmt.Fprintln(os.Stderr,string(body))

	var mmetric memMetric
	err := json.Unmarshal(body, &mmetric)
	if err != nil {
		fmt.Fprintln(os.Stderr,"error:", err)
	}
	value := mmetric.Data.Result[0].Value[1].(string)
	fmt.Fprintf(os.Stderr,"%v:%v\n", metric, value)
	numval, _ := strconv.ParseFloat(value, 64)
	return numval
}

func main() {
	http.HandleFunc("/", drawChart)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
