package bot

const (
	DefaultState State = iota
	AnyState
	LockState
	GetNextFlairState
	SubmitImageState
	CreateFlairMessageState
	AwaitFlairMessageState
	SubmitPostState

	OnText ContentType = iota
	OnPhoto
	OnVideo
)

type State int

type ContentType int
