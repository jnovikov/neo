package hostbucket

import (
	"sync"

	"neo/pkg/rendezvous"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	neopb "neo/lib/genproto/neo"
)

func New(ips []string) *HostBucket {
	return &HostBucket{
		buck:  make(map[string]*neopb.TeamBucket),
		nodes: nil,
		ips:   ips,
		r:     rendezvous.New(),
	}
}

type HostBucket struct {
	m     sync.RWMutex
	buck  map[string]*neopb.TeamBucket
	nodes []*node
	ips   []string
	r     *rendezvous.Rendezvous
}

// TODO: effective ip addition & deletion
func (hb *HostBucket) UpdateIPS(ips []string) {
	lessFunc := func(s1, s2 string) bool {
		return s1 < s2
	}
	if !cmp.Equal(ips, hb.ips, cmpopts.SortSlices(lessFunc)) {
		hb.m.Lock()
		defer hb.m.Unlock()
		hb.ips = ips
		hb.rehash()
	}
}

func (hb *HostBucket) Buckets() map[string]*neopb.TeamBucket {
	hb.m.RLock()
	defer hb.m.RUnlock()
	return hb.buck
}

func (hb *HostBucket) Exists(id string) (exists bool) {
	hb.m.RLock()
	defer hb.m.RUnlock()
	_, exists = hb.buck[id]
	return
}

func (hb *HostBucket) AddNode(id string, weight int) {
	hb.m.Lock()
	defer hb.m.Unlock()

	if _, ok := hb.buck[id]; ok {
		return
	}

	hb.buck[id] = &neopb.TeamBucket{}
	n := &node{
		id:     id,
		weight: weight,
	}
	hb.nodes = append(hb.nodes, n)
	// TODO: more effective node addition
	hb.rehash()
}

func (hb *HostBucket) DeleteNode(id string) bool {
	hb.m.Lock()
	defer hb.m.Unlock()
	if _, ok := hb.buck[id]; !ok {
		return false
	}
	for i, n := range hb.nodes {
		if n.id == id {
			last := len(hb.nodes) - 1
			hb.nodes[i] = hb.nodes[last]
			hb.nodes[last] = nil
			hb.nodes = hb.nodes[:last]
			delete(hb.buck, id)

			// TODO: more effective node deletion
			hb.rehash()
			return true
		}
	}
	return false
}

func (hb *HostBucket) rehash() {
	for _, v := range hb.buck {
		v.Reset()
	}
	if len(hb.nodes) == 0 {
		return
	}
	for _, ip := range hb.ips {
		bestHash := 0.0
		bestNode := ""

		for _, n := range hb.nodes {
			hash := hb.r.Calculate(n.id, n.weight, ip)
			if bestNode == "" || hash > bestHash {
				bestNode = n.id
				bestHash = hash
			}
		}

		hb.buck[bestNode].TeamIps = append(hb.buck[bestNode].TeamIps, ip)
	}
}
