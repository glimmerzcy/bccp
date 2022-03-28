package pbft

import (
	"encoding/json"
	"errors"
	"fmt"
	log2 "github.com/glimmerzcy/bccp/basic/log"
	"github.com/glimmerzcy/bccp/basic/node"
	"github.com/glimmerzcy/bccp/basic/server"
	"math"
	"net/http"
	"sync"
	"time"
)

type Node struct {
	node.Node

	View          *View
	CurrentState  *State
	CommittedMsgs []*RequestMsg // kinda block.
	MsgBuffer     *MsgBuffer
	MsgDelivery   chan interface{}

	Client *Client

	// note: you are 1 node, f does not need to add 1
	total int
	f     int // f
	ff    int // 2 * f
}

type MsgBuffer struct {
	ReqMsgs        []*RequestMsg
	PrePrepareMsgs []*PrePrepareMsg
	PrepareMsgs    []*VoteMsg
	CommitMsgs     []*VoteMsg
}

type View struct {
	ID      int64
	Primary string
}

const ResolvingTimeDuration = time.Millisecond * 10 // 1 second.

func NewNode(id string, sender server.Sender) *Node {
	const viewID = 10000000000 // temporary.
	node := &Node{
		Node: *node.NewNode(id, sender),
		// Hard-coded for test.
		View: &View{
			ID:      viewID,
			Primary: "node-1",
		},

		// Consensus-related struct
		CurrentState:  nil,
		CommittedMsgs: make([]*RequestMsg, 0),
		MsgBuffer: &MsgBuffer{
			ReqMsgs:        make([]*RequestMsg, 0),
			PrePrepareMsgs: make([]*PrePrepareMsg, 0),
			PrepareMsgs:    make([]*VoteMsg, 0),
			CommitMsgs:     make([]*VoteMsg, 0),
		},

		// Channels
		MsgDelivery: make(chan interface{}),

		Client: NewClient(),
		total:  0,
	}

	node.Operations["req"] = node.handleRequest
	node.Operations["pre-prepare"] = node.handlePrePrepare
	node.Operations["prepare"] = node.handlePrepare
	node.Operations["commit"] = node.handleCommit
	node.Operations["reply"] = node.handleReply
	node.Operations["add"] = node.handleAdd
	node.Operations["setF"] = node.handleSetF
	node.Operations["client"] = node.handleClient

	// Start alarm trigger
	go node.alarmToDispatcher()

	// Start message resolver
	go node.resolveMsg()

	return node
}

// Factory TODO: create node with reflection
type Factory struct {
	Name string
}

func (factory Factory) NewOperator(id string, sender server.Sender) server.Operator {
	return NewNode(id, sender)
}

func (node *Node) StartRequest(operation string) (int64, error) {
	err := node.Client.Start()
	if err != nil {
		return -1, err
	}
	msg := &RequestMsg{
		ClientID:  node.ID,
		Operation: operation,
		Timestamp: time.Now().Unix(),
	}
	node.Println("Start request as Client, time:", node.Client.StartTime)
	go node.Send(node.ID, node.View.Primary, "req", msg)

	select {
	case delay := <-node.Client.Msg:
		node.Printf("Request Finished! Delay: %d", delay)
		node.Println(node.Client.StartTime, time.Now().UnixMicro())
		return delay, nil
	}
}

func (node *Node) GetReply(msg *ReplyMsg) {
	node.Printf("Result: %s by %s\n", msg.Result, msg.NodeID)
	node.Client.Count++
	//fmt.Println(node.Client.Count)
	if node.Client.Count > node.f {
		node.Client.End()
	}
}

func (node *Node) Reply(msg *ReplyMsg) {
	// Print all committed messages.
	for _, value := range node.CommittedMsgs {
		node.Printf("Committed value: %s, %d, %s, %d", value.ClientID, value.Timestamp, value.Operation, value.SequenceID)
	}
	node.Println()

	// send reply msg to the Client
	go node.Send(node.ID, msg.ClientID, "reply", msg)
	node.Println("Reply Finished!")
}

