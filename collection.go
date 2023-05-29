package main

import "sync"

type Collection[C any] struct {
	items map[*C]bool
	lock  sync.RWMutex
}

func NewCollection[C any]() *Collection[C] {
	return &Collection[C]{
		items: make(map[*C]bool),
	}
}

func (c *Collection[C]) Add(item *C) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[item] = true
}

func (c *Collection[C]) Remove(item *C) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, item)
}

func (c *Collection[C]) Contains(item *C) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.items[item]
	return ok
}

func (c *Collection[C]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

func (c *Collection[C]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items = make(map[*C]bool)
}

func (c *Collection[C]) Each(f func(*C) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for item := range c.items {
		err := f(item)
		if err != nil {
			return err
		}
	}
	return nil
}
