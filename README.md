# 1.25-upgrade
Tools to upgrade and move a 1.25 environment to a 2.1 controller


## Initial checks

Verify that you have access to both the source 1.25 environment, and a valid 2.1+ controller.

  juju 1.25-upgrade verify-source <envname>

Check the status of all the agents.

  juju 1.25-upgrade agent-status <envname>


## Stop all the agents on the source environment.

  juju 1.25-upgrade stop-agents <envname>


## Import the environment into the controller

  juju 1.25-upgrade import <envname> <controller>



  juju 1.25-upgrade upgrade-agents <envname> <controller>

log into controller
get version, endpoints
download tools from controller
copy tools to all agents
update agent config



  juju 1.25-upgrade abort <envname> <controller>



Start the agents

  juju 1.25-upgrade start-agents <envname>
