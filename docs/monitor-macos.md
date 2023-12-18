## Monitor macOS with the ServiceNow Collector

The last three macOS versions are supported on both Intel (AMD64) and Apple Silicon (ARM-based: M1, M2, etc)

### Install for macOS Monitoring

To install using an automated script, run:

```sh
sudo sh -c "$(curl -fsSlL https://github.com/lightstep/sn-collector/releases/latest/download/install_macos.sh)" install_macos.sh
```

By default, the configuration for the collector will be installed in `/opt/sn-collector/config.yaml`.
