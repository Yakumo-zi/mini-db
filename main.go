package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
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
			Length:  uint32(4 + 4 + 1 + 4 + len(k) + 4 + len(v)),
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
	payload.Value = make(Value, payload.ValueLength)
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

func RecordToBytes(r Record) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, 128))
	err := binary.Write(buffer, binary.BigEndian, r.Length)
	if err != nil {
		return nil, err
	}
	pbs := r.Payload.Bytes()
	r.CRC = crc32.ChecksumIEEE(pbs)
	err = binary.Write(buffer, binary.BigEndian, r.CRC)
	if err != nil {
		return nil, err
	}
	err = binary.Write(buffer, binary.BigEndian, r.Flags)
	if err != nil {
		return nil, err
	}
	_, err = buffer.Write(pbs)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func RecordFromBytes(bs []byte) (record Record, err error) {
	reader := bytes.NewReader(bs)
	err = binary.Read(reader, binary.BigEndian, &record.Length)
	err = binary.Read(reader, binary.BigEndian, &record.CRC)
	err = binary.Read(reader, binary.BigEndian, &record.Flags)
	buf := make([]byte, reader.Len())
	_, _ = io.ReadFull(reader, buf)
	payload := PayloadFromBytes(buf)
	record.Payload = payload
	return record, err
}

func StoreRecordToFile(file *os.File, record Record) error {
	bs, err := RecordToBytes(record)
	if err != nil {
		return err
	}
	_, err = file.Write(bs)
	return err
}

func LoadRecordsFromFile(file *os.File) ([]Record, error) {
	records := make([]Record, 0, 10)
	var err error
	for {
		var length uint32
		err = binary.Read(file, binary.BigEndian, &length)
		if err != nil {
			fmt.Printf("load length\n")
			break
		}
		bs := make([]byte, length-4)
		err = binary.Read(file, binary.BigEndian, bs)
		if err != nil {
			fmt.Printf("load record,%d\n", length)
			break
		}
		lbs := make([]byte, 4)
		_, err = binary.Encode(lbs, binary.BigEndian, length)
		if err != nil {
			fmt.Printf("encode length\n")
			break
		}
		bs = append(lbs, bs...)
		record, err := RecordFromBytes(bs)
		if err != nil {
			fmt.Printf("parse\n")
			break
		}
		records = append(records, record)
	}
	if err != io.EOF {
		return nil, err
	}
	return records, nil
}

func main() {
	DBTest()
}

func DBTest() {
	db := MiniDB{}
	dwf, err := os.OpenFile("./data.db", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("%s", err)
		return
	}
	defer dwf.Close()
	for n := range 5 {
		key := Key(fmt.Sprintf("key%d", n))
		value := Value(fmt.Sprintf("value%d", n))
		db.Put(key, value)
	}
	records := db.Records()
	for _, record := range records {
		err = StoreRecordToFile(dwf, record)
		if err != nil {
			fmt.Printf("%s\n", err)
			break
		}
	}
	if err = dwf.Sync(); err != nil {
		fmt.Printf("%s\n", err)
	}
	drf, err := os.OpenFile("./data.db", os.O_RDONLY, 0o644)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	defer drf.Close()
	records, err = LoadRecordsFromFile(drf)
	if err != nil {
		fmt.Printf("%s\n", err)
		return
	}
	fmt.Printf("records:\n%+v\n", records)
}
