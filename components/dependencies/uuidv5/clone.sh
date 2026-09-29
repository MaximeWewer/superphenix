#!/bin/bash

REPO=$1

# If no repo is specified, we're not cloning anything
if [ -z "$REPO" ]; then exit 0; fi

# Only plain repository URLs: never an option for git.
case "$REPO" in
  https://*|ssh://*|git@*) ;;
  *) echo "unsupported repository URL" >&2; exit 1 ;;
esac

export GIT_SSH_COMMAND="ssh -i /id_rsa -o IdentitiesOnly=yes -o StrictHostKeyChecking=no"
git clone -b main -- "$REPO" remote 2>&1