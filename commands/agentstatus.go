// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"

	"github.com/juju/1.25-upgrade/juju1/state"
	"github.com/juju/1.25-upgrade/juju2/cmd/output"
)

var agentStatusDoc = ` 
The purpose of the agent-status command is to check the status of all the
agents of a 1.25 environment. The agents may be running the 1.25 binary, or a
2.x binary. The command will return the status of the agent, and what tools
they are currently set to use.

`

func newAgentStatusCommand() cmd.Command {
	command := &agentStatusCommand{}
	command.remoteCommand = "agent-status-impl"
	return wrap(command)
}

type agentStatusCommand struct {
	baseClientCommand
}

func (c *agentStatusCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "agent-status",
		Args:    "<environment name>",
		Purpose: "show the status of all the agents for the specified environment",
		Doc:     agentStatusDoc,
	}
}

func (c *agentStatusCommand) Init(args []string) error {
	args, err := c.baseClientCommand.init(args)
	if err != nil {
		return errors.Trace(err)
	}
	return cmd.CheckEmpty(args)
}

var agentStatusImplDoc = `

agent-status-impl must be executed on an API server machine of a 1.25
environment.

The command will get a list of all the machines, and their addresses, and then
ssh to all the machines to check on the status of the various agents on those
machines.

`

func newAgentStatusImplCommand() cmd.Command {
	return &agentStatusImplCommand{}
}

type agentStatusImplCommand struct {
	baseRemoteCommand
}

func (c *agentStatusImplCommand) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "agent-status-impl",
		Purpose: "controller aspect of agent-status",
		Doc:     agentStatusImplDoc,
	}
}

func (c *agentStatusImplCommand) Run(ctx *cmd.Context) error {
	st, err := c.getState(ctx)
	if err != nil {
		return errors.Annotate(err, "getting state")
	}
	defer st.Close()

	// Here we always use the 1.25 environment to get all of the machine
	// addresses. We then use those to ssh into every one of those machine
	// and run the service status script against all the agents.
	machines, err := getMachines(st)
	if err != nil {
		return errors.Annotate(err, "unable to get addresses for machines")
	}

	// The information is then gathered and parsed and formatted here before
	// the data is passed back to the caller.
	serviceStatus(ctx, machines)

	return nil
}

func serviceStatus(ctx *cmd.Context, machines []FlatMachine) {
	results := serviceCall(machines, "status")
	values := parseStatus(results)
	writer := output.TabWriter(ctx.Stdout)
	wrapper := output.Wrapper{writer}
	wrapper.Println("AGENT", "STATUS", "VERSION")
	for _, v := range values {
		wrapper.Println(v.agent, v.status, v.version)
	}
	writer.Flush()
}

type statusResult struct {
	agent   string
	status  string
	version string
}

func parseStatus(status []DistResult) []statusResult {
	var results []statusResult

	for _, r := range status {
		agents := strings.Split(r.Stdout, "-- end-of-agent --\n")
		for _, agent := range agents[:len(agents)-1] {
			var result statusResult
			parts := strings.SplitN(agent, "\n", 3)
			result.agent = parts[0]
			lsParts := strings.Split(parts[1], " ")
			toolsPath := lsParts[len(lsParts)-1]
			result.version = path.Base(toolsPath)
			switch r.Series {
			case "trusty":
				result.status = upstartStatus(parts[2])
			default:
				result.status = systemdStatus(parts[2])
			}
			logger.Debugf("%#v", result)
			results = append(results, result)
		}
	}

	sort.Sort(statusResults(results))
	return results
}

type statusResults []statusResult

func (r statusResults) Len() int           { return len(r) }
func (r statusResults) Less(i, j int) bool { return r[i].agent < r[j].agent }
func (r statusResults) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

var upstartRegexp = regexp.MustCompile(`jujud-[\w-]+ ([\w/]+)`)

func upstartStatus(output string) string {
	matches := upstartRegexp.FindStringSubmatch(output)
	switch len(matches) {
	case 2:
		return matches[1]
	default:
		logger.Warningf("unable to determine status from:\n%s", output)
		return "unknown"
	}
}

var systemdRegexp = regexp.MustCompile(`Active: (\w+ \(\w+\))`)

func systemdStatus(output string) string {
	matches := systemdRegexp.FindStringSubmatch(output)
	switch len(matches) {
	case 2:
		return matches[1]
	default:
		logger.Warningf("unable to determine status from:\n%s", output)
		return "unknown"
	}
}

func serviceCall(machines []FlatMachine, command string) []DistResult {

	script := fmt.Sprintf(`
set -xu
cd /var/lib/juju/agents
for agent in *
do
	echo $agent
	ls -al /var/lib/juju/tools/$agent
	sudo service jujud-$agent %s
	echo "-- end-of-agent --"
done
	`, command)

	return parallelCall(machines, script)
}

func getMachines(st *state.State) ([]FlatMachine, error) {
	machines, err := st.AllMachines()
	if err != nil {
		return nil, errors.Annotate(err, "getting 1.25 machines")
	}
	var result []FlatMachine
	for i, m := range machines {
		address, err := getMachineAddress(m)
		if err != nil {
			return nil, errors.Annotatef(err, "address for machine %q", m.Id())
		}
		fm := FlatMachine{
			Model:   st.EnvironUUID(),
			Series:  m.Series(),
			ID:      m.Id(),
			Address: address,
		}
		if tools, err := m.AgentTools(); err == nil {
			fm.Tools = tools.Version.String()
		}
		logger.Debugf("%d: %#v", i, fm)
		result = append(result, fm)
	}
	return result, nil
}

func getMachineAddress(m *state.Machine) (string, error) {
	// Start with the private address, which is more likely to be set
	// fallback to the public address, and error out if they are both missing.
	private, err := m.PrivateAddress()
	if err == nil {
		return private.Value, nil
	}
	public, err := m.PublicAddress()
	if err != nil {
		return "", errors.New("no private nor public address")
	}
	return public.Value, nil
}
