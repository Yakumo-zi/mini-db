package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

type Key []byte

func (k Key) String() string {
	return string(k)
}

type Value []byte

func (v Value) String() string {
	return string(v)
}

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

func (m MiniDB) Records() []Record {
	records := make([]Record, 0, len(m))
	for k, v := range m {
		payload := Payload{
			KeyLength:   uint32(len(k)),
			Key:         Key(k),
			ValueLength: uint32(len(v)),
			Value:       v,
		}
		record := Record{
			Payload: payload,
		}
		records = append(records, record)
	}
	return records
}

type Payload struct {
	KeyLength   uint32
	Key         Key
	ValueLength uint32
	Value       Value
}

func PayloadFromBytes(bs []byte) Payload {
	reader := bytes.NewReader(bs)
	payload := Payload{}
	binary.Read(reader, binary.BigEndian, &payload.KeyLength)
	payload.Key = make(Key, payload.KeyLength)
	binary.Read(reader, binary.BigEndian, payload.Key)
	binary.Read(reader, binary.BigEndian, &payload.ValueLength)
	payload.Value = make(Value, payload.KeyLength)
	binary.Read(reader, binary.BigEndian, payload.Value)
	return payload
}

func (p Payload) Length() uint32 {
	return uint32(4 + len(p.Key) + 4 + len(p.Value))
}

func (p Payload) Bytes() []byte {
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, p.KeyLength)
	binary.Write(buffer, binary.BigEndian, p.Key)
	binary.Write(buffer, binary.BigEndian, p.ValueLength)
	binary.Write(buffer, binary.BigEndian, p.Value)
	return buffer.Bytes()
}

type Record struct {
	Length  uint32
	CRC     uint32
	Flags   byte
	Payload Payload
}

func RecordToBytes(r Record) []byte {
	buffer := bytes.NewBuffer(make([]byte, 0, 128))
	binary.Write(buffer, binary.BigEndian, uint32(4+4+1+r.Payload.Length()))
	pbs := r.Payload.Bytes()
	r.CRC = crc32.ChecksumIEEE(pbs)
	fmt.Printf("crc=%d\n", r.CRC)
	binary.Write(buffer, binary.BigEndian, r.CRC)
	binary.Write(buffer, binary.BigEndian, r.Flags)
	buffer.Write(pbs)
	return buffer.Bytes()
}

func RecordFromBytes(bs []byte) Record {
	reader := bytes.NewReader(bs)
	record := Record{}
	binary.Read(reader, binary.BigEndian, &record.Length)
	binary.Read(reader, binary.BigEndian, &record.CRC)
	binary.Read(reader, binary.BigEndian, &record.Flags)
	buf := make([]byte, reader.Len())
	_, _ = io.ReadFull(reader, buf)
	payload := PayloadFromBytes(buf)
	record.Payload = payload
	return record
}

func main() {
	DBTest()
}

func DBTest() {
	db := MiniDB{}
	for n := range 5 {
		key := Key(fmt.Sprintf("key%d", n))
		value := Value(fmt.Sprintf("value%d", n))
		db.Put(key, value)
	}
	records := db.Records()
	fmt.Printf("%+v\n", records)
}
