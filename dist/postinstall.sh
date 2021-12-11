#!/bin/sh

groupadd --system ping_exporter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g ping_exporter ping_exporter || true

chown ping_exporter /etc/ping_exporter/config.yml

current_systemd_version=$(dpkg-query --showformat='${Version}' --show systemd)

for v in 233 235 242 245; do
	if dpkg --compare-versions "$current_systemd_version" ge "$v"; then
		cp /usr/local/share/ping_exporter/systemd-$v.conf /run/systemd/system/ping_exporter.service.d/
	fi
done

systemctl daemon-reload
systemctl enable ping_exporter
systemctl restart ping_exporter
