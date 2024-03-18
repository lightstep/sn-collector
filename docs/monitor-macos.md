## Monitor macOS with the ServiceNow Collector

The last three macOS versions are supported on both Intel (AMD64) and Apple Silicon (ARM-based: M1, M2, etc).

### Automated install for macOS

This install approach automatically downloads dependencies and installs the collector as a service on macOS. Priviliged (root) access is needed.

1. As `sudo`, run the following in your shell. If you do not specify an optional token, edit the configuration file after install completes. If you set an OpAMP key, [which is a project-scoped API Key](https://docs.lightstep.com/docs/create-and-manage-api-keys), certain collector remote management features will be enabled.
  - ```sh
    export CLOUDOBS_TOKEN='your-cloudobs-access-token'
    export OPAMP_KEY='your-opamp-api-key'
    sudo sh -c "$(curl -fsSlL https://github.com/lightstep/sn-collector/releases/latest/download/install-macos.sh)" install_macos.sh --ingest-token $CLOUDOBS_TOKEN --opamp-key $OPAMP_KEY
    ```

2. Review the collector configuration installed in `/opt/sn-collector/config.yaml`. The collector will automatically start running with the default configuration.

3. To *uninstall*, run:
  - ```sh
    sudo sh -c "$(curl -fsSlL https://github.com/lightstep/sn-collector/releases/latest/download/install-macos.sh)" install_macos.sh --uninstall
    ```

### Manual install for macOS

1. On the [Releases](https://github.com/lightstep/sn-collector/releases) page, download the appropriate collector `*.tar.gz` for `darwin` and your processor type. Apple Silicon processors use the `arm` binary.

2. Extract the `*.tar.gz` archive.

3. Validate the collector runs and the bundled configuration is valid.
  - ```sh
    ./sn-collector validate config.yaml
    ```
