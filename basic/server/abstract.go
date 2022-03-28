package server

import "net/http"

type Sender interface {
	Send(from string, to string, operation string, message interface{}) (resp *http.Response, err error)
	Broadcast(from string, operation string, message interface{}) (resps []*http.Response, errs []error)
}

type Operator interface {
	DoOperation(operation string, writer http.ResponseWriter, request *http.Request)
}

type Factory interface {
	NewOperator(id string, sender Sender) Operator
}
