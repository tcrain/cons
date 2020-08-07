package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
	"io"
)

type Sig struct {
	bytes []byte
	stats *sig.SigStats
}

//New creates a new empty signature object
func (s *Sig) New() sig.Sig {
	return &Sig{stats: s.stats}
}

// PeekHeader returns the indices related to the messages without impacting m.
func (*Sig) PeekHeaders(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (index types.ConsensusIndex, err error) {
	_ = unmarFunc // we want a nil unmarFunc
	return messages.PeekHeaderHead(types.NilIndexFuns, (*messages.MsgBuffer)(m))
}

func (sig *Sig) Encode(writer io.Writer) (n int, err error) {
	return writer.Write(sig.bytes)
}
func (sig *Sig) Decode(reader io.Reader) (n int, err error) {
	sig.bytes = make([]byte, sig.stats.SigSize)
	return reader.Read(sig.bytes)
}

// GetRand returns a random binary from the signature if supported.
func (sig *Sig) GetRand() types.BinVal {
	// TODO this is only for coin using threshold signatures, should cleanup
	return types.BinVal(types.GetHash(sig.bytes)[0] % 2)
}

// Serialize the signature into the message, and return the nuber of bytes written
func (sig *Sig) Serialize(m *messages.Message) (int, error) {
	l, sizeOffset, _ := messages.WriteHeaderHead(nil, nil, sig.GetID(), 0, (*messages.MsgBuffer)(m))

	(*messages.MsgBuffer)(m).AddBytes(sig.bytes)
	l += len(sig.bytes)

	// now update the size
	err := (*messages.MsgBuffer)(m).WriteUint32At(sizeOffset, uint32(l))
	if err != nil {
		return 0, err
	}

	return l, nil
}

// GetBytes returns the bytes of the signature from the message
func (sig *Sig) GetBytes(m *messages.Message) ([]byte, error) {
	size, err := (*messages.MsgBuffer)(m).PeekUint32()
	if err != nil {
		return nil, err
	}
	return (*messages.MsgBuffer)(m).ReadBytes(int(size))
}

// Derserialize  updates the fields of the ECDSA signature object from m, and returns the number of bytes read
func (sig *Sig) Deserialize(m *messages.Message, unmarFunc types.ConsensusIndexFuncs) (int, error) {
	_ = unmarFunc // we want a nil unmarFunc
	l, _, _, size, _, err := messages.ReadHeaderHead(sig.GetID(), nil, (*messages.MsgBuffer)(m))
	if err != nil {
		return l, err
	}

	sig.bytes, err = (*messages.MsgBuffer)(m).ReadBytes(sig.stats.SigSize)
	if err != nil {
		return l, err
	}
	l += len(sig.bytes)

	if size != uint32(l) {
		return l, types.ErrInvalidMsgSize
	}

	return l, nil
}

// GetID returns the header id for ECDSA signatures
func (sig *Sig) GetID() messages.HeaderID {
	return messages.HdrSleepSig
}

// GetMsgID returns the message ID for ECDSA sig header
func (sig *Sig) GetMsgID() messages.MsgID {
	return messages.BasicMsgID(sig.GetID())
}
