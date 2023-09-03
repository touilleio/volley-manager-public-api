package main

import (
	"golang.org/x/sync/errgroup"
	"time"
)

type fetcher struct {
	state *state
}

func newFetcher(state *state) (*fetcher, error) {
	internalFetcher := fetcher{
		state: state,
	}
	return &internalFetcher, nil
}

func (f fetcher) fetch() error {

	return nil
}

func (f fetcher) run(interval time.Duration, g *errgroup.Group) {

	g.Go(func() error {
		for range time.Tick(interval) {
			err := f.fetch()
			if err != nil {
				return err
			}
		}
		return nil
	})
}
