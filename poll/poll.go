/*
Package poll manages passing a results channel to an async HTTP long-polling result request; and, optionally to
an async 'producing' results request. It can also be used for similar use cases such as passing a redirect 'return'
URL to a redirect request.

A long-polling request is passed its result via a results channel.

This channel is allocated by the request that initiated the async workflow that produces the result.
The NewState function allocates this channel as a property of a State and associates it with a type 4 UUID key.
The long-poll request path to receive the result is formed by appending this UUID as the last element of a long-poll
base path.

When the long-poll request is received, it retrieves its key's State using GetState.

The result may be produced by either a background gofunction or delivered by a 'producing' request.

If a background gofunction is used, it is passed the State whose channel it eventually uses to send the results
to the long-poll request.

If a producing request is used, its path is formed in the same way as the long-poll request path and it uses GetState
in the same way to retrieve its channel and send its results to the long-poll request.

States that are over 1 hour old are deleted from the states map.
*/
package poll

import (
	"strings"
	"sync"
	"time"

	"github.com/develrns/resilient/log"

	"github.com/pborman/uuid"
)

var logger = log.Logger()

func init() {
	go purgeTicker()
}

//purgeTicker purges abandoned States once per hour
func purgeTicker() {
	var ticker = time.NewTicker(time.Hour)

	for {
		_ = <-ticker.C
		States.purgeAbandonedStates()
	}

}

//states holds active long-poll states. Since many HTTP requests and gofunctions will be concurrently
//mutating a states table, it must be mutexed.
type states struct {
	m sync.Mutex
	s map[string]*State
}

//The States Table that holds all the long-poll channels for a server.
var States = newStates(1000)

//newStates allocates a states table
func newStates(capacity int) *states {
	var states states
	states.s = make(map[string]*State, capacity)
	return &states
}

//addState adds a state to the state table
func (ss *states) addState(state *State, key string) {
	ss.m.Lock()
	defer ss.m.Unlock()
	ss.s[key] = state
	return
}

//GetState retrieves a state from the States table.
//keyOrPath may be a key UUID or a URI path whose last element is the UUID.
func (ss *states) GetState(keyOrPath string) (*State, bool) {
	var (
		state    *State
		elements []string
		key      string
		ok       bool
	)

	//Extract key from keyOrPath
	elements = strings.Split(keyOrPath, "/")
	switch len(elements) {
	case 0:
		return nil, false
	case 1:
		key = keyOrPath
	default:
		key = elements[len(elements)-1]
	}

	//Lookup State by key
	ss.m.Lock()
	defer ss.m.Unlock()
	state, ok = ss.s[key]
	if !ok {
		return nil, false
	}
	return state, true
}

//delState deletes a state from the state table
func (ss *states) delState(key string) {
	ss.m.Lock()
	defer ss.m.Unlock()
	delete(ss.s, key)
	return
}

//purgeAbandonedStates deletes all State instances that are over an hour old from the States table.
//Note that a state and/or its channel may still be referenced by a producing/consuming gofunction after
//it has been removed from the States table. A common case will be that a producer will produce the result
//and exit. At that point, if the State for that results channel has been deleted from the States table the State and
//its channel will be garbage collected.
func (ss *states) purgeAbandonedStates() {
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
A State holds the result channel for sending an async result to an HTTP long-poll result request.
Done uses its key to remove it from the States table. purgeAbandonedStates uses its created time to
determine if a State has been abandoned.

State may be read concurrently. It must not be changed once it has been created.

In this scenario a channel that holds a single value is sufficient because only one send to the channel will be done.
*/
type State struct {
	C       chan interface{}
	Key     string
	created time.Time
}

/*
NewState creates a new State; puts it in the States table and returns it.
*/
func NewState() *State {
	var (
		key   = uuid.NewRandom().String()
		state State
	)
	state.C = make(chan interface{}, 1)
	state.Key = key
	state.created = time.Now()
	States.addState(&state, key)
	return &state
}

/*
Done deletes the State from the States table. Once a long-poll request has retrieved its results channel from a State,
it should call Done.
*/
func (s *State) Done() {
	States.delState(s.Key)
	return
}
