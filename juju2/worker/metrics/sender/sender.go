// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// Package sender contains the implementation of the metric
// sender manifold.
package sender

import (
	"fmt"
	"net"
	"path"
	"runtime"
	"time"

	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju2/api/metricsadder"
	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/worker/metrics/spool"
)

const (
	DefaultMetricsSendSocketName = "metrics-send.socket"
)

type stopper interface {
	Stop() error
}

type sender struct {
	client   metricsadder.MetricsAdderClient
	factory  spool.MetricFactory
	listener stopper
}

// Do sends metrics from the metric spool to the
// controller via an api call.
func (s *sender) Do(stop <-chan struct{}) error {
	reader, err := s.factory.Reader()
	if err != nil {
		return errors.Trace(err)
	}
	defer reader.Close()
	return s.sendMetrics(reader)
}

func (s *sender) sendMetrics(reader spool.MetricReader) error {
	batches, err := reader.Read()
	if err != nil {
		return errors.Annotate(err, "failed to open the metric reader")
	}
	var sendBatches []params.MetricBatchParam
	for _, batch := range batches {
		sendBatches = append(sendBatches, spool.APIMetricBatch(batch))
	}
	results, err := s.client.AddMetricBatches(sendBatches)
	if err != nil {
		return errors.Annotate(err, "could not send metrics")
	}
	for batchUUID, resultErr := range results {
		// if we fail to send any metric batch we log a warning with the assumption that
		// the unsent metric batches remain in the spool directory and will be sent to the
		// controller when the network partition is restored.
		if _, ok := resultErr.(*params.Error); ok || params.IsCodeAlreadyExists(resultErr) {
			err := reader.Remove(batchUUID)
			if err != nil {
				logger.Errorf("could not remove batch %q from spool: %v", batchUUID, err)
			}
		} else {
			logger.Errorf("failed to send batch %q: %v", batchUUID, resultErr)
		}
	}
	return nil
}

// Handle sends metrics from the spool directory to the
// controller.
func (s *sender) Handle(c net.Conn, _ <-chan struct{}) (err error) {
	defer func() {
		if err != nil {
			fmt.Fprintf(c, "%v\n", err)
		} else {
			fmt.Fprintf(c, "ok\n")
		}
		c.Close()
	}()
	// TODO(fwereade): 2016-03-17 lp:1558657
	if err := c.SetDeadline(time.Now().Add(spool.DefaultTimeout)); err != nil {
		return errors.Annotate(err, "failed to set the deadline")
	}
	reader, err := s.factory.Reader()
	if err != nil {
		return errors.Trace(err)
	}
	defer reader.Close()
	return s.sendMetrics(reader)
}

func (s *sender) stop() {
	if s.listener != nil {
		s.listener.Stop()
	}
}

var socketName = func(baseDir, unitTag string) string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf(`\\.\pipe\send-metrics-%s`, unitTag)
	default:
		return path.Join(baseDir, DefaultMetricsSendSocketName)
	}
}

func newSender(client metricsadder.MetricsAdderClient, factory spool.MetricFactory, baseDir, unitTag string) (*sender, error) {
	s := &sender{
		client:  client,
		factory: factory,
	}
	listener, err := spool.NewSocketListener(socketName(baseDir, unitTag), s)
	if err != nil {
		return nil, errors.Trace(err)
	}
	s.listener = listener
	return s, nil
}
