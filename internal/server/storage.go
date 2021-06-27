package server

import (
	"fmt"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	bolt "go.etcd.io/bbolt"

	neopb "neo/lib/genproto/neo"
)

const (
	stateBucketKey = "states"
)

func NewBoltStorage(path string) (*CachedStorage, error) {
	db, err := bolt.Open(path, 0755, nil)
	if err != nil {
		return nil, fmt.Errorf("opening bolt db %s: %w", path, err)
	}
	return NewStorage(db)
}

func NewStorage(db *bolt.DB) (*CachedStorage, error) {
	cs := &CachedStorage{
		stateCache: nil,
		bdb:        db,
	}
	if err := cs.initDB(); err != nil {
		return nil, err
	}
	cs.initCache()
	return cs, nil
}

type CachedStorage struct {
	stateCache map[string]*neopb.ExploitState
	m          sync.RWMutex
	bdb        *bolt.DB
}

func (cs *CachedStorage) States() []*neopb.ExploitState {
	cs.m.RLock()
	defer cs.m.RUnlock()
	res := make([]*neopb.ExploitState, 0, len(cs.stateCache))
	for _, v := range cs.stateCache {
		res = append(res, v)
	}
	return res
}

func (cs *CachedStorage) GetState(exploitID string) (*neopb.ExploitState, bool) {
	cs.m.RLock()
	defer cs.m.RUnlock()
	val, ok := cs.stateCache[exploitID]
	return val, ok
}

func (cs *CachedStorage) UpdateExploitVersion(newState *neopb.ExploitState) (*neopb.ExploitState, error) {
	cs.m.Lock()
	defer cs.m.Unlock()
	if state, ok := cs.stateCache[newState.ExploitId]; ok {
		newState.Version = state.Version + 1
	} else {
		newState.Version = 1
	}

	key := []byte(fmt.Sprintf("%s:%d", newState.ExploitId, newState.Version))
	stateBytes, err := proto.Marshal(newState)
	if err != nil {
		return nil, fmt.Errorf("marshalling state: %w", err)
	}

	if err := cs.bdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucketKey))
		if err := b.Put(key, stateBytes); err != nil {
			return fmt.Errorf("setting state in db: %w", err)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("updating db state: %w", err)
	}

	cs.stateCache[newState.ExploitId] = newState
	return newState, nil
}

func (cs *CachedStorage) initCache() {
	cs.m.Lock()
	defer cs.m.Unlock()
	if cs.stateCache == nil {
		cs.stateCache = make(map[string]*neopb.ExploitState)
		if err := cs.readDB(); err != nil {
			logrus.Errorf("Failed to read exploit data from DB: %v", err)
		}
	}
}

func (cs *CachedStorage) readDB() error {
	if err := cs.bdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucketKey))
		if err := b.ForEach(func(k, v []byte) error {
			key := string(k)
			eID := strings.Split(key, ":")[0]
			es := new(neopb.ExploitState)
			if err := proto.Unmarshal(v, es); err != nil {
				return fmt.Errorf("unmarshalling exploit state: %w", err)
			}
			if v, ok := cs.stateCache[eID]; !ok || es.Version > v.Version {
				cs.stateCache[eID] = es
			}
			return nil
		}); err != nil {
			return fmt.Errorf("reading exploit states: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("reading state from db: %w", err)
	}
	return nil
}

func (cs *CachedStorage) initDB() error {
	if err := cs.bdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(stateBucketKey)); err != nil {
			return fmt.Errorf("creating state bucket: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("initializing db: %w", err)
	}
	return nil
}
