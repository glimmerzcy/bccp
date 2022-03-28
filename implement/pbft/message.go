package pbft

type SetFMsg struct {
	Total int
}

type ClientMsg struct {
	Operation string
	Delay     int64
}
