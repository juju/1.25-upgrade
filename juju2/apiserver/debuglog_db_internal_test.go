// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"fmt"
	"time"

	"github.com/juju/loggo"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
	"gopkg.in/juju/names.v2"

	"github.com/juju/1.25-upgrade/juju2/apiserver/params"
	"github.com/juju/1.25-upgrade/juju2/state"
	coretesting "github.com/juju/1.25-upgrade/juju2/testing"
)

type debugLogDBIntSuite struct {
	coretesting.BaseSuite
	sock *fakeDebugLogSocket
}

var _ = gc.Suite(&debugLogDBIntSuite{})

func (s *debugLogDBIntSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.sock = newFakeDebugLogSocket()
}

func (s *debugLogDBIntSuite) TestParamConversion(c *gc.C) {
	t1 := time.Date(2016, 11, 30, 10, 51, 0, 0, time.UTC)
	reqParams := debugLogParams{
		fromTheStart:  false,
		noTail:        true,
		backlog:       11,
		startTime:     t1,
		filterLevel:   loggo.INFO,
		includeEntity: []string{"foo"},
		includeModule: []string{"bar"},
		excludeEntity: []string{"baz"},
		excludeModule: []string{"qux"},
	}

	called := false
	s.PatchValue(&newLogTailer, func(_ state.LogTailerState, params state.LogTailerParams) (state.LogTailer, error) {
		called = true

		// Start time will be used once the client is extended to send
		// time range arguments.
		c.Assert(params.StartTime, gc.Equals, t1)
		c.Assert(params.NoTail, jc.IsTrue)
		c.Assert(params.MinLevel, gc.Equals, loggo.INFO)
		c.Assert(params.InitialLines, gc.Equals, 11)
		c.Assert(params.IncludeEntity, jc.DeepEquals, []string{"foo"})
		c.Assert(params.IncludeModule, jc.DeepEquals, []string{"bar"})
		c.Assert(params.ExcludeEntity, jc.DeepEquals, []string{"baz"})
		c.Assert(params.ExcludeModule, jc.DeepEquals, []string{"qux"})

		return newFakeLogTailer(), nil
	})

	stop := make(chan struct{})
	close(stop) // Stop the request immediately.
	err := handleDebugLogDBRequest(nil, reqParams, s.sock, stop)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(called, jc.IsTrue)
}

func (s *debugLogDBIntSuite) TestParamConversionReplay(c *gc.C) {
	reqParams := debugLogParams{
		fromTheStart: true,
		backlog:      123,
	}

	called := false
	s.PatchValue(&newLogTailer, func(_ state.LogTailerState, params state.LogTailerParams) (state.LogTailer, error) {
		called = true

		c.Assert(params.StartTime.IsZero(), jc.IsTrue)
		c.Assert(params.InitialLines, gc.Equals, 0)

		return newFakeLogTailer(), nil
	})

	stop := make(chan struct{})
	close(stop) // Stop the request immediately.
	err := handleDebugLogDBRequest(nil, reqParams, s.sock, stop)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(called, jc.IsTrue)
}

func (s *debugLogDBIntSuite) TestFullRequest(c *gc.C) {
	// Set up a fake log tailer with a 2 log records ready to send.
	tailer := newFakeLogTailer()
	tailer.logsCh <- &state.LogRecord{
		Time:     time.Date(2015, 6, 19, 15, 34, 37, 0, time.UTC),
		Entity:   names.NewMachineTag("99"),
		Module:   "some.where",
		Location: "code.go:42",
		Level:    loggo.INFO,
		Message:  "stuff happened",
	}
	tailer.logsCh <- &state.LogRecord{
		Time:     time.Date(2015, 6, 19, 15, 36, 40, 0, time.UTC),
		Entity:   names.NewUnitTag("foo/2"),
		Module:   "else.where",
		Location: "go.go:22",
		Level:    loggo.ERROR,
		Message:  "whoops",
	}
	s.PatchValue(&newLogTailer, func(_ state.LogTailerState, params state.LogTailerParams) (state.LogTailer, error) {
		return tailer, nil
	})

	stop := make(chan struct{})
	done := s.runRequest(debugLogParams{}, stop)

	s.assertOutput(c, []string{
		"ok", // sendOk() call needs to happen first.
		"machine-99: 2015-06-19 15:34:37 INFO some.where code.go:42 stuff happened\n",
		"unit-foo-2: 2015-06-19 15:36:40 ERROR else.where go.go:22 whoops\n",
	})

	// Check the request stops when requested.
	close(stop)
	s.assertStops(c, done, tailer)
}

