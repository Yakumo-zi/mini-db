package main

import (
	"errors"
	"fmt"
)

type Key []byte

func (k Key) String() string {
	return string(k)
}

type Value []byte

type DB interface {
	Get(key Key) (Value, error)
	Put(key Key, value Value) error
	Del(key Key)
}

var _ DB = (*MiniDB)(nil)

type MiniDB map[string]Value

func (m MiniDB) Get(key Key) (Value, error) {
	if v, ok := m[key.String()]; ok {
		return v, nil
	}
	return nil, errors.New("get key error")
}

func (m MiniDB) Put(key Key, value Value) error {
	m[key.String()] = value
	return nil
}

func (m MiniDB) Del(key Key) {
	delete(m, key.String())
}

func main() {
	db := MiniDB{}
	for n := range 5 {
		key := Key(fmt.Sprintf("key%d", n))
		value := Value(fmt.Sprintf("value%d", n))
		db.Put(key, value)
	}
	v, err := db.Get(Key("key3"))
	fmt.Printf("%s,%+v\n", v, err)
}
