package sleep

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/messages"
	"github.com/tcrain/cons/consensus/types"
)

type sleepPub interface {
	sig.Pub
	// doDeserializeSig deserializes a public key and signature object from m, size is the number of bytes read
	doDeserializeSig(theSig sig.Sig, newPub sig.Pub, m *messages.Message, signType types.SignType) (*sig.SigItem, int, error)
}

type SleepPriv interface {
	setPub(sig.Pub)
	sig.Priv
	IsSleepPriv()
}
