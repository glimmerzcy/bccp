package log

import (
	"log"
	"os"
	"path"
	"time"
)

func LogInit() {
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
	log.SetOutput(logFile)

}

func LogMsg(msg interface{}) {
	log.Printf("%+v\n", msg)
	//switch msg := msg.(type) {
	//case *node.RequestMsg:
	//	reqMsg := msg
	//	log.Printf("[REQUEST] ClientID: %s, Timestamp: %d, Operation: %s\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Operation)
	//case *node.PrePrepareMsg:
	//	prePrepareMsg := msg
	//	log.Printf("[PREPREPARE] ClientID: %s, Operation: %s, SequenceID: %d\n", prePrepareMsg.RequestMsg.ClientID, prePrepareMsg.RequestMsg.Operation, prePrepareMsg.SequenceID)
	//case *node.VoteMsg:
	//	voteMsg := msg
	//	if voteMsg.MsgType == node.PrepareMsg {
	//		log.Printf("[PREPARE] NodeID: %s\n", voteMsg.NodeID)
	//	} else if voteMsg.MsgType == node.CommitMsg {
	//		log.Printf("[COMMIT] NodeID: %s\n", voteMsg.NodeID)
	//	}
	//default:
	//	log.Println(msg)
	//}
}

func LogStage(stage string, isDone bool) {
	if isDone {
		log.Printf("[STAGE-DONE] %s\n", stage)
	} else {
		log.Printf("[STAGE-BEGIN] %s\n", stage)
	}
}
