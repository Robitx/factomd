// Copyright (c) 2014-2015 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package messages

import (
	"bytes"
	"fmt"
	//"encoding/binary"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/primitives"
	"io"
)

// RejectCode represents a numeric value by which a remote peer indicates
// why a message was rejected.
type RejectCode uint8

// These constants define the various supported reject codes.
const (
	RejectMalformed       RejectCode = 0x01
	RejectInvalid         RejectCode = 0x10
	RejectObsolete        RejectCode = 0x11
	RejectDuplicate       RejectCode = 0x12
	RejectNonstandard     RejectCode = 0x40
	RejectDust            RejectCode = 0x41
	RejectInsufficientFee RejectCode = 0x42
	RejectCheckpoint      RejectCode = 0x43
)

// Map of reject codes back strings for pretty printing.
var rejectCodeStrings = map[RejectCode]string{
	RejectMalformed:       "REJECT_MALFORMED",
	RejectInvalid:         "REJECT_INVALID",
	RejectObsolete:        "REJECT_OBSOLETE",
	RejectDuplicate:       "REJECT_DUPLICATE",
	RejectNonstandard:     "REJECT_NONSTANDARD",
	RejectDust:            "REJECT_DUST",
	RejectInsufficientFee: "REJECT_INSUFFICIENTFEE",
	RejectCheckpoint:      "REJECT_CHECKPOINT",
}

// String returns the RejectCode in human-readable form.
func (code RejectCode) String() string {
	if s, ok := rejectCodeStrings[code]; ok {
		return s
	}

	return fmt.Sprintf("Unknown RejectCode (%d)", uint8(code))
}

// MsgReject implements the Message interface and represents a bitcoin reject
// message.
//
// This message was not added until protocol version RejectVersion.
type MsgReject struct {
	MessageBase
	// Cmd is the command for the message which was rejected such as
	// as CmdBlock or CmdTx.  This can be obtained from the Command function
	// of a Message.
	Cmd string

	// RejectCode is a code indicating why the command was rejected.  It
	// is encoded as a uint8 on the messages.
	Code RejectCode

	// Reason is a human-readable string with specific details (over and
	// above the reject code) about why the command was rejected.
	Reason string

	// Hash identifies a specific block or transaction that was rejected
	// and therefore only applies the MsgBlock and MsgTx messages.
	Hash interfaces.IHash
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgReject) BtcDecode(r io.Reader, pver uint32) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.BtcDecode", str)
	}

	// Command that was rejected.
	cmd, err := readVarString(r, pver)
	if err != nil {
		return err
	}
	msg.Cmd = cmd

	// Code indicating why the command was rejected.
	err = readElement(r, &msg.Code)
	if err != nil {
		return err
	}

	// Human readable string with specific details (over and above the
	// reject code above) about why the command was rejected.
	reason, err := readVarString(r, pver)
	if err != nil {
		return err
	}
	msg.Reason = reason

	// CmdBlock and CmdTx messages have an additional hash field that
	// identifies the specific block or transaction.
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := readElement(r, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgReject) BtcEncode(w io.Writer, pver uint32) error {
	if pver < RejectVersion {
		str := fmt.Sprintf("reject message invalid for protocol "+
			"version %d", pver)
		return messageError("MsgReject.BtcEncode", str)
	}

	// Command that was rejected.
	err := writeVarString(w, pver, msg.Cmd)
	if err != nil {
		return err
	}

	// Code indicating why the command was rejected.
	err = writeElement(w, msg.Code)
	if err != nil {
		return err
	}

	// Human readable string with specific details (over and above the
	// reject code above) about why the command was rejected.
	err = writeVarString(w, pver, msg.Reason)
	if err != nil {
		return err
	}

	// CmdBlock and CmdTx messages have an additional hash field that
	// identifies the specific block or transaction.
	if msg.Cmd == CmdBlock || msg.Cmd == CmdTx {
		err := writeElement(w, &msg.Hash)
		if err != nil {
			return err
		}
	}

	return nil
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgReject) Command() string {
	return CmdReject
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgReject) MaxPayloadLength(pver uint32) uint32 {
	plen := uint32(0)
	// The reject message did not exist before protocol version
	// RejectVersion.
	if pver >= RejectVersion {
		// Unfortunately the bitcoin protocol does not enforce a sane
		// limit on the length of the reason, so the max payload is the
		// overall maximum message payload.
		plen = MaxMessagePayload
	}

	return plen
}

