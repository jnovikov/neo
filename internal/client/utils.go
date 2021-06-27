package client

import (
	"context"
	"errors"
	"io"

	"github.com/sirupsen/logrus"
)

func checkStreamError(tp string, err error, streamErr error) bool {
	if errors.Is(err, io.EOF) {
		logrus.Errorf("%s stream closed", tp)
		return false
	}
	if errors.Is(streamErr, context.Canceled) {
		logrus.Debugf("%s context cancelled", tp)
		return false
	}
	if err != nil {
		logrus.Errorf("Error reading from %s stream: %v", tp, err)
		return false
	}
	return true
}
