#!/bin/sh

set -e

username="sn-collector"

if getent group "$username" > /dev/null 2>&1; then
    echo "Group ${username} already exists."
else
    groupadd "$username"
fi

if id "$username" > /dev/null 2>&1; then
    echo "User ${username} already exists"
    exit 0
else
    useradd --shell /sbin/nologin --system "$username" -g "$username"
fi
