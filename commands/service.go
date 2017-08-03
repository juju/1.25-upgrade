package commands

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/juju/cmd/output"
)

func printServiceStatus(ctx *cmd.Context, machines []FlatMachine) error {
	serviceStatusOutput, err := agentServiceCommand(ctx, machines, "status")
	if err != nil {
		return errors.Trace(err)
	}
	values := parseStatus(machines, serviceStatusOutput)
	writer := output.TabWriter(ctx.Stdout)
	wrapper := output.Wrapper{writer}
	wrapper.Println("AGENT", "STATUS", "VERSION")
	for _, v := range values {
		wrapper.Println(v.agent, v.status, v.version)
	}
	writer.Flush()
	return nil
}

type statusResult struct {
	agent   string
	status  string
	version string
}

func parseStatus(machines []FlatMachine, serviceStatusOutput []string) []statusResult {
	var results []statusResult

	for i, stdout := range serviceStatusOutput {
		machine := machines[i]
		agents := strings.Split(stdout, "-- end-of-agent --\n")
		for _, agent := range agents[:len(agents)-1] {
			var result statusResult
			parts := strings.SplitN(agent, "\n", 3)
			result.agent = parts[0]
			lsParts := strings.Split(parts[1], " ")
			toolsPath := lsParts[len(lsParts)-1]
			result.version = path.Base(toolsPath)
			switch machine.Series {
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

// agentServiceCommand runs the given "service" subcommand for every Juju agent
// on the specified machines, and returns the stdout for each. If any of the
// commands fail, this function call will return an error, and anything written
// to stderr will be logged, prefixed by the name of the machine on which the
// command failed.
func agentServiceCommand(ctx *cmd.Context, machines []FlatMachine, command string) ([]string, error) {
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

	targets := flatMachineExecTargets(machines...)
	results, err := parallelExec(targets, script)
	if err != nil {
		return nil, errors.Trace(err)
	}
	var failed []string
	stdout := make([]string, len(results))
	for i, result := range results {
		if result.Code == 0 {
			stdout[i] = result.Stdout
			continue
		}
		failed = append(failed, machines[i].ID)
		if strings.TrimSpace(result.Stdout) != "" {
			w := &prefixWriter{
				Writer: ctx.GetStderr(),
				prefix: fmt.Sprintf("(%s:stdout) ", machines[i].ID),
			}
			io.WriteString(w, result.Stdout)
			w.Flush()
		}
		if strings.TrimSpace(result.Stderr) != "" {
			w := &prefixWriter{
				Writer: ctx.GetStderr(),
				prefix: fmt.Sprintf("(%s:stderr) ", machines[i].ID),
			}
			io.WriteString(w, result.Stderr)
			w.Flush()
		}
	}
	if len(failed) == 0 {
		return stdout, nil
	}
	plural := ""
	if len(failed) > 1 {
		plural = "s"
	}
	return nil, errors.Errorf("service %s command failed for machine%s %q", command, plural, failed)
}
