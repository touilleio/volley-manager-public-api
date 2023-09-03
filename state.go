package main

type state struct {
}

func newState() *state {
	internalState := state{}
	return &internalState
}
