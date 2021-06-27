package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sync"
	"time"

	"neo/pkg/pubsub"

	"neo/internal/config"
	"neo/pkg/filestream"
	"neo/pkg/hostbucket"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	neopb "neo/lib/genproto/neo"
)

const (
	broadcastChannel = "broadcast"
	singleRunChannel = "single_run"
)

var (
	ErrInvalidMessageType = errors.New("invalid message type")
)

var (
	noResponse = &neopb.Empty{}
)

type fileInterface interface {
	io.ReadWriteCloser
	Name() string
}

type filesystem interface {
	Create(string) (fileInterface, error)
	Open(string) (fileInterface, error)
}

type osFs struct {
	baseDir string
}

func (o osFs) Create(f string) (fileInterface, error) {
	fi, err := os.Create(path.Join(o.baseDir, f))
	if err != nil {
		return nil, fmt.Errorf("creating file %s in %s: %w", f, o.baseDir, err)
	}
	return fi, nil
}

func (o osFs) Open(f string) (fileInterface, error) {
	fi, err := os.Open(path.Join(o.baseDir, f))
	if err != nil {
		return nil, fmt.Errorf("opening file %s in %s: %w", f, o.baseDir, err)
	}
	return fi, nil
}

func New(cfg *Config, storage *CachedStorage) *ExploitManagerServer {
	ems := &ExploitManagerServer{
		storage: storage,
		fs:      osFs{cfg.BaseDir},
		buckets: hostbucket.New(cfg.FarmConfig.Teams),
		visits:  newVisitsMap(),
		ps:      pubsub.NewPubSub(),
	}
	ems.UpdateConfig(cfg)
	return ems
}

type ExploitManagerServer struct {
	neopb.UnimplementedExploitManagerServer
	storage  *CachedStorage
	config   *config.Config
	cfgMutex sync.RWMutex
	buckets  *hostbucket.HostBucket
	visits   *visitsMap
	fs       filesystem

	ps pubsub.PubSub
}

