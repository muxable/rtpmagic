package balancer

import (
	"math/rand"
	"sync"

	"github.com/muxable/rtpmagic/pkg/muxer/rtpnet"
)

type ConnectionMap struct {
	sync.RWMutex

	container     map[string]*rtpnet.CCWrapper
	keys          []string
	sliceKeyIndex map[string]int
}

func NewConnectionMap() *ConnectionMap {
	return &ConnectionMap{
		container:     make(map[string]*rtpnet.CCWrapper),
		sliceKeyIndex: make(map[string]int),
	}
}

// Set sets the key to a given connection.
func (m *ConnectionMap) Set(key string, conn *rtpnet.CCWrapper) {
	m.Lock()
	defer m.Unlock()

	m.container[key] = conn
	m.keys = append(m.keys, key)
	m.sliceKeyIndex[key] = len(m.keys) - 1
}

// Get gets the connection for a given key.
func (m *ConnectionMap) Get(key string) *rtpnet.CCWrapper {
	m.RLock()
	defer m.RUnlock()

	return m.container[key]
}

// Has checks if a key is in the map.
func (m *ConnectionMap) Has(key string) bool {
	m.RLock()
	defer m.RUnlock()

	_, ok := m.container[key]
	return ok
}

// Keys returns a slice of all keys.
func (m *ConnectionMap) Items() map[string]*rtpnet.CCWrapper {
	m.RLock()
	defer m.RUnlock()

	return m.container
}

// Remove removes the connection for a given key.
func (m *ConnectionMap) Remove(key string) {
	m.Lock()
	defer m.Unlock()

	index, ok := m.sliceKeyIndex[key]
	if !ok {
		return
	}

	delete(m.sliceKeyIndex, key)

	newLength := len(m.keys) - 1
	isLastIndex := newLength == index

	m.keys[index] = m.keys[newLength]
	m.keys = m.keys[:newLength]

	if !isLastIndex {
		m.sliceKeyIndex[m.keys[index]] = index
	}

	delete(m.container, key)
}

// Random returns a random key/value pair.
func (m *ConnectionMap) Random() (string, *rtpnet.CCWrapper) {
	m.RLock()
	defer m.RUnlock()

	if len(m.keys) == 0 {
		return "", nil
	}

	index := rand.Intn(len(m.keys))
	return m.keys[index], m.container[m.keys[index]]
}

// Close closes all the connections.
func (m *ConnectionMap) Close() error {
	m.Lock()
	defer m.Unlock()

	for _, conn := range m.container {
		if err := conn.Close(); err != nil {
			return err
		}
	}
	return nil
}
