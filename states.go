package main

type State int
const (
	StateDefault State = iota
	StatePostSending
	StatePostSent
	StatePostFailed
	StatePostPreparing
	StateFlairSelect
	StateFlairConfirm
	StateRepost
)
