# Copyright 2017 Canonical Ltd.
# Licensed under the AGPLv3, see LICENCE file for details.
"""
Upgrades the local agents to use the new tools binary in a directory
beside this script. Keeps all changed files in
/var/lib/juju/1.25-upgrade-rollback so that they can be restored if
needed.
"""
import json
import os
from os import path
import shutil
import subprocess
import sys
import tarfile
import yaml

FILE_FORMAT = '2.0'

# Config passed in from the upgrade tool.
CA_CERT = """{{.ControllerInfo.CACert}}"""
CONTROLLER_TAG = '{{.ControllerTag}}'
VERSION = '{{.Version}}'
API_ADDRESSES = """{{range .ControllerInfo.Addrs}}{{.}}
{{end}}""".splitlines()

BASE_DIR = '/var/lib/juju'
ROLLBACK_DIR = path.join(BASE_DIR, '1.25-upgrade-rollback')
ROLLBACK_INIT_DIR = path.join(ROLLBACK_DIR, 'init')
TOOLS_DIR = path.join(BASE_DIR, 'tools')
AGENTS_DIR = path.join(BASE_DIR, 'agents')
INIT_DIR = path.join(BASE_DIR, 'init')

UPSTART_DIR = '/etc/init'
SYSTEMD_DIR = '/etc/systemd/system'

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

OLD_CONTROLLER_KEYS = """\
stateservercert
stateserverkey
caprivatekey
apiport
stateport
sharedsecret
systemidentity
""".splitlines()

# Ensure specific text is represented in literal format.
class Literal(str):
    pass

def literal_presenter(dumper, data):
    return dumper.represent_scalar('tag:yaml.org,2002:str', data, style='|')
yaml.add_representer(Literal, literal_presenter)


def all_agents():
    return os.listdir(AGENTS_DIR)

def convert_container_agent(agent, from_type, to_type):
    parts = agent.split('-')
    match = False
    output = []
    for part in parts:
        if part == from_type:
            match = True
            part = to_type
        output.append(part)
    return match, '-'.join(output)

def convert_lxc_agent(agent):
    return convert_container_agent(agent, 'lxc', 'lxd')

def convert_lxd_agent(agent):
    return convert_container_agent(agent, 'lxd', 'lxc')

def save_rollback_info():
    os.makedirs(ROLLBACK_INIT_DIR)
    series = get_series()
    for agent in all_agents():
        tools_link = path.join(TOOLS_DIR, agent)
        target = os.readlink(tools_link)
        os.symlink(target, path.join(ROLLBACK_DIR, agent))

        agent_conf = path.join(AGENTS_DIR, agent, 'agent.conf')
        backup_path = path.join(ROLLBACK_DIR, agent + '_agent.conf')
        shutil.copy(agent_conf, backup_path)

        if series == 'trusty':
            agent_init = path.join(UPSTART_DIR, upstart_conf(agent))
            shutil.copy(agent_init, ROLLBACK_INIT_DIR)
        else:
            # Grab the service symlink...
            service = systemd_conf(agent)
            init_link = path.join(SYSTEMD_DIR, service)
            target = os.readlink(init_link)
            os.symlink(target, path.join(ROLLBACK_INIT_DIR, service))

            # ...And the init subdir.
            dirname = 'jujud-' + agent
            agent_init_dir = path.join(INIT_DIR, dirname)
            saved_dir = path.join(ROLLBACK_INIT_DIR, dirname)
            shutil.copytree(agent_init_dir, saved_dir)

def find_new_tools():
    files = [name for name in os.listdir(UPGRADE_DIR) if path.join(UPGRADE_DIR, name).endswith('.tgz')]
    assert len(files) == 1, 'too many tools files found: {}'.format(files)
    return path.join(UPGRADE_DIR, files[0])

def unpack_tools(source, dest_path):
    with tarfile.open(name=source, mode='r:gz') as contents:
        for item in contents:
            contents.extract(item, path=dest_path)
            item_path = path.join(dest_path, item.name)
            shutil.chown(item_path, 'root', 'root')

def write_tool_metadata(version, dest_path):
    with open(path.join(dest_path, 'downloaded-tools.txt'), 'w') as metadata:
        json.dump(dict(version=version, url="", size=0), metadata)

