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
General configuration settings (see issue https://github.com/tcrain/cons/issues/8)
*/
package config

import (
	"encoding/binary"
	"os"
	"time"
)

type Logtype int

var PrintMinimum bool // if true the tests wont print the configs

func init() {
	if os.Getenv("PRINT_MIN") != "" {
		PrintMinimum = true
	}
}

const (
	GOLOG Logtype = iota // uses the default go logger
	GLOG                 // glog is no longer used because it is slow
	FMT                  // prints logs using fmt package
)

type LogFmtLevel int

const (
	LOGERROR LogFmtLevel = iota
	LOGWARNING
	LOGINFO
)

//panic("add random forward timeout")

const (
	// for logging
	LoggingType     = GOLOG
	LoggingFmtLevel = LOGERROR

	//Timeouts
	ForwardTimeout              = 500  // milliseconds 	// for msg forwarder when you dont receive enough messages to foward a buffer automatically
	ProgressTimeout             = 1000 // milliseconds, if no progress in this time, let neighbors know
	MvConsTimeout               = 1000 // millseconds timeout when taking an action in the MV consensus algorithms
	MvConsVRFTimeout            = 300  // millseconds timeout for waiting for a proposal when VRFs are enabled (only used by MVCons3)
	MvConsRequestRecoverTimeout = 500  // millseconds timeout before requesting the full proposal after delivering the hash

	// For the main channel
	Timeoutrecvms        = 500                  // number of miliseconds before timeout on receiving a message, before performing some recovery action, based on ProgressTimeout (ie this value should be smaller)
	InternalBuffSize     = 50                   // buffer of messages to process for the main thread loop
	MaxMsgReprocessCount = InternalBuffSize / 2 // number of messages queued to be reprocessed (should be smaller than InternalBuffSize)
	// SendBuffSize = 100 * InternalBuffSize
	RcvConUDPTimeout = 30000 // milliseconds if we don't hear from a rcv connection we close it
	RcvConUDPUpdate  = 5000  // milliseconds how often we send a keep alive for UDP conns

	// For behavior tracker
	RejectThreshold      = 10 // max errors caused by messages received per ForgiveTimeThreshold before dropping a connection
	ForgiveTimeThreshold = 100 * time.Millisecond
	DuplicateMax         = 1000 // max duplicate messages received per DuplicateTimeCount before dropping a connection
	DuplicateTimeCount   = 10 * time.Millisecond

	// For member checkers
	DefaultLocalRandMemberChange = 3 // On consensus index mod this value, the local rand member checker will change unless set to a different value in the test options.

	// network
	MaxMsgSize             = 100000000                                       // bytes any larger message is dropped
	MaxPacketSize          = 1000                                            // max bytes per piece that makeup a larger message, should be at least 4(?), note the packets sent will be slightly bigger as they will have some metadata
	MaxTotalPacketSize     = MaxPacketSize + (3 * binary.MaxVarintLen64) + 1 // packet plus meta-data
	UDPConnCount           = 10                                              // number of udp open ports to use at each node
	UDPMaxSendRate         = 100000                                          // maximum send rate of a single udp connection (in bytes/sec), set to 0 for no limit
	UDPMovingAvgWindowSize = 20                                              // how many previous messages are used for tracking the moving average msg send size, which is then used to caluculate how much to sleep to not exceed UDPMaxSendRate
	TCPDialTimeout         = 120000                                          // time in ms for TCP dial timeout
	RetryConnectionTimeout = 5000                                            // milliseconds beore retrying to connect to a node

	DefaultMsgProcesThreads = 10    // number of threads processing messages (deserialization/verification) before passing to the single consensus thread
	ConnectRetires          = 10    // number of times to retry TCP connections
	P2pGraphGenerateRetries = 10000 // number of times to retry building a random connected graph when using static p2p network

	// For tests
	AllowConcurrentTests = false // Allow this to run tests concurrently for the same package (it creates different storage files for each test) // TODO not currently supported because of globals
	ProcCount            = 5     // Total number of nodes
	NonMembers           = 1     // Number of nodes not participating in consensus (i.e. so here we have 10-2=8 consensus participants)
	MaxBenchRounds       = 10
	FanOut               = 4
	TestSleepValidate    = true // if sleeps are performed instead of singaure validations in the unit tests
	MaxRounds            = 10
	TestMsgSize          = 100000
	WarmUpInstances      = 10 // after this many instance are run statistics will be recorded
	//MaxSleepValidations = 1 // maximum number of sleeping signature validations (per consensus thread) that can take place concurrently (when SleepValidate is true)

	// TestPort = 6559
	RunAllTests = false // We dont run all tests because it takes to long
	LatencySend = 0     // messages will be send after this value in ms, to simulate nw delay
	PrintStats  = false // if true will print performances statistics after each test

	// For cons
	KeepFuture           = 20  // We will immediately process messages up to this amount past our index, if the memberchecker is ready TODO: may want to allow this to go to infinity? for MvCons3 since can have infinate consensus instances running before a decision is made
	KeepPast             = 10  // We will keep these states in memory up to this amount behind the index
	MaxRecovers          = 20  // Max number of recover indecies to send at once
	DropFuture           = 110 // We will drop any messages farther than this amount past the index
	MaxAdditionalIndices = 99  // Maximum number of additional indices a message can have.

	// For bin cons
	// DefaultPercentOnes = 50 // number of proposals that will be 1 vs 0 (randomly chosen) for testing
	// TODO BinDropFuture = 20 // Msgs from rounds further in the future than this will be dropped

	// For mv cons
	DefaultMvConsProposalBytes        = 100000 // size of the proposal for testing
	MvBroadcastInitForBufferForwarder = false  // only for when buffer forwarder is being used, if true, then the leader broadcast init to all nodes, instead of it being gossiped

	// signatures
	AllowMultiMerge                 = true  // allows multi sigs to merge the same signer multiple times TODO check if this is a valid thing to do
	StartWithNonConflictingMultiSig = true  // different way of computing the max sig size
	AllowSubMultSig                 = false // after generating a multi merge sig, use the list of sigs to see if you can remove some // TODO check if this removes too many
	StopMultiSigEarly               = false // for testing, will stop accepting new sigs once we reach n-t // TODO maybe this can be useful??

	// RPC
	ParRegRPCPort = 4534 // port for participant register
	RPCNodePort   = 4535 // port for rpc consensus node setup

	// CausalAsset
	SignCausalAssets = false // We don't need to sign assets since the outter message must be signed by the proposer.

	// TODO this is here because small number of nodes we might not get enough randoms, should not use normally though, howto fix this?
	DefaultCoordinatorRelaxtion = 10 // The percentage chance of each node announcing itself as a coordinator (the lowest wins) for when using VRF.
	DefaultNodeRelaxation       = 10 // Additional percentage chance a node will be chosen as a member for when using VRF.

	Thrshn, Thrsht   = 10, 7 // For threshold signature tests
	Thrshn2, Thrsht2 = 10, 4

	// for benchmarks
	BuildTablePDFs = false

	// For MvCons3
	// Number of instances in the future past the supporting instance that made the supported index commit for which membership
	// is fixed (when using random membership changed)
	FixedMemberFuture = 5

	// For MvCons4
	// When a slow node is recovering, send up to this many indices of events that the node is missing
	MvCons4MaxRecoverIndices = 2
	// When a slow node is recovering, send up to this many events that the node is missing
	MvCons4MaxRecoverEvents = 100
)

var Encoding = binary.LittleEndian // encoding for marshalling
