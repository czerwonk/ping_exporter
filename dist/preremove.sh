#!/bin/sh

if [ "$1" != "remove" ]; then
	exit 0
fi

systemctl disable ping_exporter || true
systemctl stop ping_exporter    || true

for v in 233 235 242 245; do
	rm -f /run/systemd/system/ping_exporter.service.d/systemd-$v.conf
done
