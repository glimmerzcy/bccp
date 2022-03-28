package center

import (
	"fmt"
	"github.com/glimmerzcy/bccp/basic/server"
	"log"
	"net/http"
	"net/url"
)

type Center struct {
	Server     *server.Server
	ServerList []string
}

func NewCenter(server *server.Server) *Center {
	return &Center{
		server,
		make([]string, 0),
	}
}

var DefaultCenter *Center

func init() {
	DefaultCenter = NewCenter(server.DefaultServer)
}

func (center *Center) Send(to int, operation string, id string, msg string) (resp *http.Response, err error) {
	query := url.Values{}
	query.Add("id", id)
	query.Add("msg", msg)
	query.Add("operation", operation)
	queryUrl := "http://" + center.ServerList[to] + "/server?" + query.Encode()
	return http.Get(queryUrl)
}

func (center *Center) Broadcast(operation string, id string, msg string) (resps []*http.Response, errs []error) {
	total := len(center.ServerList)
	countChan := make(chan int)
	resps, errs = make([]*http.Response, 0, total), make([]error, 0, total)
	for to := range center.ServerList {
		go func(to int) {
			resp, err := center.Send(to, operation, id, msg)
			resps = append(resps, resp)
			errs = append(errs, err)
			countChan <- 1
		}(to)
	}
	for i := 0; i < total; i++ {
		select {
		case <-countChan:
			fmt.Println(i)
		}
	}
	log.Println("operation broadcast finished!")
	return resps, errs
}

func Send(to int, operation string, id string, msg string) (resp *http.Response, err error) {
	return DefaultCenter.Send(to, operation, id, msg)
}

func Broadcast(operation string, id string, msg string) {
	DefaultCenter.Broadcast(operation, id, msg)
}

func SetServerList(serverList []string) {
	DefaultCenter.ServerList = serverList
}

func GetServer() *server.Server {
	return DefaultCenter.Server
}
