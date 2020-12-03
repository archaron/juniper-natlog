#!/bin/bash
export USER="nobody"
export GROUP="nogroup"
id -u $USER &>/dev/null || useradd $USER
id -g $USER &>/dev/null || groupadd $GROUP
chown $USER:$GROUP /opt/natlog/etc/natlog.yaml
systemctl daemon-reload
