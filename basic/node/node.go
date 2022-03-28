package node

import (
	"github.com/glimmerzcy/bccp/basic/server"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

type Node struct {
	ID string
	*log.Logger
	server.Sender
	Operations map[string]RouteFunc
}

// Factory TODO: create node with reflection
type Factory struct {
	Name string
}

type RouteFunc = func(http.ResponseWriter, *http.Request)

func NewNode(id string, sender server.Sender) *Node {
	node := &Node{
		ID:         id,
		Logger:     nil,
		Sender:     sender,
		Operations: make(map[string]RouteFunc),
	}

	// log init
	logPath := "log"
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		os.Mkdir(logPath, os.ModeDir)
	} else if err != nil {
		panic(err)
	}
	logName := time.Now().Format("20060102_150405.log")
	logFile, err := os.OpenFile(path.Join(logPath, logName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		panic(err)
	}
	node.Logger = log.New(logFile, node.ID+" ", log.Lshortfile|log.Ldate|log.Ltime|log.Lmicroseconds)

	return node
}

func (node *Node) DoOperation(operation string, writer http.ResponseWriter, request *http.Request) {
	node.Println("do", operation)
	node.Operations[operation](writer, request)
}
