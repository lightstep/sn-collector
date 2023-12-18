## Monitor Linux with the ServiceNow Collector

| Linux Distibution                              | Support Status            | Architecture |
| ---------------------------------------------- | ------------------------- | ------------ |
| Red Hat Enterprise Linux (RHEL), Amazon Linux  | last three major versions | ARM, AMD     |
| Ubuntu                                         | last three major versions | ARM, AMD     |
| Alpine                                         | last three major versions | ARM, AMD     |
| Debian                                         | last three major versions | ARM, AMD     |

### Install for for Linux server monitoring (no containers)

Gather system metrics from a Linux system using an installed software package. Use this for servers and hosts that **do not** have Docker or a container runtime.

1. Download the appropriate package for your system and architecture from the Releases page of this repository.

2. Install the downloaded package using the appropriate package manager for your Linux distribution.
  - RPM (RHEL, CentOS, Amazon Linux) package with `yum`:
    ```sh
    sudo yum install -y otelcol-servicenow_version_linux_arch.rpm 
    ```
  - Debian (Ubuntu) package with `apt-get`:
    ```sh
    sudo apt-get install -y otelcol-servicenow_version_linux_arch.deb 
    ```
  - APK (Alpine Linux) package with `apk`:
    ```sh
    sudo apk add --allow-untrusted otelcol-servicenow_version_linux_arch.apk 
    ```
3. Follow the post-install instructions on starting the collector service.

### Install for for Linux host monitoring with Docker

Gather host system metrics from a Linux using a Docker image.

1. Pull the latest Docker image for the collector.
  - ```sh
    docker pull ghcr.io/lightstep/sn-collector/sn-collector-experimental:latest
    ```

2. Run the collector as a container, but mount the host filesystem to gather host metrics.
  - ```sh
    docker run --rm --name sn-collector-experimental \
      -v ./collector/config/otelcol-docker-hostmetrics.yaml:/etc/otelcol/config.yaml \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -v /:/hostfs
      -e LS_TOKEN=your-cloud-obs-token
      ghcr.io/lightstep/sn-collector/sn-collector-experimental:latest
    ```

3. View the container logs and verify data is being sent.