// GetReq can be called when the node's CurrentState is nil.
// Consensus start procedure for the Primary.
func (node *Node) GetReq(reqMsg *RequestMsg) error {
	log2.LogMsg(reqMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}

	// Start the consensus process.
	prePrepareMsg, err := node.CurrentState.StartConsensus(reqMsg)
	if err != nil {
		return err
	}

	log2.LogStage(fmt.Sprintf("Consensus Process (ViewID:%d)", node.CurrentState.ViewID), false)

	// Send handlePrePrepare message
	if prePrepareMsg != nil {
		go node.Broadcast(node.ID, "pre-prepare", prePrepareMsg)
		log2.LogStage("Pre-prepare", true)
	}

	return nil
}

// GetPrePrepare can be called when the node's CurrentState is nil.
// Consensus start procedure for normal participants.
func (node *Node) GetPrePrepare(prePrepareMsg *PrePrepareMsg) error {
	log2.LogMsg(prePrepareMsg)

	// Create a new state for the new consensus.
	err := node.createStateForNewConsensus()
	if err != nil {
		return err
	}

	prePareMsg, err := node.CurrentState.PrePrepare(prePrepareMsg)
	if err != nil {
		return err
	}

	if prePareMsg != nil {
		// Attach node ID to the message
		prePareMsg.NodeID = node.ID

		log2.LogStage("Pre-prepare", true)
		go node.Broadcast(node.ID, "prepare", prePareMsg)
		log2.LogStage("Prepare", false)
	}

	return nil
}

func (node *Node) GetPrepare(prepareMsg *VoteMsg) error {
	log2.LogMsg(prepareMsg)

	commitMsg, err := node.CurrentState.Prepare(prepareMsg)
	if err != nil {
		return err
	}

	if commitMsg != nil {
		// Attach node ID to the message
		commitMsg.NodeID = node.ID

		log2.LogStage("Prepare", true)
		go node.Broadcast(node.ID, "commit", commitMsg)
		log2.LogStage("Commit", false)
	}

	return nil
}

func (node *Node) GetCommit(commitMsg *VoteMsg) error {
	if node.CurrentState == nil {
		return nil
	}
	//util.LogMsg(commitMsg)
	//fmt.Println(node.ID)
	//fmt.Println(node.CurrentState)
	replyMsg, committedMsg, err := node.CurrentState.Commit(commitMsg)
	if err != nil {
		return err
	}

	if replyMsg != nil {
		if committedMsg == nil {
			return errors.New("committed message is nil, even though the reply message is not nil")
		}

		// Attach node ID to the message
		replyMsg.NodeID = node.ID

		// Save the last version of committed messages to node.
		node.CommittedMsgs = append(node.CommittedMsgs, committedMsg)

		log2.LogStage("Commit", true)
		node.Reply(replyMsg)
		log2.LogStage("Reply", true)
		node.CurrentState = nil
	}

	return nil
}

func (node *Node) createStateForNewConsensus() error {
	// Check if there is an ongoing consensus process.
	if node.CurrentState != nil {
		node.Println(node.CurrentState, "another consensus is ongoing")
		return errors.New("another consensus is ongoing")
	}

	// Get the last sequence ID
	var lastSequenceID int64
	if len(node.CommittedMsgs) == 0 {
		lastSequenceID = -1
	} else {
		lastSequenceID = node.CommittedMsgs[len(node.CommittedMsgs)-1].SequenceID
	}

	// Create a new state for this new consensus process in the Primary
	node.CurrentState = CreateState(node.View.ID, lastSequenceID, node.f, node.ff)

	log2.LogStage("Create the replica status", true)

	return nil
}

