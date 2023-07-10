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
	OnMediaGroup
)

type State int

type ContentType int
