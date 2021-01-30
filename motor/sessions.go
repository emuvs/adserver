package motor

import (
	"net/http"
	"sync"
	"time"
)

const sessionHeaderName = "Adserver-Session-Uid"

type aSession struct {
	uid      string
	mapped   map[string]string
	store    *aStore
	mutex    sync.Mutex
	lastUsed time.Time
}

func (session *aSession) get(key string) string {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	return session.mapped[key]
}

func (session *aSession) set(key, value string) {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	session.mapped[key] = value
}

func (session *aSession) clear() {
	session.mutex.Lock()
	defer session.mutex.Unlock()
	for key := range session.mapped {
		delete(session.mapped, key)
	}
}

func (session *aSession) close() {
	closeStore(session)
}

var sessions = map[string]*aSession{}
var sessionsMutex = sync.Mutex{}

func newSession() *aSession {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	result := &aSession{
		uid:      randomString(32),
		mapped:   map[string]string{},
		store:    nil,
		mutex:    sync.Mutex{},
		lastUsed: time.Now(),
	}
	for sessions[result.uid] != nil {
		result.uid = randomString(32)
	}
	sessions[result.uid] = result
	return result
}

func createSession(w http.ResponseWriter, r *http.Request) *aSession {
	result := newSession()
	r.Header.Set(sessionHeaderName, result.uid)
	w.Header().Set(sessionHeaderName, result.uid)
	return result
}

func getSession(uid string) *aSession {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	result := sessions[uid]
	if result != nil {
		result.lastUsed = time.Now()
	}
	return result
}

func popSession(w http.ResponseWriter, r *http.Request) *aSession {
	sessionUID := r.Header.Get(sessionHeaderName)
	if sessionUID == "" {
		return createSession(w, r)
	}
	session := getSession(sessionUID)
	if session == nil {
		return createSession(w, r)
	}
	w.Header().Set(sessionHeaderName, session.uid)
	return session
}

func cleanSessions() {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	for uid, session := range sessions {
		elapsed := time.Since(session.lastUsed)
		if elapsed.Minutes() > 30 {
			session.clear()
			session.close()
			delete(sessions, uid)
		}
	}
}

func maintainSessions() {
	for true {
		time.Sleep(10 * time.Minute)
		cleanSessions()
	}
}
