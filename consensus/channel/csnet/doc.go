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
This package contains the code for maintining and using TCP and UDP networking for the consensus.
In UDP a node may have multiple open ports being used for sending and receiving, in TCP there will be
one listening address that will spawn connections on other ports as usual.
All interaction with the consensus should take place through NetMainChannel (meaning the other objects should be
considered private (TODO maybe actually make private).

Each node should construct a NetMainChannel object.
For each external node that this node may receive messages from, NetMainChannel.AddExternal node should be called.
For each node that this node will be sending message to, NetMainChannel.CreateSendConnection should additionally be called.
The node will maintain send connections to at most the connCount input parameter to NewNetMainChannel, if NetMainChannel.CreateSendConnection
was called for more nodes, then these nodes will be connected to when the others can no longer be reached.
To send messages NetMainChannel.Send and NetMainChannel.SendTo should be used.
NetMainChannel.Close will close all the connections.
*/
package csnet
