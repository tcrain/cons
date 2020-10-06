package deserialized

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

// DeserializedItem describes the state of a message that has been deserialized
// or attepmted to be derserialized.
type DeserializedItem struct {
	Index            types.ConsensusIndex   // This is the index of the item in the total order of consensus iterations.
	HeaderType       messages.HeaderID      // This is the HeaderID of the message.
	Header           messages.MsgHeader     // This is the derserialized header.
	Message          sig.EncodedMsg         // This is a pointer to the message bytes of only this header (note this message could be shared among multiple headers so we should not write to it directly).
	IsDeserialized   bool                   // If true the message has been deserialized successfuly, and all fields of this struct must be valie, if false only the Message field is valid and contains a message that has not yet been deserialized.
	IsLocal          types.LocalMessageType // This is true if the message was sent from the local node.
	NewTotalSigCount int                    // The number of signatures for the specific message received so far (0 if unsigend message type).
	NewMsgIDSigCount int                    // The number of signatures for the MsgID of the message (see messages.MsgID) (0 if unsigned message type).
	MC               interface{}            // This points to the member checkers, and is just used to check they do not change between deserializing and processing a message // TODO clean this up
}

// CopyBasic returns a new Desierilzed item that has the same fields: Index, IsDeserialized, IsLocal
func (di *DeserializedItem) CopyBasic() *DeserializedItem {
	return &DeserializedItem{
		Index: di.Index,
		// FirstIndex: di.FirstIndex,
		// AdditionalIndices: di.AdditionalIndices,
		IsDeserialized: di.IsDeserialized,
		IsLocal:        di.IsLocal,
	}
}
