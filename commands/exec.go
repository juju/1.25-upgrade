package commands

import (
	"bytes"
	"sync"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/utils"
	"github.com/juju/utils/ssh"
)

const systemIdentity = "/var/lib/juju/system-identity"

type FlatMachine struct {
	Model   string
	ID      string
	Address string
}

type RunResult struct {
	Code   int
	Stdout string
	Stderr string
}

// runViaSSH runs script in the remote machine with address addr.
func runViaSSH(addr string, script, identity string) (RunResult, error) {
	// This is taken from cmd/juju/ssh.go there is no other clear way to set user
	userAddr := "ubuntu@" + addr
	sshOptions := ssh.Options{}
	if identity != "" {
		sshOptions.SetIdentities(identity) //
	}
	userCmd := ssh.Command(userAddr, []string{"sudo", "-n", "bash", "-c " + utils.ShQuote(script)}, &sshOptions)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	userCmd.Stdout = &stdoutBuf
	userCmd.Stderr = &stderrBuf
	var result RunResult
	logger.Debugf("executing %s, script:\n%s", addr, script)
	err := userCmd.Run()
	result.Stdout = stdoutBuf.String()
	result.Stderr = stderrBuf.String()
	if err != nil {
		if rc, ok := err.(*cmd.RcPassthroughError); ok {
			result.Code = rc.Code
		} else {
			return result, errors.Trace(err)
		}
	}

	return result, nil
}

type DistResult struct {
	Model     string
	MachineID string
	Error     error
	Code      int
	Stdout    string
	Stderr    string
}

func parallelCall(machines []FlatMachine, script string) []DistResult {

	var (
		wg      sync.WaitGroup
		results []DistResult
		lock    sync.Mutex
	)

	for _, machine := range machines {
		wg.Add(1)
		go func(machine FlatMachine) {
			defer wg.Done()
			run, err := runViaSSH(machine.Address, script, systemIdentity)
			result := DistResult{
				Model:     machine.Model,
				MachineID: machine.ID,
				Error:     err,
				Code:      run.Code,
				Stdout:    run.Stdout,
				Stderr:    run.Stderr,
			}
			lock.Lock()
			defer lock.Unlock()
			results = append(results, result)
		}(machine)
	}

	logger.Debugf("Waiting for copies for finish")
	wg.Wait()

	// Sort the results
	return results
}
