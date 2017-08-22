package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/ssh"
	"golang.org/x/sync/errgroup"
)

const systemIdentity = "/var/lib/juju/system-identity"

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

// withProxyCommandForHost returns an option decorator for setting
// an SSH proxy command to proxy through the given host.
func withProxyCommandForHost(hostAddr string) execOption {
	return withProxyCommand(
		"ssh", "-q",
		"-i", systemIdentity,
		"-o", "StrictHostKeyChecking no",
		"-o UserKnownHostsFile /dev/null",
		"ubuntu@"+hostAddr,
		"nc %h %p",
	)
}

func withProxyCommand(cmd ...string) execOption {
	return func(opts *execOptions) {
		opts.SetProxyCommand(cmd...)
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
	// This is taken from cmd/juju/ssh.go there is no other clear way to set user
	userAddr := "ubuntu@" + addr

	options := execOptions{Options: defaultSSHOptions()}
	options.stdout = os.Stdout
	options.stderr = os.Stderr
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
