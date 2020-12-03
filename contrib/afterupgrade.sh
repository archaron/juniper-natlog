#!/bin/bash
systemctl daemon-reload
if [ "`systemctl is-active natlog`" != "active" ]
then
    systemctl restart natlog
fi
