// Copyright 2015 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package messages

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/FactomProject/factomd/common/constants"
	"github.com/FactomProject/factomd/common/interfaces"
	"github.com/FactomProject/factomd/common/primitives"
)

// Communicate a Directory Block State

type RemoveServerMsg struct {
	MessageBase
	Timestamp     interfaces.Timestamp // Message Timestamp
	ServerChainID interfaces.IHash     // ChainID of new server
	ServerType    int                  // 0 = Federated, 1 = Audit

	Signature interfaces.IFullSignature
}

var _ interfaces.IMsg = (*RemoveServerMsg)(nil)
var _ Signable = (*RemoveServerMsg)(nil)

func (m *RemoveServerMsg) GetHash() interfaces.IHash {
	return m.GetMsgHash()
}

func (m *RemoveServerMsg) GetMsgHash() interfaces.IHash {
	if m.MsgHash == nil {
		data, err := m.MarshalForSignature()
		if err != nil {
			return nil
		}
		m.MsgHash = primitives.Sha(data)
	}
	return m.MsgHash
}

func (m *RemoveServerMsg) Type() byte {
	return constants.REMOVESERVER_MSG
}

func (m *RemoveServerMsg) Int() int {
	return -1
}

func (m *RemoveServerMsg) Bytes() []byte {
	return nil
}

func (m *RemoveServerMsg) GetTimestamp() interfaces.Timestamp {
	return m.Timestamp
}

func (m *RemoveServerMsg) Validate(state interfaces.IState) int {
	return 1
	// Check to see if identity exists and is audit or fed server
	if !state.VerifyIsAuthority(m.ServerChainID) {
		fmt.Printf("RemoveServerMsg Error: [%s] is not a server, cannot be removed\n", m.ServerChainID.String()[:8])
		return -1
	}
	// Check signatures
	bytes, err := m.MarshalForSignature()
	if err != nil {
		fmt.Println("RemoveServerMsg Error: Err is not nil, err: ", err.Error())
		return -1
	}
	if m.Signature == nil {
		fmt.Println("RemoveServerMsg Error: No signiture on RemoveServerMsg")
		return -1
	}
	sig := m.Signature.GetSignature()
	authSigned, err := state.VerifyFederatedSignature(bytes, sig)

	//ackSigned, err := m.VerifySignature()
	if err != nil {
		fmt.Println("RemoveServerMsg Error: Err is not nil, err: ", err.Error())
		return -1
	}
	if !authSigned {
		return -1
	}
	return 1
}

// Returns true if this is a message for this server to execute as
// a leader.
func (m *RemoveServerMsg) ComputeVMIndex(state interfaces.IState) {
	m.VMIndex = state.ComputeVMIndex(constants.ADMIN_CHAINID)
}

// Execute the leader functions of the given message
func (m *RemoveServerMsg) LeaderExecute(state interfaces.IState) {
	state.LeaderExecute(m)
}

func (m *RemoveServerMsg) FollowerExecute(state interfaces.IState) {
	state.FollowerExecuteMsg(m)
}

// Acknowledgements do not go into the process list.
func (e *RemoveServerMsg) Process(dbheight uint32, state interfaces.IState) bool {
	return state.ProcessRemoveServer(dbheight, e)
}

func (e *RemoveServerMsg) JSONByte() ([]byte, error) {
	return primitives.EncodeJSON(e)
}

func (e *RemoveServerMsg) JSONString() (string, error) {
	return primitives.EncodeJSONString(e)
}

func (e *RemoveServerMsg) JSONBuffer(b *bytes.Buffer) error {
	return primitives.EncodeJSONToBuffer(e, b)
}

func (m *RemoveServerMsg) Sign(key interfaces.Signer) error {
	signature, err := SignSignable(m, key)
	if err != nil {
		return err
	}
	m.Signature = signature
	return nil
}

func (m *RemoveServerMsg) GetSignature() interfaces.IFullSignature {
	return m.Signature
}

func (m *RemoveServerMsg) VerifySignature() (bool, error) {
	return VerifyMessage(m)
}

func (m *RemoveServerMsg) UnmarshalBinaryData(data []byte) (newData []byte, err error) {
	defer func() {
		return
		if r := recover(); r != nil {
			err = fmt.Errorf("Error unmarshalling Add Server Message: %v", r)
		}
	}()
	newData = data
	if newData[0] != m.Type() {
		return nil, fmt.Errorf("Invalid Message type")
	}
	newData = newData[1:]

	m.Timestamp = new(primitives.Timestamp)
	newData, err = m.Timestamp.UnmarshalBinaryData(newData)
	if err != nil {
		return nil, err
	}

	m.ServerChainID = new(primitives.Hash)
	newData, err = m.ServerChainID.UnmarshalBinaryData(newData)
	if err != nil {
		return nil, err
	}

	m.ServerType = int(newData[0])
	newData = newData[1:]

	if len(newData) > 32 {
		m.Signature = new(primitives.Signature)
		newData, err = m.Signature.UnmarshalBinaryData(newData)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (m *RemoveServerMsg) UnmarshalBinary(data []byte) error {
	_, err := m.UnmarshalBinaryData(data)
	return err
}

func (m *RemoveServerMsg) MarshalForSignature() ([]byte, error) {
	var buf primitives.Buffer

	binary.Write(&buf, binary.BigEndian, m.Type())

	t := m.GetTimestamp()
	data, err := t.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	data, err = m.ServerChainID.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	binary.Write(&buf, binary.BigEndian, uint8(m.ServerType))

	return buf.DeepCopyBytes(), nil
}

func (m *RemoveServerMsg) MarshalBinary() ([]byte, error) {
	var buf primitives.Buffer

	data, err := m.MarshalForSignature()
	if err != nil {
		return nil, err
	}
	buf.Write(data)

	if m.Signature != nil {
		data, err = m.Signature.MarshalBinary()
		if err != nil {
			return nil, err
		}
		buf.Write(data)
	}

	return buf.DeepCopyBytes(), nil
}

func (m *RemoveServerMsg) String() string {
	var stype string
	if m.ServerType == 0 {
		stype = "Federated"
	} else {
		stype = "Audit"
	}
	return fmt.Sprintf("RemoveServer (%s): ChainID: %x Time: %x Msg Hash %x ",
		stype,
		m.ServerChainID.Bytes()[:3],
		&m.Timestamp,
		m.GetMsgHash().Bytes()[:3])

}

func (m *RemoveServerMsg) IsSameAs(b *RemoveServerMsg) bool {
	if b == nil {
		return false
	}
	if m.Timestamp.GetTimeMilli() != b.Timestamp.GetTimeMilli() {
		return false
	}
	if !m.ServerChainID.IsSameAs(b.ServerChainID) {
		return false
	}
	if m.ServerType != b.ServerType {
		return false
	}
	if m.Signature == nil && b.Signature != nil {
		return false
	}
	if m.Signature != nil {
		if m.Signature.IsSameAs(b.Signature) == false {
			return false
		}
	}
	return true
}

func NewRemoveServerMsg(state interfaces.IState, chainId interfaces.IHash, serverType int) interfaces.IMsg {
	msg := new(RemoveServerMsg)
	msg.ServerChainID = chainId
	msg.ServerType = serverType
	msg.Timestamp = state.GetTimestamp()

	return msg

}