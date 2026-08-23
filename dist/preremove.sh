#!/bin/sh

# dpkg passes "remove", rpm passes "0"; anything else is an upgrade.
case "$1" in
remove | 0) ;;
*) exit 0 ;;
esac

systemctl disable ping_exporter || true
systemctl stop ping_exporter    || true

for v in 233 235 242 245; do
	rm -f /run/systemd/system/ping_exporter.service.d/systemd-$v.conf
done
