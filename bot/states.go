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
	AwaitSendState
	TwitterAskState
	TwitterSendState
	ExtAskState
	ExtSendState

	OnText ContentType = iota
	OnPhoto
	OnVideo
	OnAnimation
	OnMediaGroup
)

type State int

type ContentType int