func (em *ExploitManagerServer) UpdateConfig(cfg *Config) {
	env := make([]string, 0, len(cfg.Environ))
	for k, v := range cfg.Environ {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	em.cfgMutex.Lock()
	defer em.cfgMutex.Unlock()
	em.config = &config.Config{
		PingEvery:    cfg.PingEvery,
		SubmitEvery:  cfg.SubmitEvery,
		FarmURL:      cfg.FarmConfig.URL,
		FarmPassword: cfg.FarmConfig.Password,
		FlagRegexp:   regexp.MustCompile(cfg.FarmConfig.FlagRegexp),
		Environ:      env,
	}
	em.buckets.UpdateTeams(cfg.FarmConfig.Teams)
}

func (em *ExploitManagerServer) Ping(_ context.Context, r *neopb.PingRequest) (*neopb.PingResponse, error) {
	logrus.Infof("Got %s from: %s", neopb.PingRequest_PingType_name[int32(r.GetType())], r.GetClientId())

	if r.Type == neopb.PingRequest_HEARTBEAT {
		em.cfgMutex.RLock()
		defer em.cfgMutex.RUnlock()
		em.visits.Add(r.GetClientId())
		em.buckets.AddNode(r.GetClientId(), int(r.GetWeight()))
	} else if r.Type == neopb.PingRequest_LEAVE {
		em.visits.MarkInvalid(r.GetClientId())
	}

	return &neopb.PingResponse{
		State: &neopb.ServerState{
			ClientTeamMap: em.buckets.Buckets(),
			Exploits:      em.storage.States(),
			Config:        config.ToProto(em.config),
		},
	}, nil
}

func (em *ExploitManagerServer) UploadFile(stream neopb.ExploitManager_UploadFileServer) error {
	info := &neopb.FileInfo{Uuid: uuid.New().String()}
	of, err := em.fs.Create(info.GetUuid())
	if err != nil {
		return logErrorf(codes.Internal, "Failed to create file: %v", err)
	}
	defer func() {
		if cerr := of.Close(); cerr != nil {
			err = logErrorf(codes.Internal, "Failed to close output file")
		}
		if err != nil {
			if rerr := os.Remove(of.Name()); rerr != nil {
				logrus.Errorf("Error removing the file on error: %v", err)
			}
		}
	}()

	if err := filestream.Save(stream, of); err != nil {
		return logErrorf(codes.Internal, "Failed to upload file from stream: %v", err)
	}
	if err := stream.SendAndClose(info); err != nil {
		return logErrorf(codes.Internal, "Failed to send response & close connection: %v", err)
	}
	return nil
}

func (em *ExploitManagerServer) DownloadFile(fi *neopb.FileInfo, stream neopb.ExploitManager_DownloadFileServer) error {
	f, err := em.fs.Open(fi.GetUuid())
	if err != nil {
		return logErrorf(codes.NotFound, "Failed to find file by uuid(%s): %v", fi.GetUuid(), err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Errorf("Error closing downloaded file: %v", err)
		}
	}()
	if err := filestream.Load(f, stream); err != nil {
		return logErrorf(codes.NotFound, "Failed to find file by uuid(%s): %v", fi.GetUuid(), err)
	}
	return nil
}

func (em *ExploitManagerServer) Exploit(_ context.Context, r *neopb.ExploitRequest) (*neopb.ExploitResponse, error) {
	state, ok := em.storage.GetState(r.GetExploitId())
	if !ok {
		return nil, logErrorf(codes.NotFound, "Failed to find an exploit state = %v", state.ExploitId)
	}
	return &neopb.ExploitResponse{
		State: state,
	}, nil
}

func (em *ExploitManagerServer) UpdateExploit(_ context.Context, r *neopb.UpdateExploitRequest) (*neopb.UpdateExploitResponse, error) {
	newState, err := em.storage.UpdateExploitVersion(r.GetState())
	if err != nil {
		return nil, logErrorf(codes.Internal, "Failed to update exploit version: %v", err)
	}
	return &neopb.UpdateExploitResponse{State: newState}, nil
}

func (em *ExploitManagerServer) BroadcastCommand(_ context.Context, r *neopb.Command) (*neopb.Empty, error) {
	logrus.Infof("Received broadcast request to run %v", r)
	em.ps.Publish(broadcastChannel, r)
	return noResponse, nil
}

func (em *ExploitManagerServer) BroadcastRequests(_ *neopb.Empty, stream neopb.ExploitManager_BroadcastRequestsServer) error {
	handler := func(msg interface{}) error {
		cmd, ok := msg.(*neopb.Command)
		if !ok {
			return ErrInvalidMessageType
		}
		if err := stream.Send(cmd); err != nil {
			return fmt.Errorf("sending command: %w", err)
		}
		return nil
	}
	sub := em.ps.Subscribe(broadcastChannel, handler)
	defer em.ps.Unsubscribe(sub)

	sub.Run(stream.Context())
	return nil
}

func (em *ExploitManagerServer) SingleRun(_ context.Context, r *neopb.SingleRunRequest) (*neopb.Empty, error) {
	logrus.Infof("Received single run request %v", r)
	em.ps.Publish(singleRunChannel, r)
	return noResponse, nil
}

func (em *ExploitManagerServer) SingleRunRequests(_ *neopb.Empty, stream neopb.ExploitManager_SingleRunRequestsServer) error {
	handler := func(msg interface{}) error {
		req, ok := msg.(*neopb.SingleRunRequest)
		if !ok {
			return ErrInvalidMessageType
		}
		if err := stream.Send(req); err != nil {
			return fmt.Errorf("sending single run request: %w", err)
		}
		return nil
	}
	sub := em.ps.Subscribe(singleRunChannel, handler)
	defer em.ps.Unsubscribe(sub)

	sub.Run(stream.Context())
	return nil
}

func (em *ExploitManagerServer) checkClients() {
	deadClients := em.visits.Invalidate(time.Now(), em.config.PingEvery)
	logrus.Infof("Heartbeat: got dead clients: %v", deadClients)
	for _, c := range deadClients {
		em.buckets.DeleteNode(c)
	}
}

func (em *ExploitManagerServer) HeartBeat(ctx context.Context) {
	ticker := time.NewTicker(em.config.PingEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			em.checkClients()
		case <-ctx.Done():
			return
		}
	}
}

func logErrorf(code codes.Code, fmt string, values ...interface{}) error {
	err := status.Errorf(code, fmt, values...)
	logrus.Errorf("%v", err)
	return err // nolint
}
