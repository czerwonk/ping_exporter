#!/bin/sh

if [ "$1" != "remove" ]; then
	exit 0
fi

systemctl daemon-reload
userdel  ping_exporter || true
groupdel ping_exporter 2>/dev/null || true
