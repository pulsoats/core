package websocket

type Command struct {
	Op      CmdOp
	Payload any
}
type CmdOp int

const (
	CmdSendJSON CmdOp = iota
	CmdClose
)
