package main

import (
	util "github.com/glimmerzcy/bccp/basic/log"
	"github.com/glimmerzcy/bccp/basic/server"
	"github.com/glimmerzcy/bccp/implement/pbft"
)

func main() {
	util.LogInit()
	server.SetFactory(pbft.Factory{Name: "pbft"})
	server.Wait()
}
