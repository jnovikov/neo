package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"neo/internal/client"
	"neo/pkg/archive"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	neopb "neo/lib/genproto/neo"
)

type addCLI struct {
	*baseCLI
	path      string
	isArchive bool
	exploitID string
	runEvery  time.Duration
	timeout   time.Duration
	endless   bool
}

func NewAdd(cmd *cobra.Command, args []string, cfg *client.Config) NeoCLI {
	c := &addCLI{
		baseCLI: &baseCLI{cfg},
		path:    args[0],
	}

	var err error
	if c.exploitID, err = cmd.Flags().GetString("id"); err != nil {
		logrus.Fatalf("Could not get exploit id: %v", err)
	}
	if c.isArchive, err = cmd.Flags().GetBool("dir"); err != nil {
		logrus.Fatalf("Could not get parse directory: %v", err)
	}
	if c.runEvery, err = cmd.Flags().GetDuration("interval"); err != nil {
		logrus.Fatalf("Could not parse run interval: %v", err)
	}
	if c.timeout, err = cmd.Flags().GetDuration("timeout"); err != nil {
		logrus.Fatalf("Could not parse run timeout: %v", err)
	}
	if c.endless, err = cmd.Flags().GetBool("endless"); err != nil {
		logrus.Fatalf("Could not parse endless: %v", err)
	}
	return c
}

func (ac *addCLI) Run(ctx context.Context) error {
	_, err := os.Stat(ac.path)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", ac.path, err)
	}
	// Replace path with abs path.
	if ac.path, err = filepath.Abs(ac.path); err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if errs := ac.validateEntry(ac.path); len(errs) > 0 {
		for _, v := range errs {
			logrus.Errorf("%v", v)
		}
		return errors.New("invalid exploit")
	}

	dir, file := path.Split(ac.path)
	if ac.exploitID == "" {
		ac.exploitID = file
	}
	logrus.Infof("Going to add exploit with id = %s", ac.exploitID)

	c, err := ac.client()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	state, err := c.Ping(ctx, neopb.PingRequest_CONFIG_REQUEST)
	if err != nil {
		return fmt.Errorf("failed to get config from server: %w", err)
	}
	exists := false
	for _, v := range state.GetExploits() {
		if v.GetExploitId() == ac.exploitID {
			exists = true
			break
		}
	}

	if exists {
		fmt.Println("The exploit with this id already exists. Do you want to override (add new version) y/N ?")
		var tmp string
		if _, err := fmt.Scanln(&tmp); err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}
		if !strings.Contains(strings.ToLower(tmp), "y") {
			logrus.Fatalf("Aborted.")
		}
	}

	var f *os.File
	if ac.isArchive {
		f, err = ioutil.TempFile("", "ARCHIVE")
		if err != nil {
			return fmt.Errorf("failed to create tmpfile: %w", err)
		}

		defer func() {
			_ = os.Remove(f.Name())
		}()
		if err := archive.Tar(dir, f); err != nil {
			return fmt.Errorf("failed to create TarGz archive: %w", err)
		}

		// Seek file to start to correctly use it for reading.
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("failed to seek archive file: %w", err)
		}
	} else {
		f, err = os.Open(ac.path)
		if err != nil {
			return fmt.Errorf("failed to open exploit path: %w", err)
		}
	}
	defer func() {
		_ = f.Close()
	}()

	fileInfo, err := c.UploadFile(ctx, f)
	if err != nil {
		return fmt.Errorf("failed to upload exploit file: %w", err)
	}

	exState := &neopb.ExploitState{
		ExploitId: ac.exploitID,
		File:      fileInfo,
		Config: &neopb.ExploitConfiguration{
			Entrypoint: file,
			IsArchive:  ac.isArchive,
			RunEvery:   ac.runEvery.String(),
			Timeout:    ac.timeout.String(),
		},
		Endless:  ac.endless,
		Disabled: false,
	}
	newState, err := c.UpdateExploit(ctx, exState)
	if err != nil {
		return fmt.Errorf("failed to update exploit: %w", err)
	}
	logrus.Infof("Updated exploit state: %v", newState)
	return nil
}

func (ac *addCLI) validateEntry(f string) (errors []string) {
	data, err := ioutil.ReadFile(f)
	if err != nil {
		errors = append(errors, err.Error())
		return
	}
	if !isBinary(data) {
		if string(data[:2]) != "#!" {
			desc := fmt.Sprintf(
				"Please use shebang (e.g. %s) as the first line of your script",
				"#!/usr/bin/env python3",
			)
			errors = append(errors, desc)
		}

		// PYTHONUNBUFFERED=1 is set for python scripts, so no need to flush the buffer
		if !bytes.Contains(data, []byte("#!/usr/bin/env python")) {
			re := regexp.MustCompile(`(?m)flush[(=]`)
			if !re.Match(data) {
				desc := "Please flush the output, e.g. print(..., flush=True) in python"
				errors = append(errors, desc)
			}
		}
	}
	return
}
