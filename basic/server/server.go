package server

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
)

type Server struct {
	// id to operator, contains all operators managed by this Server
	OperatorTable map[string]Operator
	// id to url, contains all operators in the network
	RouteTable map[string]string
	// inject to it when use
	Factory
	status chan int
}

func NewServer() *Server {
	return &Server{
		make(map[string]Operator),
		make(map[string]string),
		nil,
		make(chan int),
	}
}

var DefaultServer *Server

// TODO: set port with args and flag.
// e.g. port=1000
func init() {
	DefaultServer = NewServer()
	DefaultServer.Start(":1000")
}

func (server *Server) Start(addr string) {
	log.Println("Server start")

	mux := http.NewServeMux()
	mux.HandleFunc("/server", server.HandleServer)
	mux.HandleFunc("/node", server.HandleNode)

	go func() {
		err := http.ListenAndServe(addr, mux)
		if err != nil {
			panic(err)
		}
	}()
}

func (server *Server) Wait() {
	select {
	case status := <-server.status:
		log.Println("Server stop with status:", status)
	}
}

func (server *Server) HandleServer(_ http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	log.Println(query)
	operation := query.Get("operation")
	id := query.Get("id")
	msg := query.Get("msg")
	switch operation {
	case "new":
		server.OperatorTable[id] = server.NewOperator(id, server)
	case "delete":
		delete(server.OperatorTable, id)
	case "add":
		server.RouteTable[id] = msg
	case "stop":
		server.status <- 0
	}
}

// HandleNode TODO: find handler by reflection
func (server *Server) HandleNode(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	log.Println(query)
	operation := query.Get("operation")
	id := query.Get("to")
	server.OperatorTable[id].DoOperation(operation, writer, request)
}

func (server *Server) Send(from string, to string, operation string, message interface{}) (resp *http.Response, err error) {
	query := url.Values{}
	query.Add("from", from)
	query.Add("to", to)
	query.Add("operation", operation)
	queryUrl := "http://" + server.RouteTable[to] + "/node?" + query.Encode()
	jsonMessage, _ := json.Marshal(message)
	buff := bytes.NewBuffer(jsonMessage)
	return http.Post(queryUrl, "application/json", buff)
}

func (server *Server) Broadcast(from string, operation string, message interface{}) (resps []*http.Response, errs []error) {
	total := len(server.RouteTable)
	countChan := make(chan int)
	resps, errs = make([]*http.Response, 0, total), make([]error, 0, total)
	for id := range server.RouteTable {
		if id == from {
			continue
		}
		go func(id string) {
			resp, err := server.Send(from, id, operation, message)
			resps = append(resps, resp)
			errs = append(errs, err)
			countChan <- 1
		}(id)
	}
	for i := 0; i < total; i++ {
		select {
		case <-countChan:
		}
	}
	log.Println(from, "operation broadcast finished!")
	return resps, errs
}

func SetFactory(factory Factory) {
	DefaultServer.Factory = factory
}

func Send(from string, to string, operation string, message interface{}) (resp *http.Response, err error) {
	return DefaultServer.Send(from, to, operation, message)
}

func Broadcast(from string, operation string, message interface{}) (resps []*http.Response, errs []error) {
	return DefaultServer.Broadcast(from, operation, message)
}

func Wait() {
	DefaultServer.Wait()
}
