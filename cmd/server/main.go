package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/signal"

	"neo/internal/server"
	"neo/pkg/grpcauth"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	neopb "neo/lib/genproto/neo"
)

var (
	configFile = pflag.StringP("config", "c", "client_config.yml", "yaml config file")
)

func load(path string, cfg *server.Config) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", path, err)
	}
	if err := server.ReadConfig(data, cfg); err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	return nil
}

func watchConfig(ctx context.Context, srv *server.ExploitManagerServer) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating file watcher: %w", err)
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				err = watcher.Close()
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
					cfg := &server.Config{}
					err := load(*configFile, cfg)
					if err != nil {
						logrus.Errorf("Failed to reload read configuration: %v", err)
					} else {
						logrus.Infof("Reloaded config: %v", cfg)
						srv.UpdateConfig(cfg)
					}
				}
			}
		}
	}()
	if err := watcher.Add(*configFile); err != nil {
		return fmt.Errorf("adding watched file %s: %w", *configFile, err)
	}
	return nil
}

func main() {
	pflag.Parse()
	cfg := &server.Config{}
	if err := load(*configFile, cfg); err != nil {
		logrus.Fatalf("Failed to read config: %v", err)
	}

	ctx := context.Background()
	fc := server.NewFarmClient(cfg.FarmConfig)
	if err := fc.FillConfig(ctx, &cfg.FarmConfig); err != nil {
		logrus.Fatalf("Failed to fetch config from farm: %v", err)
	}

	st, err := server.NewBoltStorage(cfg.DBPath)
	if err != nil {
		logrus.Fatalf("Failed to create bolt storage: %v", err)
	}
	if cfg.PingEvery <= 0 {
		logrus.Fatalf("ping_every should be positive")
	}
	logrus.Infof("Config: %+v", cfg)
	srv := server.New(cfg, st)
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		logrus.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if cfg.GrpcAuthKey != "" {
		authInterceptor := grpcauth.NewServerInterceptor(cfg.GrpcAuthKey)
		opts = append(opts, grpc.UnaryInterceptor(authInterceptor.Unary()))
		opts = append(opts, grpc.StreamInterceptor(authInterceptor.Stream()))
	}
	s := grpc.NewServer(opts...)
	neopb.RegisterExploitManagerServer(s, srv)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	if err := watchConfig(ctx, srv); err != nil {
		logrus.Errorf("Failed to start config auto-reload: %v", err)
	}
	go srv.HeartBeat(ctx)
	go func() {
		<-c
		logrus.Info("Received shutdown signal, stopping server")
		cancel()
		s.GracefulStop()
	}()
	logrus.Infof("Starting server on port %s", cfg.Port)
	if err := s.Serve(lis); err != nil {
		logrus.Fatalf("failed to serve: %v", err)
	}
}
