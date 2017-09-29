package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/ssh"
	"golang.org/x/sync/errgroup"
)

const systemIdentity = "/var/lib/juju/system-identity"

type execOptions struct {
	ssh.Options
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
	hostAddr string
}

type execOption func(*execOptions)

func withSystemIdentity() execOption {
	return withIdentity(systemIdentity)
}

func withIdentity(identity string) execOption {
	return func(opts *execOptions) {
		opts.SetIdentities(identity)
	}
}

func withStdin(r io.Reader) execOption {
	return func(opts *execOptions) {
		opts.stdin = r
	}
}

func withStdout(w io.Writer) execOption {
	return func(opts *execOptions) {
		opts.stdout = w
	}
}

func withStderr(w io.Writer) execOption {
	return func(opts *execOptions) {
		opts.stderr = w
	}
}

func makeProxyCommand(hostAddr string) []string {
	return []string{"ssh", "-q",
		"-i", systemIdentity,
		"-o", "StrictHostKeyChecking no",
		"-o UserKnownHostsFile /dev/null",
		"ubuntu@" + hostAddr,
		"nc %h %p",
	}
}

// withProxyCommandForHost returns an option decorator for setting
// an SSH proxy command to proxy through the given host.
func withProxyCommandForHost(hostAddr string) execOption {
	return func(opts *execOptions) {
		opts.hostAddr = hostAddr
		opts.SetProxyCommand(makeProxyCommand(hostAddr)...)
	}
}

func defaultSSHOptions() ssh.Options {
	var options ssh.Options
	// Strict host key checking must be disabled because Juju 1.25 did not
	// populate SSH host keys.
	options.SetStrictHostKeyChecking(ssh.StrictHostChecksNo)
	options.SetKnownHostsFile(os.DevNull)
	return options
}

// runViaSSH runs script in the remote machine with address addr.
func runViaSSH(addr, script string, opts ...execOption) (int, error) {
	options := execOptions{Options: defaultSSHOptions()}
	options.stdout = os.Stdout
	options.stderr = os.Stderr
	options.hostAddr = addr
	for _, opt := range opts {
		opt(&options)
	}

	// Throttle on the host address if we're proxying.
	throttler.Acquire(options.hostAddr)
	defer throttler.Release(options.hostAddr)

	// This is taken from cmd/juju/ssh.go there is no other clear way to set user
	userAddr := "ubuntu@" + addr

	userCmd := ssh.Command(
		userAddr,
		[]string{"sudo", "-n", "bash", "-c " + utils.ShQuote(script)},
		&options.Options,
	)
	userCmd.Stdin = options.stdin
	userCmd.Stdout = options.stdout
	userCmd.Stderr = options.stderr

	// logger.Debugf("executing %s, script:\n%s", addr, script)
	if err := userCmd.Run(); err != nil {
		if rc, ok := err.(*cmd.RcPassthroughError); ok {
			return rc.Code, nil
		} else {
			return -1, errors.Trace(err)
		}
	}
	return 0, nil
}

type FlatMachine struct {
	Model      string
	Series     string
	ID         string
	InstanceID string
	Address    string
	Tools      string

	// HostAddress, if non-empty, is the address of the
	// host machine that contains this machine. If this
	// is set, it implies the machine is a container.
	HostAddress string
}

func flatMachineExecTargets(machines ...FlatMachine) []execTarget {
	targets := make([]execTarget, len(machines))
	for i, m := range machines {
		targets[i] = execTarget{
			m.Address,
			m.HostAddress,
		}
	}
	return targets
}

type execTarget struct {
	addr     string
	hostAddr string
}

type execResult struct {
	Code   int
	Stdout string
	Stderr string
}

const maxPerHost = 5
const acquireTimeout = 5 * time.Minute

// hostThrottler prevents us from trying to open too many ssh
// connections to a host (especially for proxied connections to
// containers).
type hostThrottler struct {
	mu    sync.Mutex
	chans map[string]chan struct{}
}

func newHostThrottler() *hostThrottler {
	return &hostThrottler{chans: make(map[string]chan struct{})}
}

