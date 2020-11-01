package server

import (
	"fmt"
	"strings"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/sirupsen/logrus"

	neopb "neo/lib/genproto/neo"
)

const (
	stateBucketKey         = "states"
	configurationBucketKey = "configuration"
)

func NewBoltStorage(path string) (*CachedStorage, error) {
	db, err := bolt.Open(path, 0755, nil)
	if err != nil {
		return nil, err
	}
	return NewStorage(db)
}

func NewStorage(db *bolt.DB) (*CachedStorage, error) {
	cs := &CachedStorage{
		stateCache:  nil,
		configCache: nil,
		bdb:         db,
	}
	if err := cs.initDB(); err != nil {
		return nil, err
	}
	cs.initCache()
	return cs, nil
}

type CachedStorage struct {
	stateCache  map[string]*neopb.ExploitState
	configCache map[string]*neopb.ExploitConfiguration
	m           sync.RWMutex
	bdb         *bolt.DB
}

func (cs *CachedStorage) States() []*neopb.ExploitState {
	cs.m.RLock()
	defer cs.m.RUnlock()
	var res []*neopb.ExploitState
	for _, v := range cs.stateCache {
		res = append(res, v)
	}
	return res
}

func (cs *CachedStorage) State(exploitId string) (*neopb.ExploitState, bool) {
	cs.m.RLock()
	defer cs.m.RUnlock()
	val, ok := cs.stateCache[exploitId]
	return val, ok
}

func (cs *CachedStorage) Configuration(s *neopb.ExploitState) (*neopb.ExploitConfiguration, bool) {
	cs.m.RLock()
	defer cs.m.RUnlock()
	val, ok := cs.configCache[cs.configCacheKey(s)]
	return val, ok
}

func (cs *CachedStorage) UpdateExploitVersion(newState *neopb.ExploitState, cfg *neopb.ExploitConfiguration) error {
	cs.m.Lock()
	defer cs.m.Unlock()
	if state, ok := cs.stateCache[newState.ExploitId]; ok {
		newState.Version = state.Version + 1
	} else {
		newState.Version = 1
	}

	if err := cs.bdb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucketKey))
		key := []byte(fmt.Sprintf("%s:%d", newState.ExploitId, newState.Version))
		stateBytes, err := proto.Marshal(newState)
		if err != nil {
			return err
		}
		if err := b.Put(key, stateBytes); err != nil {
			return err
		}
		b = tx.Bucket([]byte(configurationBucketKey))
		confBytes, err := proto.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := b.Put(key, confBytes); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	cs.stateCache[newState.ExploitId] = newState
	cs.configCache[cs.configCacheKey(newState)] = cfg
	return nil
}

func (cs *CachedStorage) configCacheKey(s *neopb.ExploitState) string {
	return fmt.Sprintf("%s:%d", s.ExploitId, s.Version)
}

func (cs *CachedStorage) initCache() {
	cs.m.Lock()
	defer cs.m.Unlock()
	if cs.stateCache == nil || cs.configCache == nil {
		cs.stateCache = make(map[string]*neopb.ExploitState)
		cs.configCache = make(map[string]*neopb.ExploitConfiguration)
		if err := cs.readDB(); err != nil {
			logrus.Errorf("Failed to read exploit data from DB: %v", err)
		}
	}
}

func (cs *CachedStorage) readDB() error {
	return cs.bdb.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(stateBucketKey))
		err := b.ForEach(func(k, v []byte) error {
			key := string(k)
			eId := strings.Split(key, ":")[0]
			es := new(neopb.ExploitState)
			if err := proto.Unmarshal(v, es); err != nil {
				return err
			}
			if v, ok := cs.stateCache[eId]; !ok || es.Version > v.Version {
				cs.stateCache[eId] = es
			}
			return nil
		})
		b = tx.Bucket([]byte(configurationBucketKey))
		err = b.ForEach(func(k, v []byte) error {
			cfg := new(neopb.ExploitConfiguration)
			if err := proto.Unmarshal(v, cfg); err != nil {
				return err
			}
			cs.configCache[string(k)] = cfg
			return nil
		})
		return err
	})
}

func (cs *CachedStorage) initDB() error {
	return cs.bdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(stateBucketKey))
		_, err = tx.CreateBucketIfNotExists([]byte(configurationBucketKey))
		return err
	})
}