def install_tools():
    new_tools_path = find_new_tools()
    # get 2.2.3-xenial-amd64 from ~/1.25-agent-upgrade/2.2.3-xenial-amd64.tgz
    tools_base, _ = path.splitext(path.basename(new_tools_path))
    dest_path = path.join(TOOLS_DIR, tools_base)
    os.mkdir(dest_path)
    unpack_tools(new_tools_path, dest_path)
    write_tool_metadata(tools_base, dest_path)
    # Make all the hook tools link to jujud.
    make_links(dest_path, HOOK_TOOLS, path.join(dest_path, 'jujud'))
    # Make all of the agent tools dirs link to the new version.
    make_links(TOOLS_DIR, [convert_lxc_agent(a)[1] for a in all_agents()], dest_path)

def make_links(in_dir, names, target):
    for name in names:
        link_path = path.join(in_dir, name)
        force_symlink(target, link_path)

def update_configs():
    series = get_series()
    need_init_reload = False
    for agent in all_agents():
        lxc, lxd_agent = convert_lxc_agent(agent)
        if lxc:
            os.rename(path.join(AGENTS_DIR, agent), path.join(AGENTS_DIR, lxd_agent))
            update_init_scripts(series, agent, lxd_agent)
            need_init_reload = True
            agent = lxd_agent
        data = read_agent_config(agent)
        if agent.startswith('machine-'):
            data = update_machine_config(agent, data)
        else:
            data = update_unit_config(agent, data)
        write_agent_config(agent, data)
    if need_init_reload:
        reload_init(series)

def config_path(agent):
    return path.join(AGENTS_DIR, agent, 'agent.conf')

def read_agent_config(agent):
    with open(config_path(agent)) as f:
        data = yaml.load(f)
    return data

def write_agent_config(agent, data):
    with open(config_path(agent), 'w') as f:
        f.write('# format %s\n' % FILE_FORMAT)
        yaml.dump(data, stream=f, default_flow_style=False)

def update_machine_config(agent, data):
    # None of these machines will need to manage the environ anymore.
    data['jobs'] = ['JobHostUnits']
    # Get rid of API/mongo hosting keys.
    for name in OLD_CONTROLLER_KEYS:
        if name in data:
            del data[name]
    # Convert any lxc agent tags to lxd
    data['tag'] = convert_lxc_agent(data['tag'])[1]
    return update_unit_config(agent, data)

def update_unit_config(agent, data):
    # Set controller and model.
    env_tag = data['environment']
    data['model'] = env_tag.replace('environment', 'model')
    data['controller'] = CONTROLLER_TAG

    data['upgradedToVersion'] = VERSION
    data['cacert'] = Literal(CA_CERT)

    data['apiaddresses'] = API_ADDRESSES

    # Get rid of unneeded attributes.
    for name in ('environment', 'stateaddresses', 'statepassword'):
        del data[name]

    return data

def update_init_scripts(series, lxc_agent, lxd_agent):
    if series == 'trusty':
        update_upstart_scripts(lxc_agent, lxd_agent)
    else:
        update_systemd_scripts(lxc_agent, lxd_agent)

def update_upstart_scripts(lxc_agent, lxd_agent):
    lxc_path = path.join(UPSTART_DIR, upstart_conf(lxc_agent))
    lxd_path = path.join(UPSTART_DIR, upstart_conf(lxd_agent))
    rewrite_lxc_to_lxd(lxc_path, lxd_path)
    safe_unlink(lxc_path)

def rewrite_lxc_to_lxd(lxc_path, lxd_path):
    "Copies contents of lxc_path file into lxd_path, converting lxc->lxd on the way"
    with open(lxc_path, 'r') as source:
        data = source.read()
    updated = data.replace('lxc', 'lxd')
    with open(lxd_path, 'w') as dest:
        dest.write(updated)

def update_systemd_scripts(lxc_agent, lxd_agent):
    lxc_dir = path.join(INIT_DIR, 'jujud-' + lxc_agent)
    lxd_dir = path.join(INIT_DIR, 'jujud-' + lxd_agent)

    # Create lxd versions of service file and exec-start script.
    os.mkdir(lxd_dir)
    lxd_service_path = path.join(lxd_dir, systemd_conf(lxd_agent))
    rewrite_lxc_to_lxd(
        path.join(lxc_dir, systemd_conf(lxc_agent)),
        lxd_service_path)
    lxd_exec_start = path.join(lxd_dir, 'exec-start.sh')
    rewrite_lxc_to_lxd(path.join(lxc_dir, 'exec-start.sh'), lxd_exec_start)
    os.chmod(lxd_exec_start, 0o755)

    shutil.rmtree(lxc_dir)

    # Correct the link from /etc/systemd/system
    os.unlink(path.join(SYSTEMD_DIR, systemd_conf(lxc_agent)))
    force_symlink(lxd_service_path, path.join(SYSTEMD_DIR, systemd_conf(lxd_agent)))

