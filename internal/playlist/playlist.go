package playlist

import (
	"errors"

	"github.com/5218664b/douyu-streamer/internal/library"
)

type Playlist struct {
	items   []library.Item
	current int
	history []library.Item
}

func New(items []library.Item) (*Playlist, error) {
	if len(items) == 0 {
		return nil, errors.New("playlist requires at least one item")
	}

	return &Playlist{
		items:   append([]library.Item(nil), items...),
		current: 0,
	}, nil
}

func (p *Playlist) Items() []library.Item {
	return append([]library.Item(nil), p.items...)
}

func (p *Playlist) Current() library.Item {
	return p.items[p.current]
}

func (p *Playlist) Next() library.Item {
	if len(p.items) == 1 {
		return p.items[0]
	}

	next := p.current + 1
	if next >= len(p.items) {
		next = 0
	}
	return p.items[next]
}

func (p *Playlist) Advance() library.Item {
	current := p.Current()
	p.history = append(p.history, current)
	p.current++
	if p.current >= len(p.items) {
		p.current = 0
	}
	return p.Current()
}

func (p *Playlist) History() []library.Item {
	return append([]library.Item(nil), p.history...)
}

func (p *Playlist) Select(index int) (library.Item, error) {
	if index < 1 || index > len(p.items) {
		return library.Item{}, errors.New("playlist index out of range")
	}

	current := p.Current()
	p.history = append(p.history, current)
	p.current = index - 1
	return p.Current(), nil
}
