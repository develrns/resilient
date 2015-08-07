/*
Package poll manages long polling for async results.

The host name of a State's pollBaseURI must not be load balanced since its poll State is not available to any other service instances.

It is assumed that the a poll request and the code producing the result are executing as separate gofunctions. The poll state is used
by both to synchronize and send the result from the request execution gofunction to the poll request HTTP handler via an interface{} channel.
One poll state is used to handle the result synchronization for each request.
*/
package poll

import (
	"sync"
	"time"

	"github.com/develrns/resilient/log"

	"github.com/pborman/uuid"
)

var logger = log.Logger()

func init() {
	go purgeTicker()
}

//Purge poll states that have been abandoned
func purgeTicker() {
	var ticker = time.NewTicker(time.Hour)

	for {
		_ = <-ticker.C
		States.purgeAbandonedStates()
	}

}

type statesT struct {
	m sync.Mutex
	s map[string]*State
}

//Only one instance of the states table per server is required.
//States holds the active redirect states. Since many HTTP requests will be mutating this state table, it must be mutexed.
var States = newStates(1000)

func newStates(capacity int) *statesT {
	var states statesT
	states.s = make(map[string]*State, capacity)
	return &states
}

//addState adds a state to the state table
func (ss *statesT) addState(state *State, pollPath string) {
	ss.m.Lock()
	defer ss.m.Unlock()
	ss.s[pollPath] = state
	return
}

//State retrieves a state in the state table. This is used by a poll handler to wait for the arrival of the State's result.
func (ss *statesT) State(pollURI string) (*State, bool) {
	var (
		state *State
		ok    bool
	)
	ss.m.Lock()
	defer ss.m.Unlock()
	state, ok = ss.s[pollURI]
	if !ok {
		return nil, false
	}
	return state, true
}

//delState deletes a state from the state table
func (ss *statesT) delState(key string) {
	ss.m.Lock()
	defer ss.m.Unlock()
	delete(ss.s, key)
	return
}

//purgeAbandonedStates deletes abandoned states from the state table
func (ss *statesT) purgeAbandonedStates() {
	ss.m.Lock()
	defer ss.m.Unlock()
	for key, state := range ss.s {
		if time.Now().After(state.created.Add(time.Hour)) {
			delete(ss.s, key)
		}
	}
	return
}

/*
State manages the long polling between a requestor and a server.
The requestor uses an HTTP long poll to wait for the arrival of a request's result.
The server must provide an HTTP handler for receiving the poll GET requests to the request specific poll paths.
The server is this poll State.

The sequence of events is as follows:

1. Create a State giving it a base URI a SubjectDN identifying the poller. the requestor is given a 202 Accept response with a
Location header set to the pollURI.

2. When a long poll request is received, look up the state by calling States.State and receive its channel.
The request executor will send the result on this channel when it is available. When the result arrives, respond with it.
It is possible that that a poll will timeout while waiting for the state. If so, the requestor must resend it.
The poll request must not delete the state if its poll response fails; otherwise, call Done to remove the state.
*/
type State struct {
	C         chan interface{}
	subjectDN string
	pollPath  string
	created   time.Time
}

/*
NewState creates a new poll State; puts it in the server state table; and, returns the state and its pollPath.
*/
func NewState(pathBase, subjectDN string) (*State, string) {
	var (
		key   = uuid.NewRandom().String()
		state State
	)
	state.C = make(chan interface{})
	state.subjectDN = subjectDN
	state.pollPath = pathBase + key
	state.created = time.Now()
	States.addState(&state, state.pollPath)
	return &state, state.pollPath
}

/*
SubjectDN returns the Subject DN of the requestor for which this poll state was created.
*/
func (s *State) SubjectDN() string {
	return s.subjectDN
}

/*
Done deletes the state from the state table.
*/
func (s *State) Done() {
	States.delState(s.pollPath)
	return
}