def get_series():
    return subprocess.check_output(['/usr/bin/lsb_release', '-cs']).decode().strip()

def main():
    assert not path.exists(ROLLBACK_DIR), 'saved rollback information found - aborting'
    save_rollback_info()
    install_tools()
    update_configs()

def safe_unlink(location):
    # path.exists returns False for broken symlinks.
    if path.exists(location) or path.islink(location):
        os.unlink(location)

def force_symlink(target, dest):
    safe_unlink(dest)
    os.symlink(target, dest)

def upstart_conf(agent):
    return 'jujud-{}.conf'.format(agent)

def systemd_conf(agent):
    return 'jujud-{}.service'.format(agent)

def rollback_upstart(lxd_agent, lxc_agent):
    safe_unlink(path.join(UPSTART_DIR, upstart_conf(lxd_agent)))
    saved_conf = path.join(ROLLBACK_INIT_DIR, upstart_conf(lxc_agent))
    shutil.copy(saved_conf, UPSTART_DIR)

def rollback_systemd(lxd_agent, lxc_agent):
    # Get rid of any lxd version of the agent files under /var/lib/juju/init.
    lxd_init_dir = path.join(INIT_DIR, 'jujud-' + lxd_agent)
    if path.exists(lxd_init_dir):
        shutil.rmtree(lxd_init_dir)

    # Reinstate the lxc init files.
    lxc_init_dir = path.join(INIT_DIR, 'jujud-' + lxc_agent)
    if path.exists(lxc_init_dir):
        shutil.rmtree(lxc_init_dir)
    shutil.copytree(path.join(ROLLBACK_INIT_DIR, 'jujud-' + lxc_agent), lxc_init_dir)

    # Get rid of any lxd symlink from /etc/systemd/system.
    safe_unlink(path.join(SYSTEMD_DIR, systemd_conf(lxd_agent)))
    # Reinstate the lxc link.
    lxc_agent_conf = systemd_conf(lxc_agent)
    force_symlink(path.join(lxc_init_dir, lxc_agent_conf), path.join(SYSTEMD_DIR, lxc_agent_conf))

def rollback_init_files(series, lxd_agent, lxc_agent):
    if series == 'trusty':
        rollback_upstart(lxd_agent, lxc_agent)
    else:
        rollback_systemd(lxd_agent, lxc_agent)

def reload_init(series):
    if series == 'trusty':
        command = ['/sbin/initctl', 'reload-configuration']
    else:
        command = ['/bin/systemctl', 'daemon-reload']
    subprocess.check_call(command)

def rollback():
    assert path.exists(ROLLBACK_DIR), 'no rollback information found'
    series = get_series()
    need_init_reload = False
    for agent in all_agents():
        lxd, lxc_agent = convert_lxd_agent(agent)
        if lxd:
            # We need to rename the agent dir and get rid of the tools
            # link for the lxd version of the agent.
            os.rename(
                path.join(AGENTS_DIR, agent),
                path.join(AGENTS_DIR, lxc_agent))
            safe_unlink(path.join(TOOLS_DIR, agent))

            rollback_init_files(series, agent, lxc_agent)
            need_init_reload = True
            agent = lxc_agent

        link_path = path.join(ROLLBACK_DIR, agent)
        target = os.readlink(link_path)
        dest = path.join(TOOLS_DIR, agent)
        force_symlink(target, dest)

        agent_conf = path.join(AGENTS_DIR, agent, 'agent.conf')
        backup_path = path.join(ROLLBACK_DIR, agent + '_agent.conf')
        shutil.copy(backup_path, agent_conf)

    tools_base, _ = path.splitext(path.basename(find_new_tools()))
    added_tools = path.join(TOOLS_DIR, tools_base)
    shutil.rmtree(added_tools)
    shutil.rmtree(ROLLBACK_DIR)

    if need_init_reload:
        reload_init(series)

if __name__ == "__main__":
    if len(sys.argv) == 2 and sys.argv[1] == "rollback":
        rollback()
    else:
        main()
    sys.exit(0)
