package pbft

type RequestMsg struct {
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientID"`
	Operation  string `json:"Operation"`
	SequenceID int64  `json:"sequenceID"`
}

type ReplyMsg struct {
	ViewID    int64  `json:"viewID"`
	Timestamp int64  `json:"timestamp"`
	ClientID  string `json:"clientID"`
	NodeID    string `json:"nodeID"`
	Result    string `json:"result"`
}

type PrePrepareMsg struct {
	ViewID     int64       `json:"viewID"`
	SequenceID int64       `json:"sequenceID"`
	Digest     string      `json:"digest"`
	RequestMsg *RequestMsg `json:"requestMsg"`
}

type VoteMsg struct {
	ViewID     int64  `json:"viewID"`
	SequenceID int64  `json:"sequenceID"`
	Digest     string `json:"digest"`
	NodeID     string `json:"nodeID"`
	MsgType    `json:"msgType"`
}

type MsgType int

type RouterMsg struct {
	ID  string `json:"id"`
	URL string `json:"url"`
	OP  int    `json:"op"`
}

const (
	PrepareMsg MsgType = iota
	CommitMsg
)