func (s *debugLogDBIntSuite) TestRequestStopsWhenTailerStops(c *gc.C) {
	tailer := newFakeLogTailer()
	s.PatchValue(&newLogTailer, func(_ state.LogTailerState, params state.LogTailerParams) (state.LogTailer, error) {
		close(tailer.logsCh) // make the request stop immediately
		return tailer, nil
	})

	err := handleDebugLogDBRequest(nil, debugLogParams{}, s.sock, nil)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(tailer.stopped, jc.IsTrue)
}

func (s *debugLogDBIntSuite) TestMaxLines(c *gc.C) {
	// Set up a fake log tailer with a 5 log records ready to send.
	tailer := newFakeLogTailer()
	for i := 0; i < 5; i++ {
		tailer.logsCh <- &state.LogRecord{
			Time:     time.Date(2015, 6, 19, 15, 34, 37, 0, time.UTC),
			Entity:   names.NewMachineTag("99"),
			Module:   "some.where",
			Location: "code.go:42",
			Level:    loggo.INFO,
			Message:  "stuff happened",
		}
	}
	s.PatchValue(&newLogTailer, func(_ state.LogTailerState, params state.LogTailerParams) (state.LogTailer, error) {
		return tailer, nil
	})

	done := s.runRequest(debugLogParams{maxLines: 3}, nil)

	s.assertOutput(c, []string{
		"ok", // sendOk() call needs to happen first.
		"machine-99: 2015-06-19 15:34:37 INFO some.where code.go:42 stuff happened\n",
		"machine-99: 2015-06-19 15:34:37 INFO some.where code.go:42 stuff happened\n",
		"machine-99: 2015-06-19 15:34:37 INFO some.where code.go:42 stuff happened\n",
	})

	// The tailer should now stop by itself after the line limit was reached.
	s.assertStops(c, done, tailer)
}

func (s *debugLogDBIntSuite) runRequest(params debugLogParams, stop chan struct{}) chan error {
	done := make(chan error)
	go func() {
		done <- handleDebugLogDBRequest(&fakeState{}, params, s.sock, stop)
	}()
	return done
}

func (s *debugLogDBIntSuite) assertOutput(c *gc.C, expectedWrites []string) {
	timeout := time.After(coretesting.LongWait)
	for i, expectedWrite := range expectedWrites {
		select {
		case actualWrite := <-s.sock.writes:
			c.Assert(actualWrite, gc.Equals, expectedWrite)
		case <-timeout:
			c.Fatalf("timed out waiting for socket write (received %d)", i)
		}
	}
}

func (s *debugLogDBIntSuite) assertStops(c *gc.C, done chan error, tailer *fakeLogTailer) {
	select {
	case err := <-done:
		c.Assert(err, jc.ErrorIsNil)
		c.Assert(tailer.stopped, jc.IsTrue)
	case <-time.After(coretesting.LongWait):
		c.Fatal("timed out waiting for request handler to stop")
	}
}

type fakeState struct {
	state.LogTailerState
}

func newFakeLogTailer() *fakeLogTailer {
	return &fakeLogTailer{
		logsCh: make(chan *state.LogRecord, 10),
	}
}

type fakeLogTailer struct {
	state.LogTailer
	logsCh  chan *state.LogRecord
	stopped bool
}

func (t *fakeLogTailer) Logs() <-chan *state.LogRecord {
	return t.logsCh
}

func (t *fakeLogTailer) Stop() error {
	t.stopped = true
	return nil
}

func (t *fakeLogTailer) Err() error {
	return nil
}

func newFakeDebugLogSocket() *fakeDebugLogSocket {
	return &fakeDebugLogSocket{
		writes: make(chan string, 10),
	}
}

type fakeDebugLogSocket struct {
	writes chan string
}

func (s *fakeDebugLogSocket) sendOk() {
	s.writes <- "ok"
}

func (s *fakeDebugLogSocket) sendError(err error) {
	s.writes <- fmt.Sprintf("err: %v", err)
}

func (s *fakeDebugLogSocket) sendLogRecord(r *params.LogMessage) error {
	s.writes <- fmt.Sprintf("%s: %s %s %s %s %s\n",
		r.Entity,
		s.formatTime(r.Timestamp),
		r.Severity,
		r.Module,
		r.Location,
		r.Message)
	return nil
}

func (c *fakeDebugLogSocket) formatTime(t time.Time) string {
	return t.In(time.UTC).Format("2006-01-02 15:04:05")
}
