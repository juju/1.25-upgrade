# 1.25-upgrade
Tools to upgrade and move a 1.25 environment to a 2.2.4 controller.

The basic approach for the upgrade is to run a series of commands which:
* Stop all of the agents in the source environment.
* Convert any LXC containers in the environment into LXD containers (2.2.4 doesn't support LXC containers).
* Export the state database from the source environment (in the 2.x export format) and import that into the target controller.
* Update the agent binaries on all of the machines to those for the target controller, and rewrite the agent configuration files to talk to the target.
* Activate the migrated model in the target controller.
* Start the agents - at this point the Juju model should be fully functional and hosted in the target controller.

Before you start, ensure that you can ssh to the source environment's machine-0 as ubuntu - this is needed so the 1.25-upgrade binary can copy itself into the source environment and perform upgrade steps. SSH keys can be added to the machine using the Juju 1 `authorized-keys add` command.

Note: the juju-1.25-upgrade binary runs as a [Juju plugin](https://jujucharms.com/docs/2.2/juju-plugins) - it can be run using either the juju1 or juju2 command (however they're installed) It embeds client code for both Juju 1.25 and Juju 2.2.4, so it doesn't need to run the commands or need them to be installed in specific paths.

## Update MAAS agent name

(This is only needed if the source environment is in MAAS.)
There's one setting we need to change in MAAS which we can't do in any other way than by using PSQL on the MAAS region controller.

    juju 1.25-upgrade update-maas-agentname <envname>

This will display a command that needs to be run from a shell on the region controller and then wait for the update. Copy the command and run it and the update-maas-agentname command will see the change and finish.


## Initial checks

Verify that you have access to both the source 1.25 environment, and a valid 2.2.4 controller.

    juju 1.25-upgrade verify-source <envname>

Check the status of all the agents.

    juju 1.25-upgrade agent-status <envname>

### Check that there are no LXC monitors in the same systemd control group as jujud machine agents.
(If you don't have any LXC containers in your environment *or* all of the container hosts are running Trusty, then you won't need to worry about this.) Any LXC containers in the jujud machine agent's service control group will be shut down when `stop-agents` is run, interrupting any workload they're running. You can detect this situation by looking at the status for the jujud machine agent service on the host:

    systemctl status jujud-machine-<n>.service
    
If the CGroup for the service includes any processes that look like:

    [lxc monitor] /var/lib/lxc juju-machine-<n>-lxc-<m>
    
... then those containers will be stopped along with the jujud process. To avoid this, you can restart the containers one by one (hopefully allowing the workload they're handling to failover between them) - this will remove them from the machine agent's control group.

## Stop all the agents in the source environment.

    juju 1.25-upgrade stop-agents <envname>


## Stop and backup the LXC containers in the source environment (optional/recommended)

Run the following command to take a backup of the LXC containers in the
original environment:

    juju 1.25-upgrade backup-lxc <envname> <backup-dir>

where "backup-dir" is a client-local directory in which backups of the LXC
containers will be stored. This command will backup all LXC containers in
the environment, so plan your capacity accordingly, and expect this to take
some time if your client is far from the environment.

WARNING: backing up the LXC containers requires that they be stopped first.
After a container is backed up, it will be started again.

To back up the containers individually, you can supply a regular expression
via the --match flag, which matches the container IDs. You can also supply
the --dry-run flag to list the containers that will be backed up.

You can skip the backup-lxc step at your own risk. The migration to LXD
will discard the LXC root filesystem.

## Migrate LXC containers in the source environment to LXD.

After backing up the LXC containers, migrate them to LXD:

    juju 1.25-upgrade migrate-lxc <envname>

After this command is run, the containers will have been migrated to LXD,
and renamed according to how Juju 2.2.3 expects them.

WARNING: migrating the LXC containers requires that they be stopped first.
After a container is migrated, it will be started again (as a LXD container),
and the Juju agent running within it will be stopped.

To migrate the containers individually, you can supply a regular expression
via the --match flag, which matches the container IDs. You can also supply
the --dry-run flag to list the containers that will be migrated.

## Import the environment into the controller

    juju 1.25-upgrade import <envname> <controller>
    
If the name of the 1.25 environment isn't the same as the name of the cloud in the target, specify the cloud name using the `--target-cloud` option.

You can see that the model has been created in the target controller by running (with Juju 2, however that's installed):

    juju2 models

The new model will be shown as busy until the upgrade is finished and the model is activated.
If the provider is one where we use tagging to determine which resources are part of the environment (like Openstack), the tags will also be upgraded here.

This command doesn't modify the source environment's state database.

## Upgrade the agent tools and configuration on the source env machines

    juju 1.25-upgrade upgrade-agents <envname> <controller>

The last step of the `upgrade-agents` command will perform a
connectivity check to ensure that all of the agents can connect to the
target controller API.

## Finalise the import and activate the new model

    juju 1.25-upgrade activate <envname> <controller>

## Start the agents

    juju 1.25-upgrade start-agents <envname>
    
## Post-upgrade cleanup

Once the new model is up and running under the target controller the old
machine 0 can be removed (assuming that wasn't hosting any actual workload
units).

If your model is running under Openstack the upgrade will have left security
groups with old names behind, which can be removed now using Openstack tools. 
These will have names matching the patterns `juju-<environmentname>` or 
`juju-<environmentname>-0`.


# In the case of an error during upgrade

Ensure all of the agents are stopped

    juju 1.25-upgrade stop-agents <envname>

Run

    juju 1.25-upgrade abort <envname> <controller>

This will removed the imported model on the target controller (as long
as it wasn't activated), undo the upgrade-agent steps and downgrade the 
provider tagging if relevant.

If the environment includes containers, then you will also have to 
revert the upgraded LXD containers to LXC ones:

    juju 1.25-upgrade revert-lxd <envname>
    
(Alternatively you could restore the backups you made of the LXC containers, 
rather than reverting from the LXD containers. Note that the migrate-lxc command 
does not store backups on the hosts, as the hosts may not have sufficient 
disk space for duplicate root filesystems:

    juju 1.25-upgrade restore-lxc <envname> <backup-dir>

)

After aborting the upgrade, you should start the agents back up:

    juju 1.25-upgrade start-agents <envname>
