package commands

import (
	"bytes"
	"io"
	"os"
	"sync"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/ssh"
)

const systemIdentity = "/var/lib/juju/system-identity"

type FlatMachine struct {
	Model   string
	Series  string
	ID      string
	Address string
	Tools   string
}

type RunResult struct {
	Code int
}

type execOptions struct {
	ssh.Options
	stdout io.Writer
	stderr io.Writer
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

// runViaSSH runs script in the remote machine with address addr.
func runViaSSH(addr, script string, opts ...execOption) (int, error) {
	// This is taken from cmd/juju/ssh.go there is no other clear way to set user
	userAddr := "ubuntu@" + addr

	var options execOptions
	options.stdout = os.Stdout
	options.stderr = os.Stderr
	// Strict host key checking must be disabled because Juju 1.25 did not
	// populate SSH host keys.
	options.SetStrictHostKeyChecking(ssh.StrictHostChecksNo)
	options.SetKnownHostsFile(os.DevNull)
	for _, opt := range opts {
		opt(&options)
	}

	userCmd := ssh.Command(
		userAddr,
		[]string{"sudo", "-n", "bash", "-c " + utils.ShQuote(script)},
		&options.Options,
	)
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

type DistResult struct {
	Model     string
	Series    string
	MachineID string
	Error     error
	Code      int
	Stdout    string
	Stderr    string
}

func parallelCall(machines []FlatMachine, script string) []DistResult {

	var wg sync.WaitGroup
	results := make([]DistResult, len(machines))

	for i, machine := range machines {
		wg.Add(1)
		go func(i int, machine FlatMachine) {
			defer wg.Done()
			var stdoutBuf bytes.Buffer
			var stderrBuf bytes.Buffer
			rc, err := runViaSSH(
				machine.Address, script,
				withSystemIdentity(),
				withStdout(&stdoutBuf),
				withStderr(&stderrBuf),
			)
			results[i] = DistResult{
				Model:     machine.Model,
				Series:    machine.Series,
				MachineID: machine.ID,
				Error:     err,
				Code:      rc,
				Stdout:    stdoutBuf.String(),
				Stderr:    stderrBuf.String(),
			}
		}(i, machine)
	}

	logger.Debugf("Waiting for copies for finish")
	wg.Wait()

	// TODO Sort the results
	return results
}
