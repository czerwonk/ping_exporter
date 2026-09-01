#!/bin/sh

case "$1" in
remove | purge | 0) ;;
*) exit 0 ;;
esac

systemctl daemon-reload

if command -v semanage >/dev/null 2>&1; then
	semanage fcontext -d '/usr/bin/ping_exporter' 2>/dev/null || true
fi

userdel  ping_exporter || true
groupdel ping_exporter 2>/dev/null || true
