#!/bin/sh

set -e

manage_systemd_service() {
    systemctl daemon-reload

    echo "configured systemd service"

    cat << EOF

The "sn-collector" service has been configured!

The collector's config file can be found here: 
  /opt/sn-collector/config.yaml

To view logs from the collector, run:
  sudo journalctl --unit=sn-collector

For more information on configuring the collector, see the docs:
  https://github.com/lightstep/sn-collector

To stop the sn-collector service, run:
  sudo systemctl stop sn-collector

To start the sn-collector service, run:
  sudo systemctl start sn-collector

To restart the sn-collector service, run:
  sudo systemctl restart sn-collector

To enable the service on startup, run:
  sudo systemctl enable sn-collector

If you have any other questions please contact us at support@lightstep.com
EOF
}

init_type() {
  systemd_test="$(systemctl 2>/dev/null || : 2>&1)"
  if command printf "$systemd_test" | grep -q '\-.mount'; then
    command printf "systemd"
    return
  fi

  command printf "unknown"
  return
}

manage_service() {
  service_type="$(init_type)"
  case "$service_type" in
    systemd)
      manage_systemd_service
      ;;
    *)
      echo "could not detect init system, skipping service configuration"
  esac
}

finish_permissions() {
  # Goreleaser does not set plugin file permissions, so do them here
  # We also change the owner of the binary to sn-collector
  chown -R sn-collector:sn-collector /opt/sn-collector/otelcol-servicenow

  # Initialize the log file to ensure it is owned by sn-collector.
  # This prevents the service (running as root) from assigning ownership to
  # the root user. By doing so, we allow the user to switch to sn-collector
  # user for 'non root' installs.
  mkdir -p /opt/sn-collector/log
  touch /opt/sn-collector/log/collector.log
  chown sn-collector:sn-collector /opt/sn-collector/log/collector.log
}


finish_permissions
manage_service