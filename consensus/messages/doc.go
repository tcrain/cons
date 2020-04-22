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
Package messages objects and methods for basic serialization of data into structured
messages that can be read by this project.

Integers are encoded using config.encoding
BigInts are encoded using GOB
Binary values are stored as a single byte, a 0 value byte is considered 0, any other value is considered 1

A struture of a message for the consensus is as follows:
  - A 4 byte encoded integer indicating the total size of the message in bytes (not including these 4 bytes)
  - A series of header/contents
    - A 4 byte integer indicating the size of the header (including these 4 bytes).
    - A 4 byte encoded integer indicating the header type.
    - If the header is signed, an encoded consensus id indicating the consensus index this header belongs to, which can be either:
      - An 8 byte encoded integer.
      - Or a hash (encoding defined by the hash type defined in the configuration).
    - An uint64 encoded as a uvarint indicating the number of additional consensus ids this header belongs to (if total order is being used this will be 0, see causal ordering for more details on where this is used).
    - The additional consensus indices as given by the previous field.
    - If the header contains at least 1 consensus index then a 32 byte unique ID for this consensus (see CsID) which is included in every signed header (TODO insure only messages with indices can be signed).

    - The contents are then defined custom for each header type

For signed headers, the signed part starts after the size and includes the consensus index, the consensus unique ID,
and any other contents as determined by the result of the call to InternalSignedMsgHeader.SerializeInternal for the
message type.

*/
package messages
