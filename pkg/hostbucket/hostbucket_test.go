package hostbucket

import (
	"math"
	"testing"

	"neo/pkg/testutils"

	"github.com/denisbrodbeck/machineid"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestHostBucket_Add(t *testing.T) {
	for _, tc := range []struct {
		b     *HostBucket
		users []string
		want  map[string]int
	}{
		{
			b:     New(map[string]string{"id1": "ip1"}),
			users: []string{"u1"},
			want:  map[string]int{"u1": 1},
		},
		{
			b:     New(map[string]string{"id1": "ip1", "id2": "ip2", "id3": "ip3"}),
			users: []string{},
			want:  map[string]int{},
		},
	} {
		for _, u := range tc.users {
			tc.b.AddNode(u, 1)
		}
		b := tc.b.Buckets()
		gotTeams := make(map[string]string)
		for cid, wantn := range tc.want {
			ipb := b[cid]
			teams := ipb.GetTeams()
			if len(teams) != wantn {
				t.Errorf("HostBucket.AddNode(): incorrent number of gotIps in user bucket want=%d, got=%d",
					wantn, len(ipb.GetTeams()),
				)
			}
			for k, v := range teams {
				gotTeams[k] = v
			}
		}
		less := func(s1, s2 string) bool {
			return s1 < s2
		}
		if diff := cmp.Diff(tc.b.teams, gotTeams, cmpopts.SortSlices(less)); diff != "" && len(tc.want) > 0 {
			t.Errorf("HostBucket.AddNode() summary ips mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestHostBucket_Add_Distribution(t *testing.T) {
	populate := func(idCount int, weightMax int) *HostBucket {
		teams := make(map[string]string)
		for id := range teams {
			teams[id] = testutils.RandomIP()
		}
		hb := New(teams)

		mid, err := machineid.ID()
		if err != nil {
			t.Fatalf("Could not get machine id: %v", err)
		}
		for i := 0; i < idCount; i++ {
			id := testutils.RandomString(len(mid))
			w := testutils.RandomInt(1, weightMax+1)
			hb.AddNode(id, w)
		}
		return hb
	}

	for _, tc := range []struct {
		ipCount       int
		idCount       int
		maxDeviation  float64
		maxStdDev     float64
		maxMeanStdDev float64
		weightMax     int
		runs          int
	}{
		{
			100,
			5,
			1.5,
			0.7,
			0.3,
			32,
			30,
		},
		{
			1000,
			10,
			1,
			0.5,
			0.2,
			32,
			30,
		},
	} {
		meanStdDev := 0.0
		for i := 0; i < tc.runs; i++ {
			b := populate(tc.idCount, tc.weightMax)
			sizes := make([]float64, tc.idCount)
			meanSize := 0.0
			for i := range sizes {
				id := b.nodes[i].id
				ips := b.buck[id].GetTeams()
				sizes[i] = float64(len(ips)) / float64(b.nodes[i].weight)
				meanSize += sizes[i]
			}
			meanSize /= float64(len(sizes))

			stdDev := 0.0
			for i := range sizes {
				id := b.nodes[i].id
				deviation := math.Abs((sizes[i] - meanSize) / meanSize)
				if deviation > tc.maxDeviation {
					t.Errorf(
						"Deviation for bucket %s too large: %f > %f, target size: %f, weight %d, got size: %f",
						id,
						deviation,
						tc.maxDeviation,
						meanSize,
						b.nodes[i].weight,
						sizes[i],
					)
				}
				curDev := math.Abs(sizes[i] - meanSize)
				stdDev += curDev * curDev
			}
			stdDev = math.Sqrt(stdDev/float64(len(sizes))) / meanSize
			if stdDev > tc.maxStdDev {
				t.Errorf("Std too large: %f > %f", stdDev, tc.maxStdDev)
			}
			meanStdDev += stdDev
		}
		meanStdDev /= float64(tc.runs)
		if meanStdDev > tc.maxMeanStdDev {
			t.Errorf("Mean std too large: %f > %f", meanStdDev, tc.maxMeanStdDev)
		}
		t.Logf("Mean std dev: %f", meanStdDev)
	}
}

func TestHostBucket_Balancing(t *testing.T) {
	populate := func(ipCount, idCount int) *HostBucket {
		teams := make(map[string]string, ipCount)
		for id := range teams {
			teams[id] = testutils.RandomIP()
		}
		hb := New(teams)

		mid, err := machineid.ID()
		if err != nil {
			t.Fatalf("Could not get machine id: %v", err)
		}
		for i := 0; i < idCount; i++ {
			id := testutils.RandomString(len(mid))
			hb.AddNode(id, 1)
		}
		return hb
	}

	for _, tc := range []struct {
		ipCount   int
		idCount   int
		maxMoved  float64
		cntAdd    int
		cntDelete int
	}{
		{
			100,
			5,
			0.3,
			0,
			1,
		},
		{
			1000,
			10,
			0.2,
			1,
			0,
		},
	} {
		b := populate(tc.ipCount, tc.idCount)

		beforeByIP := make(map[string]string)
		for _, n := range b.nodes {
			teams := b.buck[n.id].GetTeams()
			for _, ip := range teams {
				beforeByIP[ip] = n.id
			}
		}

		getCntMoved := func() int {
			cntMoved := 0
			for _, n := range b.nodes {
				teams := b.buck[n.id].GetTeams()
				for _, ip := range teams {
					if n.id != beforeByIP[ip] {
						cntMoved++
					}
					beforeByIP[ip] = n.id
				}
			}
			return cntMoved
		}

		for i := 0; i < tc.cntDelete; i++ {
			toDelete := testutils.RandomInt(0, tc.idCount)
			b.DeleteNode(b.nodes[toDelete].id)
		}
		cntMoved := getCntMoved()
		movedFraction := float64(cntMoved) / float64(tc.ipCount)
		if movedFraction > tc.maxMoved {
			t.Errorf("Too many ips moved after delete: %f%%, %d of %d", movedFraction*100, cntMoved, tc.ipCount)
		}

		mid, err := machineid.ID()
		if err != nil {
			t.Fatalf("Could not get machine id: %v", err)
		}
		for i := 0; i < tc.cntAdd; i++ {
			id := testutils.RandomString(len(mid))
			b.AddNode(id, 1)
		}
		cntMoved = getCntMoved()
		movedFraction = float64(cntMoved) / float64(tc.ipCount)
		if movedFraction > tc.maxMoved {
			t.Errorf("Too many ips moved after add: %f%%, %d of %d", movedFraction*100, cntMoved, tc.ipCount)
		}
	}
}

func TestHostBucket_Delete(t *testing.T) {
	populate := func(teams map[string]string, ids []string) *HostBucket {
		hb := New(teams)
		for _, id := range ids {
			hb.AddNode(id, 1)
		}
		return hb
	}
	for _, tc := range []struct {
		b      *HostBucket
		delete []string
		want   map[string]int
	}{
		{
			b:      populate(map[string]string{"team1": "ip1", "team2": "ip2"}, []string{"id1", "id2"}),
			delete: []string{"id1"},
			want:   map[string]int{"id2": 2},
		},
		{
			b:      populate(map[string]string{"team1": "ip1", "team2": "ip2"}, []string{"id1"}),
			delete: []string{"id1"},
			want:   map[string]int{},
		},
	} {
		for _, tid := range tc.delete {
			tc.b.DeleteNode(tid)
		}
		gotTeams := make(map[string]string)
		for cid, wantn := range tc.want {
			ipb := tc.b.buck[cid]
			teams := ipb.GetTeams()
			if len(teams) != wantn {
				t.Errorf("HostBucket.DeleteNode(): incorrent number of gotIps in user bucket want=%d, got=%d",
					wantn, len(teams),
				)
			}
			for k, v := range teams {
				gotTeams[k] = v
			}
		}
		if diff := cmp.Diff(tc.b.teams, gotTeams, cmpopts.SortSlices(testutils.LessString)); diff != "" && len(tc.want) > 0 {
			t.Errorf("HostBucket.DeleteNode() summary ips mismatch (-want +got):\n%s", diff)
		}
	}
}
