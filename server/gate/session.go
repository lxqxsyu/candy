package gate

import (
	"time"

	"github.com/dearcode/candy/server/util/log"
)

const (
	stateOffline = iota
	stateOnline
)

type session struct {
	id    int64
	state int
	last  int64
	addr  string
}

func newSession(addr string) *session {
	log.Debugf("addr:%s", addr)
	return &session{addr: addr}
}

func (s *session) online(id int64) {
	log.Debugf("id:%v", id)
	s.state = stateOnline
	s.id = id
	s.last = time.Now().Unix()
}

func (s *session) offline() {
	log.Debugf("id:%v", s.id)
	s.state = stateOffline
}

func (s *session) update() {
	s.last = time.Now().Unix()
	log.Debugf("last:%v", s.last)
}

func (s *session) getAddr() string {
	log.Debugf("session getAddr addr:%v", s.addr)
	return s.addr
}

func (s *session) getID() int64 {
	log.Debugf("session getID id:%v", s.id)
	return s.id
}

func (s *session) isOnline() bool {
	log.Debugf("session isOnline flag:%v", s.state == stateOnline)
	return s.state == stateOnline
}
