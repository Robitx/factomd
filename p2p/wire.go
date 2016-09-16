// Copyright 2016 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package p2p

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

var _ = fmt.Print

type Wire struct {
	conn    net.Conn
	encoder *gob.Encoder // Wire format is gobs in this version, may switch to binary
	decoder *gob.Decoder // Wire format is gobs in this version, may switch to binary

	isNew bool
}

var Writes int
var Reads int
var WritesErr int
var ReadsErr int

var Deadline time.Duration = time.Duration(100)

type ParcelPack struct {
	Payload []byte
}

func NewWire(conn net.Conn) *Wire {
	w := new(Wire)
	w.conn = conn
	return w
}


func (m *Wire) Init() {
	m.encoder = gob.NewEncoder(m)
	m.decoder = gob.NewDecoder(m)
}

func (m *Wire) Close() {
	if m.conn != nil {
		m.conn.Close()
	}
	m.decoder = nil
	m.encoder = nil
}

func (m *Wire) Send(p Parcel) (err error) {
	if m.encoder != nil {
		m.conn.SetWriteDeadline(time.Now().Add(Deadline))
		pack := new(ParcelPack)
		var err error
		pack.Payload, err = p.MarshalBinary()
		if err != nil && len(pack.Payload) == 0 {
			return err
		}
		m.encoder.Encode(pack)
	}
	return err
}

func (m *Wire) Receive() (p *Parcel, err error) {
	var pack ParcelPack
	p = new(Parcel)
	m.conn.SetReadDeadline(time.Now().Add(Deadline))
	err = m.decoder.Decode(&pack)
	if len(pack.Payload) > 0 {
		err = p.UnmarshalBinary(pack.Payload)
		if err != nil {
			return nil, err
		}
		return p, err
	}
	return nil, err
}

func (m *Wire) Write(b []byte) (int, error) {

	i, e := m.conn.Write(b)

	Writes += i

	if i > 0 {
		e = nil
	}

	if e != nil {
		WritesErr++
	}
	//fmt.Println("Write Done",time.Now().String())
	return i, e
}

func (m *Wire) Read(b []byte) (int, error) {

	i, e := m.conn.Read(b)

	if i > 0 {
		e = nil
	}

	//end := 10
	//if end > len(b) {
	//	end = len(b)
	//}
	//if e == nil {
	//	fmt.Printf("bbbb Read  %s %d bytes, Data: %x\n", time.Now().String(), len(b), b[:end])
	//}
	Reads += i

	if e != nil {
		ReadsErr++
	}
	//fmt.Println("Read Done",time.Now().String())
	return i, e
}
