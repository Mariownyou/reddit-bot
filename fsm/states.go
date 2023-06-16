package fsm

const (
	DefaultState State = iota
	AnyState
	LockState
	GetNextFlairState
	SubmitImageState
	CreateFlairMessageState
	AwaitFlairMessageState
	SubmitPostState

	TextContentType ContentType = iota
	PhotoContentType
	VideoContentType
)

type State int

type ContentType int
