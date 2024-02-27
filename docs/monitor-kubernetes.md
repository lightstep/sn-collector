## Monitor Kubernetes with the ServiceNow Collector

| Kuberenetes Distibution                        | Support Status            | Architecture |
| ---------------------------------------------- | ------------------------- | ------------ |
| GKE (Google Cloud)                             | last three major versions | ARM, AMD     |
| EKS (AWS)                                      | last three major versions | ARM, AMD     |
| AKS (Azure)                                    | last three major versions | ARM, AMD     |
| Kubernetes                                     | last three major versions | ARM, AMD     |

* **Note:** We recommend Red Hat OpenShift customers use the [Red Hat OpenTelemetry Distribution](https://docs.openshift.com/container-platform/4.12/otel/otel-using.html).

### Deploy for Kubernetes monitoring with The OpenTelemetry Operator and Helm

#### Requirements

* `helm` v3
* Kubernetes cluster with local access via `kubectl`
* ability to pull from the Docker image repository `ghcr.io/lightstep/sn-collector`

#### 1. Add OpenTelemetry Helm Repository

We use the OpenTelemetry Helm charts to install the OpenTelemetry Operator. The Operator makes it easy to scale and configure collectors in Kubernetes.

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

#### 2. Create a ServiceNow Namespace

This namespace is where the OpenTelemetry components will live in your cluster.

```sh
kubectl create namespace servicenow
```

#### 3. Set credentials

Paste your token carefully and escape it in single-quotes so special characters aren't interpreted by your shell.

```sh
export CLOUDOBS_TOKEN='<your-cloudobs-token>'
kubectl create secret generic servicenow-cloudobs-token \
    -n servicenow --from-literal='token=$CLOUDOBS_TOKEN'
```

Set username for Event Manangement:
```sh
export SERVICENOW_EVENTS_USERNAME='<your-mid-user>'
kubectl create secret generic servicenow-events-user \
    -n servicenow --from-literal='USERNAME=$SERVICENOW_EVENTS_USERNAME'
```

Set password for Event Manangement:
```sh
export SERVICENOW_EVENTS_PASSWORD='<your-mid-user-pw>'
kubectl create secret generic servicenow-events-password \
    -n servicenow --from-literal='PASSWORD=$SERVICENOW_EVENTS_PASSWORD'
```

#### 3. Deploy ServiceNow Collector for Cluster Monitoring

You're now ready to deploy a collector to your cluster to collect cluster-level metrics and events.

```sh
helm upgrade otel-collector-cluster open-telemetry/opentelemetry-collector --install --namespace servicenow --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-cluster.yaml
```

The pod will deploy after a few seconds, to check status and for errors, run:

```sh
kubectl get pods -n servicenow
```

#### 3. Deploy ServiceNow Collector for Node and Workloads Monitoring

Next, deploy collectors to each Kubernetes host to get workload metrics (via Kubelet).

```sh
helm upgrade otel-collector open-telemetry/opentelemetry-collector --install --namespace servicenow --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-node.yaml
```
