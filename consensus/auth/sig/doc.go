/*
github.com/tcrain/cons - Experimental project for testing and scaling consensus algorithms.
Copyright (C) 2020 The project authors - tcrain

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

*/
/*
This package implements different types of signing mechanisms.
Each type includes a public key type object, a private key type object, and a signature type object.
The public key objects and signature objects must implment the messages.MsgHeader interface, so they can
be serialized and sent over the network.

The follow signature types are supported:
    - ECDSA signatures using the go crypto library
    - Threshold signatures (this needs to be cleaned up, see TODO)
    - BLS signatures
    - BLS muti-signatures based on https://crypto.stanford.edu/~dabo/pubs/papers/BLSmultisig.html
    - EDDSA/SCHNORR signatures
*/
package sig