func (node *Node) routeMsgWhenAlarmed() []error {
	if node.CurrentState == nil {
		// Check ReqMsgs, send them.
		if len(node.MsgBuffer.ReqMsgs) != 0 {
			msgs := make([]*RequestMsg, len(node.MsgBuffer.ReqMsgs))
			copy(msgs, node.MsgBuffer.ReqMsgs)

			node.MsgDelivery <- msgs
		}

		// Check PrePrepareMsgs, send them.
		if len(node.MsgBuffer.PrePrepareMsgs) != 0 {
			msgs := make([]*PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
			copy(msgs, node.MsgBuffer.PrePrepareMsgs)

			node.MsgDelivery <- msgs
		}
	} else {
		switch node.CurrentState.CurrentStage {
		case PrePrepared:
			// Check PrepareMsgs, send them.
			if len(node.MsgBuffer.PrepareMsgs) != 0 {
				msgs := make([]*VoteMsg, len(node.MsgBuffer.PrepareMsgs))
				copy(msgs, node.MsgBuffer.PrepareMsgs)

				node.MsgDelivery <- msgs
			}
		case Prepared:
			// Check CommitMsgs, send them.
			if len(node.MsgBuffer.CommitMsgs) != 0 {
				msgs := make([]*VoteMsg, len(node.MsgBuffer.CommitMsgs))
				copy(msgs, node.MsgBuffer.CommitMsgs)

				node.MsgDelivery <- msgs
			}
		}
	}

	return nil
}

var mutex sync.Mutex

func (node *Node) resolveMsg() {
	for {
		// Get buffered messages from the dispatcher.
		//mutex.Lock()
		msgs := <-node.MsgDelivery
		switch msgs.(type) {
		case []*RequestMsg:
			errs := node.resolveRequestMsg(msgs.([]*RequestMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*PrePrepareMsg:
			errs := node.resolvePrePrepareMsg(msgs.([]*PrePrepareMsg))
			if len(errs) != 0 {
				for _, err := range errs {
					fmt.Println(err)
				}
				// TODO: send err to ErrorChannel
			}
		case []*VoteMsg:
			voteMsgs := msgs.([]*VoteMsg)
			if len(voteMsgs) == 0 {
				break
			}

			if voteMsgs[0].MsgType == PrepareMsg {
				errs := node.resolvePrepareMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}
			} else if voteMsgs[0].MsgType == CommitMsg {
				errs := node.resolveCommitMsg(voteMsgs)
				if len(errs) != 0 {
					for _, err := range errs {
						fmt.Println(err)
					}
					// TODO: send err to ErrorChannel
				}
			}
		}
		//mutex.Unlock()
	}
}

func (node *Node) alarmToDispatcher() {
	for {
		time.Sleep(ResolvingTimeDuration)
		err := node.routeMsgWhenAlarmed()
		if err != nil {
			node.Println(err)
		}
	}
}

func (node *Node) resolveRequestMsg(msgs []*RequestMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, reqMsg := range msgs {
		err := node.GetReq(reqMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrePrepareMsg(msgs []*PrePrepareMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, prePrepareMsg := range msgs {
		err := node.GetPrePrepare(prePrepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolvePrepareMsg(msgs []*VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, prepareMsg := range msgs {
		err := node.GetPrepare(prepareMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) resolveCommitMsg(msgs []*VoteMsg) []error {
	errs := make([]error, 0)

	// Resolve messages
	for _, commitMsg := range msgs {
		err := node.GetCommit(commitMsg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errs
	}

	return nil
}

func (node *Node) handleRequest(_ http.ResponseWriter, request *http.Request) {
	var msg RequestMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}

	if node.CurrentState == nil {
		// Copy buffered messages first.
		msgs := make([]*RequestMsg, len(node.MsgBuffer.ReqMsgs))
		copy(msgs, node.MsgBuffer.ReqMsgs)

		// Append a newly arrived message.
		msgs = append(msgs, &msg)

		// Empty the buffer.
		node.MsgBuffer.ReqMsgs = make([]*RequestMsg, 0)

		// Send messages.
		node.MsgDelivery <- msgs
	} else {
		node.MsgBuffer.ReqMsgs = append(node.MsgBuffer.ReqMsgs, &msg)
	}
}

func (node *Node) handlePrePrepare(_ http.ResponseWriter, request *http.Request) {
	var msg PrePrepareMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}

	if node.CurrentState == nil {
		// Copy buffered messages first.
		msgs := make([]*PrePrepareMsg, len(node.MsgBuffer.PrePrepareMsgs))
		copy(msgs, node.MsgBuffer.PrePrepareMsgs)

		// Append a newly arrived message.
		msgs = append(msgs, &msg)

		// Empty the buffer.
		node.MsgBuffer.PrePrepareMsgs = make([]*PrePrepareMsg, 0)

		// Send messages.
		node.MsgDelivery <- msgs
	} else {
		node.MsgBuffer.PrePrepareMsgs = append(node.MsgBuffer.PrePrepareMsgs, &msg)
	}
}

func (node *Node) handlePrepare(_ http.ResponseWriter, request *http.Request) {
	var msg VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}

	if node.CurrentState == nil || node.CurrentState.CurrentStage != PrePrepared {
		node.MsgBuffer.PrepareMsgs = append(node.MsgBuffer.PrepareMsgs, &msg)
	} else {
		// Copy buffered messages first.
		msgs := make([]*VoteMsg, len(node.MsgBuffer.PrepareMsgs))
		copy(msgs, node.MsgBuffer.PrepareMsgs)

		// Append a newly arrived message.
		msgs = append(msgs, &msg)

		// Empty the buffer.
		node.MsgBuffer.PrepareMsgs = make([]*VoteMsg, 0)

		// Send messages.
		node.MsgDelivery <- msgs
	}
}

func (node *Node) handleCommit(_ http.ResponseWriter, request *http.Request) {
	var msg VoteMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}

	if node.CurrentState == nil || node.CurrentState.CurrentStage != Prepared {
		node.MsgBuffer.CommitMsgs = append(node.MsgBuffer.CommitMsgs, &msg)
	} else {
		// Copy buffered messages first.
		msgs := make([]*VoteMsg, len(node.MsgBuffer.CommitMsgs))
		copy(msgs, node.MsgBuffer.CommitMsgs)

		// Append a newly arrived message.
		msgs = append(msgs, &msg)

		// Empty the buffer.
		node.MsgBuffer.CommitMsgs = make([]*VoteMsg, 0)

		// Send messages.
		node.MsgDelivery <- msgs
	}
}

func (node *Node) handleReply(_ http.ResponseWriter, request *http.Request) {
	var msg ReplyMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}

	node.GetReply(&msg)
}

func (node *Node) handleAdd(_ http.ResponseWriter, _ *http.Request) {
	node.SetF(node.total + 1)
}

func (node *Node) handleSetF(_ http.ResponseWriter, request *http.Request) {
	var msg SetFMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}
	node.SetF(msg.Total)
}

func (node *Node) SetF(total int) {
	node.total = total
	node.f = node.getF(total)
	node.ff = node.getFF(total)
	node.Println("f:", node.f, "; 2f:", node.ff)
}

func (node *Node) getF(total int) int {
	f := float64(total-1) / 3
	return int(math.Ceil(f))
}

func (node *Node) getFF(total int) int {
	ff := float64(total-1) / 1.5
	return int(math.Ceil(ff))
}

func (node *Node) handleClient(writer http.ResponseWriter, request *http.Request) {
	var msg ClientMsg
	err := json.NewDecoder(request.Body).Decode(&msg)
	if err != nil {
		node.Println(err)
		return
	}
	delay, err2 := node.StartRequest("Test")
	if err2 != nil {
		node.Println(err)
		return
	}
	msg.Delay = delay
	jsonMessage, _ := json.Marshal(msg)
	writer.Write(jsonMessage)
}
