package commands

import (
	"bytes"
	"io"
	"os"

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
	Model   string
	Series  string
	ID      string
	Address string
	Tools   string

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

type prefixWriter struct {
	io.Writer
	prefix string
	buf    bytes.Buffer
}

func (w *prefixWriter) Write(data []byte) (int, error) {
	w.buf.Write(data)
	for {
		line, err := w.buf.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return len(data), err
		}
		if err := w.write(line); err != nil {
			return 0, err
		}
	}
}

func (w *prefixWriter) Flush() error {
	line := w.buf.Bytes()
	if len(line) > 0 {
		w.buf.Truncate(0)
		return w.write(line)
	}
	return nil
}

func (w *prefixWriter) write(line []byte) error {
	if _, err := w.Writer.Write([]byte(w.prefix)); err != nil {
		return err
	}
	if _, err := w.Writer.Write(line); err != nil {
		return err
	}
	return nil
}
