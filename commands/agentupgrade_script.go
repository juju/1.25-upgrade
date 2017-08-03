// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package commands

// Generated code - do not edit.

const agentUpgradeScript = `# Copyright 2017 Canonical Ltd.
# Licensed under the AGPLv3, see LICENCE file for details.
"""
Upgrades the local agents to use the new tools binary in a directory
beside this script. Keeps all changed files in
/var/lib/juju/1.25-upgrade-rollback so that they can be restored if
needed.
"""
import os
from os import path
import shutil
import sys

# Config passed in from the upgrade tool.
FILE_FORMAT = '2.0'
CA_CERT = """{{.ControllerInfo.CACert}}"""
CONTROLLER_TAG = '{{.ControllerTag}}'

BASE_DIR = '/var/lib/juju'
ROLLBACK_DIR = path.join(BASE_DIR, '1.25-upgrade-rollback')
TOOLS_DIR = path.join(BASE_DIR, 'tools')
AGENTS_DIR = path.join(BASE_DIR, 'agents')

UPGRADE_DIR, SCRIPT = path.split(__file__)

HOOK_TOOLS = """\
action-fail
action-get
action-set
add-metric
application-version-set
close-port
config-get
is-leader
juju-log
juju-reboot
leader-get
leader-set
network-get
opened-ports
open-port
payload-register
payload-status-set
payload-unregister
relation-get
relation-ids
relation-list
relation-set
resource-get
status-get
status-set
storage-add
storage-get
storage-list
unit-get
""".splitlines()

def all_agents():
    return os.listdir(AGENTS_DIR)

def save_rollback_info():
    os.mkdir(ROLLBACK_DIR)
    for agent in all_agents():
        tools_link = path.join(TOOLS_DIR, agent)
        target = os.readlink(tools_link)
        os.symlink(target, path.join(ROLLBACK_DIR, agent))

        agent_conf = path.join(AGENTS_DIR, agent, 'agent.conf')
        backup_path = path.join(ROLLBACK_DIR, agent + '_agent.conf')
        shutil.copy(agent_conf, backup_path)

def find_new_tools():
    dirs = [name for name in os.listdir(UPGRADE_DIR) if path.isdir(path.join(UPGRADE_DIR, name))]
    assert len(dirs) == 1, 'too many tools dirs found: {}'.format(dirs)
    return path.join(UPGRADE_DIR, dirs[0])

def install_tools():
    new_tools_path = find_new_tools()
    dest_path = path.join(TOOLS_DIR, path.basename(new_tools_path))
    shutil.copytree(new_tools_path, dest_path)
    # Make all the hook tools link to jujud.
    make_links(dest_path, HOOK_TOOLS, path.join(dest_path, 'jujud'))
    # Make all of the agent tools dirs link to the new version.
    make_links(TOOLS_DIR, all_agents(), dest_path)

def make_links(in_dir, names, target):
    for name in names:
        link_path = path.join(in_dir, name)
        if path.exists(link_path):
            os.unlink(link_path)
        os.symlink(target, link_path)

def update_configs():
    pass

def main():
    assert not path.exists(ROLLBACK_DIR), 'saved rollback information found - aborting'
    save_rollback_info()
    install_tools()
    update_configs()

def rollback():
    assert path.exists(ROLLBACK_DIR), 'no rollback information found'
    for agent in all_agents():
        link_path = path.join(ROLLBACK_DIR, agent)
        target = os.readlink(link_path)
        dest = path.join(TOOLS_DIR, agent)
        if path.exists(dest):
            os.unlink(dest)
        os.symlink(target, dest)

        agent_conf = path.join(AGENTS_DIR, agent, 'agent.conf')
        backup_path = path.join(ROLLBACK_DIR, agent + '_agent.conf')
        shutil.copy(backup_path, agent_conf)

    added_tools = path.join(TOOLS_DIR, path.basename(find_new_tools()))
    shutil.rmtree(added_tools)
    shutil.rmtree(ROLLBACK_DIR)

if __name__ == "__main__":
    if len(sys.argv) == 2 and sys.argv[1] == "rollback":
        rollback()
    else:
        main()
    sys.exit(0)
`