func (t *hostThrottler) getChan(address string) chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	hostChan, found := t.chans[address]
	if !found {
		hostChan = make(chan struct{}, maxPerHost)
		for i := 0; i < maxPerHost; i++ {
			hostChan <- struct{}{}
		}
		t.chans[address] = hostChan
	}
	return hostChan
}

func (t *hostThrottler) Acquire(address string) {
	hostChan := t.getChan(address)
	select {
	case <-hostChan:
		return
	case <-time.After(acquireTimeout):
		panic(fmt.Sprintf("timed out waiting for SSH throttling to %q - missing Release?", address))
	}
}

func (t *hostThrottler) Release(address string) {
	hostChan := t.getChan(address)
	select {
	case hostChan <- struct{}{}:
		return
	default:
		panic(fmt.Sprintf("too many releases for %q - missing Acquire?", address))
	}
}

var throttler = newHostThrottler()

// parallelExec executes a script on each of the given targets,
// and returns their results. An error is only returned if any
// of the the SSH executions cannot proceed; if the execution
// proceeds, but the specified script fails, an error will not
// be returned; the exit code and output will be captured in
// the results.
func parallelExec(targets []execTarget, script string) ([]execResult, error) {
	results := make([]execResult, len(targets))
	var group errgroup.Group
	for i, target := range targets {
		i, target := i, target // copy for closure
		group.Go(func() error {
			var stdoutBuf bytes.Buffer
			var stderrBuf bytes.Buffer
			opts := []execOption{
				withSystemIdentity(),
				withStdout(&stdoutBuf),
				withStderr(&stderrBuf),
			}
			if target.hostAddr != "" {
				// This is a container; proxy through
				// the host machine.
				opts = append(opts, withProxyCommandForHost(target.hostAddr))
			}
			rc, err := runViaSSH(target.addr, script, opts...)
			if err != nil {
				return err
			}
			results[i] = execResult{
				Code:   rc,
				Stdout: stdoutBuf.String(),
				Stderr: stderrBuf.String(),
			}
			return nil
		})
	}
	return results, group.Wait()
}

// prefixWriter is an implementation of io.Writer, which prefixes each line
// written with a given string.
type prefixWriter struct {
	io.Writer
	prefix  string
	started bool
}

// Write is part of the io.Writer interfaces.
func (w *prefixWriter) Write(data []byte) (int, error) {
	ndata := len(data)
	for len(data) > 0 {
		if !w.started {
			if _, err := io.WriteString(w.Writer, w.prefix); err != nil {
				return -1, err
			}
			w.started = true
		}
		i := bytes.IndexRune(data, '\n')
		if i >= 0 {
			// There's a newline, so write out the line to the
			// underlying writer, and clear w.started.
			w.started = false
		} else {
			// No more newlines, just write out the remainder
			// of the data.
			i = len(data) - 1
		}
		n, err := w.Writer.Write(data[:i+1])
		if err != nil {
			return n, err
		}
		data = data[i+1:]
	}
	return ndata, nil
}

func reportResults(ctx *cmd.Context, operation string, machines []FlatMachine, results []execResult) error {
	resultsOutput, err := json.MarshalIndent(results, "", "   ")
	if err != nil {
		return errors.Trace(err)
	}
	logger.Debugf("full %s results: %s", operation, string(resultsOutput))

	var badMachines []string
	for i, res := range results {
		machine := machines[i].ID
		if res.Code == 0 {
			fmt.Fprintf(ctx.Stdout, "%s successful on machine %s\n", operation, machine)
		} else {
			fmt.Fprintf(
				ctx.Stdout,
				"%s failed on machine %s: exited with %d\nOutput was:\n%s\nError was:\n%s\n\n",
				operation,
				machine,
				res.Code,
				res.Stdout,
				res.Stderr,
			)
			badMachines = append(badMachines, machine)
		}
	}

	if len(badMachines) > 0 {
		plural := "s"
		if len(badMachines) == 1 {
			plural = ""
		}
		return errors.Errorf("%s failed on machine%s %s",
			operation, plural, strings.Join(badMachines, ", "))
	}

	return nil
}
