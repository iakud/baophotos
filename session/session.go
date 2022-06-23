package session

import (
	"container/list"
	"encoding/base64"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	kMaxLeftTime int    = 300
	kCookieName  string = "baophotos"
)

var defaultManager *Manager

func Start(w http.ResponseWriter, r *http.Request) *Session {
	return defaultManager.SessionStart(w, r)
}

func Destroy(w http.ResponseWriter, r *http.Request) {
	defaultManager.SessionDestroy(w, r)
}

func init() {
	defaultManager = NewManager(kCookieName, kMaxLeftTime)
}

type Session struct {
	manager      *Manager
	id           string
	timeAccessed time.Time
	value        map[interface{}]interface{}
}

func newSession(manager *Manager, sessionId string) *Session {
	s := &Session{
		manager:      manager,
		id:           sessionId,
		timeAccessed: time.Now(),
		value:        make(map[interface{}]interface{}),
	}
	return s
}

func (s *Session) ID() string {
	return s.id
}

func (s *Session) Set(key, value interface{}) error {
	s.value[key] = value
	s.manager.updateSession(s.id)
	return nil
}

func (s *Session) Get(key interface{}) interface{} {
	if v, ok := s.value[key]; ok {
		s.manager.updateSession(s.id)
		return v
	}
	return nil
}

func (s *Session) Delete(key interface{}) error {
	delete(s.value, key)
	s.manager.updateSession(s.id)
	return nil
}

type Manager struct {
	cookieName  string
	lock        sync.Mutex
	sessions    map[string]*list.Element
	list        *list.List
	maxLifeTime int
}

func NewManager(cookieName string, maxLifeTime int) *Manager {
	m := &Manager{
		cookieName:  cookieName,
		maxLifeTime: maxLifeTime,
		sessions:    make(map[string]*list.Element),
		list:        list.New(),
	}
	go m.sessionWatcher()
	return m
}

func (m *Manager) sessionId() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (m *Manager) SessionStart(w http.ResponseWriter, r *http.Request) *Session {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		sessionId := m.sessionId()
		session := m.newSession(sessionId)
		cookie := http.Cookie{
			Name:     m.cookieName,
			Value:    url.QueryEscape(sessionId),
			Path:     "/",
			HttpOnly: true,
			MaxAge:   m.maxLifeTime,
		}
		http.SetCookie(w, &cookie)
		return session
	}
	sessionId, _ := url.QueryUnescape(cookie.Value)
	return m.getSession(sessionId)
}

func (m *Manager) SessionDestroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || cookie.Value == "" {
		return
	}
	m.removeSession(cookie.Value)
	newCookie := http.Cookie{
		Name:     m.cookieName,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now(),
		MaxAge:   -1,
	}
	http.SetCookie(w, &newCookie)
}

func (m *Manager) newSession(sessionId string) *Session {
	m.lock.Lock()
	defer m.lock.Unlock()
	session := newSession(m, sessionId)
	element := m.list.PushBack(session)
	m.sessions[sessionId] = element
	return session
}

func (m *Manager) getSession(sessionId string) *Session {
	m.lock.Lock()
	defer m.lock.Unlock()
	if element, ok := m.sessions[sessionId]; ok {
		return element.Value.(*Session)
	}
	session := newSession(m, sessionId)
	element := m.list.PushBack(session)
	m.sessions[sessionId] = element
	return session
}

// 删除session
func (m *Manager) removeSession(sessionId string) {
	m.lock.Lock()
	defer m.lock.Lock()
	if element, ok := m.sessions[sessionId]; ok {
		delete(m.sessions, sessionId)
		m.list.Remove(element)
	}
}

func (m *Manager) updateSession(sessionId string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if element, ok := m.sessions[sessionId]; ok {
		element.Value.(*Session).timeAccessed = time.Now()
		m.list.MoveToFront(element)
	}
}

func (m *Manager) sessionWatcher() {
	ticker := time.NewTicker(time.Duration(m.maxLifeTime) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			m.cleanSession(now)
		}
	}
}

func (m *Manager) cleanSession(now time.Time) {
	liftTime := now.Add(-time.Duration(m.maxLifeTime) * time.Second)
	m.lock.Lock()
	defer m.lock.Unlock()

	for {
		element := m.list.Back()
		if element == nil {
			break
		}
		session := element.Value.(*Session)
		if session.timeAccessed.After(liftTime) {
			break
		}
		m.list.Remove(element)
		delete(m.sessions, session.id)
	}
}
