# Cryptographic operations

Various cryptographic libraries are used for different operations.
Many are experimental.
Where possible I try to use existing implementations, but
some simple operations operations are implemented in this library (i.e. no guarantees they are correct).

## Signatures

#### Normal signatures
Used to sign consensus messages
- ECDSA - https://golang.org/pkg/crypto/ecdsa/
- EDDSA - https://github.com/dedis/kyber/tree/master/sign/eddsa
- BLS - https://github.com/dedis/kyber/tree/master/sign/bls

#### Encrypted/authenticated channels
Used for consensus algorithms that do not sign messages (but can also be
enabled for all algorithms).
- NaCl secretbox - https://godoc.org/golang.org/x/crypto/nacl/secretbox - 
the public/private keys of nodes are used to generate a Diffe Hellman shared secret
used as input to the functions.

#### Quantum-safe signatures
Used to sign consensus messages as an alternative to the normal signatures.
- libqos - https://github.com/open-quantum-safe/liboqs - Calls the go bindings
to this library (https://github.com/open-quantum-safe/liboqs-go). Note that this library
keeps changing so it may not work at all times.

#### Multisignatures
- BLS multisignatures - [bls](../consensus/auth/sig/bls/) - This is
implemented using the kyber library. Note that the kyber library
now  includes an implementation of BLS multisignatures, so should
transfer to using this. Note that the multisignatures implemented
there also include the hash of all the public keys signed, this is more
secure than the ones implemented here that just takes the hash of
individual public keys. This BitID implemtnations (the
BitIDs are structures used to track the signers of a message) would
have to be changed to track this.

### Threshold signatures
- Threshold-BLS - https://github.com/dedis/kyber/tree/master/sign/tbls - Threshold
key are generated centrally before the experiment then distributed
(i.e. no distributed key generation).

### Common coin
Used for asynchronous consensus implementations.
- Random Oracles in Constantinople: Practical Asynchronous Byzantine Agreement Using Cryptography -
[proof coin](../consensus/auth/sig/coinproof/) Implemented using existing libraries in Kyber
- Threshold-BLS common coin - common coin using the threshold signature

#### Verifiable Random Functions
Used for random membership selection.
- ECDSA P256 VRF - https://golang.org/pkg/crypto/ecdsa/
- BLS -  [bls](../consensus/auth/sig/bls/) - Here I use the BLS signature as the proof and hashed as the outputput as a VRF.
This is probably not a valid VRF implementation, and is just used for experimentation.

