#!/bin/sh

groupadd --system ping_exporter || true
useradd --system -d /nonexistent -s /usr/sbin/nologin -g ping_exporter ping_exporter || true

chown ping_exporter /etc/ping_exporter/config.yml

# The base SELinux policy labels /usr/bin/ping* as ping_exec_t, which systemd
# may not execute (AVC denial "execute_no_trans"), so pin the binary to bin_t.
if command -v selinuxenabled >/dev/null 2>&1 && selinuxenabled; then
	if command -v semanage >/dev/null 2>&1; then
		semanage fcontext -a -t bin_t '/usr/bin/ping_exporter' 2>/dev/null ||
			semanage fcontext -m -t bin_t '/usr/bin/ping_exporter' 2>/dev/null ||
			true
	fi

	if command -v restorecon >/dev/null 2>&1; then
		restorecon -F /usr/bin/ping_exporter || true
	fi
fi

current_systemd_version=$(systemctl --version | sed -n '1s/^systemd v\{0,1\}\([0-9]\{1,\}\).*/\1/p')

for v in 233 235 242 245; do
	if [ -n "$current_systemd_version" ] && [ "$current_systemd_version" -ge "$v" ]; then
		cp /usr/local/share/ping_exporter/systemd-$v.conf /run/systemd/system/ping_exporter.service.d/
	fi
done

systemctl daemon-reload
systemctl enable ping_exporter
systemctl restart ping_exporter
