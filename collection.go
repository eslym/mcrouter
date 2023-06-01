package main

import "sync"

type set[C comparable] struct {
	items map[C]bool
	lock  sync.RWMutex
}

type Set[C comparable] interface {
	Add(item C)
	Remove(item C)
	Contains(item C) bool
	Len() int
	Clear()
	Each(callback func(C) error) error
	Filter(callback func(C) (bool, error)) error
}

func NewSet[C comparable]() Set[C] {
	return &set[C]{
		items: make(map[C]bool),
	}
}

func (c *set[C]) Add(item C) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[item] = true
}

func (c *set[C]) Remove(item C) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, item)
}

func (c *set[C]) Contains(item C) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.items[item]
	return ok
}

func (c *set[C]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

func (c *set[C]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items = make(map[C]bool)
}

func (c *set[C]) Each(callback func(C) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for item := range c.items {
		err := callback(item)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *set[C]) Filter(callback func(C) (bool, error)) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for item := range c.items {
		ok, err := callback(item)
		if err != nil {
			return err
		}
		if !ok {
			delete(c.items, item)
		}
	}
	return nil
}

type _map[K comparable, V any] struct {
	items map[K]V
	lock  sync.RWMutex
}

type Map[K comparable, V any] interface {
	Set(key K, value V)
	Get(key K) (V, bool)
	Remove(key K)
	Contains(key K) bool
	Len() int
	Clear()
	Each(callback func(K, V) error) error
	Filter(callback func(K, V) (bool, error)) error
}

func NewMap[K comparable, V any]() Map[K, V] {
	return &_map[K, V]{
		items: make(map[K]V),
	}
}

func (c *_map[K, V]) Set(key K, value V) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items[key] = value
}

func (c *_map[K, V]) Get(key K) (V, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	value, ok := c.items[key]
	return value, ok
}

func (c *_map[K, V]) Remove(key K) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.items, key)
}

func (c *_map[K, V]) Contains(key K) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	_, ok := c.items[key]
	return ok
}

func (c *_map[K, V]) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.items)
}

func (c *_map[K, V]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.items = make(map[K]V)
}

func (c *_map[K, V]) Each(callback func(K, V) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	for key, value := range c.items {
		err := callback(key, value)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *_map[K, V]) Filter(callback func(K, V) (bool, error)) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for key, value := range c.items {
		ok, err := callback(key, value)
		if err != nil {
			return err
		}
		if !ok {
			delete(c.items, key)
		}
	}
	return nil
}
