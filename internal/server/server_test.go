package server

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"neo/pkg/hostbucket"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	neopb "neo/lib/genproto/neo"
)

func testServer() (*ExploitManagerServer, func()) {
	db, cleanupDB := testDB()
	st, err := NewStorage(db)
	if err != nil {
		panic(err)
	}
	dir, err := ioutil.TempDir("", "server_test")
	if err != nil {
		panic(err)
	}
	es := New(&Config{
		BaseDir: dir,
	}, st)
	return es, func() {
		cleanupDB()
		if err := os.RemoveAll(dir); err != nil {
			panic(err)
		}
	}
}

func TestExploitManagerServer_UpdateExploit(t *testing.T) {
	es, clean := testServer()
	defer clean()
	cfg := &neopb.ExploitConfiguration{
		Entrypoint: "bin",
		IsArchive:  false,
	}
	r := &neopb.UpdateExploitRequest{
		State: &neopb.ExploitState{
			ExploitId: "1",
			File:      &neopb.FileInfo{Uuid: "1"},
			Config:    cfg,
		},
	}
	resp, err := es.UpdateExploit(context.Background(), r)
	if err != nil {
		t.Fatalf("UpdateExploit() failed with unexpected error = %v", err)
	}
	want := &neopb.ExploitState{
		ExploitId: "1",
		Version:   1,
		File:      r.GetState().GetFile(),
		Config:    cfg,
	}
	if diff := cmp.Diff(want, resp.GetState(), protocmp.Transform()); diff != "" {
		t.Errorf("UpdateExploit() mismatch (-want +got):\n%s", diff)
	}
}

func TestExploitManagerServer_Exploit(t *testing.T) {
	es, clean := testServer()
	defer clean()
	cfg := &neopb.ExploitConfiguration{
		Entrypoint: "bin",
		IsArchive:  false,
	}
	state := &neopb.ExploitState{
		ExploitId: "1",
		File:      &neopb.FileInfo{Uuid: "1"},
		Config:    cfg,
	}
	req := &neopb.UpdateExploitRequest{State: state}
	ctx := context.Background()
	_, err := es.UpdateExploit(ctx, req)
	if err != nil {
		t.Fatalf("UpdateExploit() failed with unexpected error = %v", err)
	}
	resp, err := es.Exploit(ctx, &neopb.ExploitRequest{ExploitId: state.ExploitId})
	if err != nil {
		t.Fatalf("Exploit() failed with unexpected error = %v", err)
	}
	wantState := &neopb.ExploitState{
		ExploitId: "1",
		Version:   1,
		File:      state.GetFile(),
		Config:    cfg,
	}
	if diff := cmp.Diff(wantState, resp.GetState(), protocmp.Transform()); diff != "" {
		t.Errorf("Exploit() state mismatch (-want +got):\n%s", diff)
	}
}

func TestExploitManagerServer_Ping(t *testing.T) {
	es, clean := testServer()
	defer clean()
	es.buckets = hostbucket.New(map[string]string{"id1": "ip1", "id2": "ip2"})
	es.config.FarmURL = "test"
	ctx := context.Background()
	cfg := &neopb.ExploitConfiguration{
		Entrypoint: "bin",
		IsArchive:  false,
	}
	state := &neopb.ExploitState{
		ExploitId: "1",
		File:      &neopb.FileInfo{Uuid: "1"},
		Config:    cfg,
	}
	r := &neopb.UpdateExploitRequest{State: state}
	updateResp, err := es.UpdateExploit(ctx, r)
	if err != nil {
		t.Fatalf("UpdateExploit(): unexpected error = %v", err)
	}

	req := &neopb.PingRequest{ClientId: "id1", Type: neopb.PingRequest_HEARTBEAT}
	resp, err := es.Ping(ctx, req)
	if err != nil {
		t.Fatalf("Ping(): unexpected error = %v", err)
	}
	want := []*neopb.ExploitState{updateResp.GetState()}
	if diff := cmp.Diff(want, resp.GetState().GetExploits(), protocmp.Transform()); diff != "" {
		t.Errorf("Ping() states mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(es.buckets.Buckets(), resp.GetState().GetClientTeamMap(), protocmp.Transform()); diff != "" {
		t.Errorf("Ping() bucket mismatch (-want +got):\n%s", diff)
	}
	if len(es.buckets.Buckets()[req.ClientId].GetTeams()) == 0 {
		t.Errorf("Ping() ip bucket with zero len")
	}
	gotURL := resp.GetState().GetConfig().GetFarmUrl()
	if es.config.FarmURL != gotURL {
		t.Errorf("Ping() config mismatch want farmURL: %s, got: %s", es.config.FarmURL, gotURL)
	}
	if !es.visits.visits["id1"].Before(time.Now()) {
		t.Errorf("Ping() visits missmatch")
	}
}

func TestExploitManagerServer_BroadcastCommand(t *testing.T) {
	es, clean := testServer()
	defer clean()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received *neopb.Command
	signal := make(chan struct{})
	handler := func(msg interface{}) error {
		cmd, ok := msg.(*neopb.Command)
		if !ok {
			t.Errorf("Invalid message passed to handler: %v", msg)
		}
		received = cmd
		close(signal)
		return nil
	}
	testSub := es.ps.Subscribe(broadcastChannel, handler)
	defer es.ps.Unsubscribe(testSub)
	go testSub.Run(ctx)

	r := &neopb.Command{Command: "echo 123"}
	_, err := es.BroadcastCommand(ctx, r)
	if err != nil {
		t.Errorf("Error received from broadcast command: %v", err)
	}

	select {
	case <-signal:
		break
	case <-time.After(time.Millisecond * 100):
		t.Errorf("Handler was not called in time")
	}

	if received != r {
		t.Errorf("Received incorrect command: expected = %v, actual = %v", r, received)
	}
}
