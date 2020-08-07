package channel

import (
	"github.com/tcrain/cons/consensus/auth/sig"
	"github.com/tcrain/cons/consensus/auth/sig/sleep"
)

type EncryptInterface interface {
	FailedGetPub()
	GotPub(pubBytes []byte, randBytes []byte) error
	GetRandomBytes() (ret [24]byte)
	GetMyPubBytes() []byte
	WaitForPub() (err error)
	Decode(msg []byte, includeSize bool) ([]byte, error)
	GetExternalPub() sig.Pub
	Encode(msg []byte, includeSize bool) []byte
}

func GenerateEncrypter(myPriv sig.Priv,
	shouldWaitForPub bool,
	externalPub sig.Pub) EncryptInterface {

	if _, ok := myPriv.(sleep.SleepPriv); ok {
		return NewSleepEcrypter(myPriv, shouldWaitForPub, externalPub)
	} else {
		return NewEcrypter(myPriv, shouldWaitForPub, externalPub)
	}
}
