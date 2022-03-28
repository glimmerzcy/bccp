package expriment

import (
	"encoding/json"
	"fmt"
	"github.com/glimmerzcy/bccp/basic/center"
	util "github.com/glimmerzcy/bccp/basic/log"
	"github.com/glimmerzcy/bccp/basic/parse"
	"github.com/glimmerzcy/bccp/basic/server"
	_ "github.com/glimmerzcy/bccp/basic/server"
	"github.com/glimmerzcy/bccp/implement/pbft"
	"github.com/wcharczuk/go-chart"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

var RouteTable = make(map[string]string)
var NodeNum int
var SleepTime time.Duration

//106.3.97.70
//106.3.97.36
//106.3.97.28
//106.3.97.67
//106.3.97.45
//106.3.97.120
//106.3.97.212
var serverList = []string{
	"106.3.97.70:1000",
	"106.3.97.36:1000",
	"106.3.97.28:1000",
	"106.3.97.67:1000",
	"106.3.97.45:1000",
	"106.3.97.120:1000",
	"106.3.97.212:1000",
	"localhost:1000",
}

//curl "http://localhost:1000/server?operation=stop"
//ps -aux | grep main

const prefix = "http://"
const suffix = ":1000"

func AddNode() {
	NodeNum++
	nodeId := parse.ID2name(NodeNum)
	RouteTable[nodeId] = parse.ID2url(NodeNum)
	http.Get("http://localhost:1000/server?operation=new&id=" + nodeId)
	http.Get("http://localhost:1000/server?operation=add&id=" + nodeId + "&msg=localhost:1000")
	for id := range RouteTable {
		http.Get("http://localhost:1000/node?from=center&operation=add&to=" + id)
	}
	server.Send("center", nodeId, "setF", pbft.SetFMsg{Total: NodeNum})
}

func Test(nodes int, times int) {
	rand.Seed(998244353)
	util.LogInit()

	server.SetFactory(pbft.Factory{Name: "pbft"})
	center.SetServerList(serverList)
	fmt.Println("jj")
	center.Broadcast("stop", "", "")

	//center.Send(1, "add", "node-5", "")
	//
	//center.GetServer().Send("center", "node-2", "client", nil)

}

func TestServer(nodes int, times int) {
	rand.Seed(998244353)
	util.LogInit()

	server.SetFactory(pbft.Factory{Name: "pbft"})

	NodeNum = 0
	for NodeNum < 3 {
		AddNode()
	}
	data := make([][]int, nodes+1)
	avg := make([]int, nodes+1)
	//data := make([]int64, 0, 10)
	for i := 0; i <= nodes; i++ {
		data[i] = make([]int, times)
		//SleepTime += time.Second
		if i <= NodeNum {
			continue
		}
		AddNode()
		for j := 0; j < times; j++ {
			nodeId := "node-" + strconv.Itoa(rand.Intn(NodeNum)+1)
			resp, err := server.Send("center", nodeId, "client", pbft.ClientMsg{Operation: "Test"})

			var msg pbft.ClientMsg
			err2 := json.NewDecoder(resp.Body).Decode(&msg)
			delay := msg.Delay
			if err != nil {
				panic(err)
			}
			if err2 != nil {
				panic(err2)
			}
			data[i][j] = int(delay)
			time.Sleep(time.Second)
			//time.Sleep(SleepTime)
		}
		avg[i] = ArrayAverage(data[i])
		fmt.Println(i, avg[i], data[i])
		log.Println(i, avg[i], data[i])
	}
	draw(avg)
}

func draw(data []int) {
	xv, yv := make([]float64, len(data)), make([]float64, len(data))
	for x, y := range data {
		xv[x] = float64(x)
		yv[x] = float64(y)
	}
	xticks := make([]chart.Tick, 0)
	for _, val := range xv {
		xticks = append(xticks, chart.Tick{Value: val, Label: fmt.Sprintf("%d", int(val))})
	}
	graph := chart.Chart{
		XAxis: chart.XAxis{
			ValueFormatter: func(v interface{}) string {
				typed := int(v.(float64))
				return fmt.Sprintf("%d", typed)
			},
			Ticks: xticks,
		},
		Series: []chart.Series{
			chart.ContinuousSeries{
				XValues: xv,
				YValues: yv,
			},
		},
	}
	f, _ := os.Create(time.Now().Format("20060102_150405.png"))
	defer f.Close()
	_ = graph.Render(chart.PNG, f)
}

func ArrayAverage(data []int) int {
	//sum := float64(ArraySum(data) - ArrayMin(data) - ArrayMax(data))
	sum := float64(ArraySum(data))
	//num := float64(len(data) - 2)
	num := float64(len(data))
	return int(sum / num)
}

func ArraySum(data []int) int {
	ans := 0
	for _, d := range data {
		ans += d
	}
	return ans
}

func ArrayMin(data []int) int {
	ans := math.MaxInt
	for _, d := range data {
		if d < ans {
			ans = d
		}
	}
	return ans
}

func ArrayMax(data []int) int {
	ans := math.MinInt
	for _, d := range data {
		if d > ans {
			ans = d
		}
	}
	return ans
}