// NewMsgReject returns a new bitcoin reject message that conforms to the
// Message interface.  See MsgReject for details.
func NewMsgReject(command string, code RejectCode, reason string) *MsgReject {
	return &MsgReject{
		Cmd:    command,
		Code:   code,
		Reason: reason,
	}
}

var _ interfaces.IMsg = (*MsgReject)(nil)

func (m *MsgReject) Process(uint32, interfaces.IState) {}

func (m *MsgReject) GetHash() interfaces.IHash {
	return nil
}

func (m *MsgReject) GetMsgHash() interfaces.IHash {
	if m.MsgHash == nil {
		data, err := m.MarshalBinary()
		if err != nil {
			return nil
		}
		m.MsgHash = primitives.Sha(data)
	}
	return m.MsgHash
}


func (m *MsgReject) GetTimestamp() interfaces.Timestamp {
	return 0
}

func (m *MsgReject) Type() int {
	return -1
}

func (m *MsgReject) Int() int {
	return -1
}

func (m *MsgReject) Bytes() []byte {
	return nil
}

func (m *MsgReject) UnmarshalBinaryData(data []byte) (newdata []byte, err error) {
	var pver uint32
	r := bytes.NewBuffer(data)
	m.Cmd, err = readVarString(r, pver)
	if err != nil {
		return
	}

	err = readElement(r, &m.Code)
	if err != nil {
		return
	}

	m.Reason, err = readVarString(r, pver)
	if err != nil {
		return
	}

	err = readElement(r, &m.Hash)
	if err != nil {
		return
	}

	return nil, nil
}

func (m *MsgReject) UnmarshalBinary(data []byte) error {
	_, err := m.UnmarshalBinaryData(data)
	return err
}

func (m *MsgReject) MarshalBinary() (data []byte, err error) {
	var pver uint32
	buf := bytes.NewBuffer(make([]byte, 0, m.MaxPayloadLength(pver)))
	err = writeVarString(buf, 0, m.Cmd)
	if err != nil {
		return
	}

	//binary.Write(buf, binary.BigEndian, m.Code)
	err = writeElement(buf, m.Code)
	if err != nil {
		return
	}
	err = writeVarString(buf, 0, m.Reason)
	if err != nil {
		return
	}
	err = writeElement(buf, &m.Hash)
	if err != nil {
		return
	}

	//h, err := m.Hash.MarshalBinary()
	//if err != nil {
	//return nil, err
	//}
	//buf.Write(h)

	data = buf.Bytes()
	return
}

func (m *MsgReject) MarshalForSignature() (data []byte, err error) {
	return nil, nil
}

func (m *MsgReject) String() string {
	return ""
}

// Validate the message, given the state.  Three possible results:
//  < 0 -- MsgReject is invalid.  Discard
//  0   -- Cannot tell if message is Valid
//  1   -- MsgReject is valid
func (m *MsgReject) Validate(dbheight uint32, state interfaces.IState) int {
	return 0
}

// Returns true if this is a message for this server to execute as
// a leader.
func (m *MsgReject) Leader(state interfaces.IState) bool {
	switch state.GetNetworkNumber() {
	case 0: // Main Network
		panic("Not implemented yet")
	case 1: // Test Network
		panic("Not implemented yet")
	case 2: // Local Network
		panic("Not implemented yet")
	default:
		panic("Not implemented yet")
	}
}

// Execute the leader functions of the given message
func (m *MsgReject) LeaderExecute(state interfaces.IState) error {
	return nil
}

// Returns true if this is a message for this server to execute as a follower
func (m *MsgReject) Follower(interfaces.IState) bool {
	return true
}

func (m *MsgReject) FollowerExecute(interfaces.IState) error {
	return nil
}

func (e *MsgReject) JSONByte() ([]byte, error) {
	return primitives.EncodeJSON(e)
}

func (e *MsgReject) JSONString() (string, error) {
	return primitives.EncodeJSONString(e)
}

func (e *MsgReject) JSONBuffer(b *bytes.Buffer) error {
	return primitives.EncodeJSONToBuffer(e, b)
}
