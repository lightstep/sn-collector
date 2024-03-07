## Monitor Kubernetes with the ServiceNow Collector

| Kuberenetes Distibution                        | Support Status            | Architecture |
| ---------------------------------------------- | ------------------------- | ------------ |
| GKE (Google Cloud)                             | last three major versions | ARM, AMD     |
| EKS (AWS)                                      | last three major versions | ARM, AMD     |
| AKS (Azure)                                    | last three major versions | ARM, AMD     |
| Kubernetes                                     | last three major versions | ARM, AMD     |

* **Note:** We recommend Red Hat OpenShift customers use the [Red Hat OpenTelemetry Distribution](https://docs.openshift.com/container-platform/4.15/otel/otel-installing.html).

### Deploy for Kubernetes monitoring with The OpenTelemetry Operator and Helm

#### Requirements

* `helm` v3
* Kubernetes cluster with local access via `kubectl`
* active workloads running in your cluster (no workloads or a test cluster? [See below for deploying the OpenTelemetry demo](#optional-run-the-opentelemetry-demo))
* ability to pull from the public Docker image repository `ghcr.io/lightstep/sn-collector`
* `ClusterRole` 

#### 1. Add OpenTelemetry helm repository

We use the OpenTelemetry Helm charts to configure collectors for Kubernetes monitoring. Helm charts make it easy to deploy and configure Kubernetes manifests.

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

#### 2. Create a ServiceNow namespace

This namespace is where the OpenTelemetry components will live in your cluster.

```sh
kubectl create namespace servicenow
```

#### 3. Set credentials

```sh
export CLOUDOBS_TOKEN='<your-cloudobs-token>'
kubectl create secret generic servicenow-cloudobs-token \
    -n servicenow --from-literal=token=$CLOUDOBS_TOKEN
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

You're now ready to deploy a collector to your cluster to collect cluster-level metrics and events. To preview the generated manifest before deploying, add the `--dry-run` option to the below command:

```sh
helm upgrade otel-collector-cluster open-telemetry/opentelemetry-collector \ 
    --install --namespace servicenow \
    --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-cluster.yaml
```

The pod will deploy after a few seconds, to check status and for errors, run:

```sh
kubectl get pods -n servicenow
```

#### 3. Deploy ServiceNow Collector for Node and Workloads Monitoring

Next, deploy collectors to each Kubernetes host to get workload metrics (via Kubelet). To preview the generated manifest before deploying, add the `--dry-run` option to the below command:

```sh
helm upgrade otel-collector \
    open-telemetry/opentelemetry-collector \
    --install --namespace servicenow \
    --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-node.yaml
```

#### 4. See data in ServiceNow

If all went well, Kubernetes metrics and events will be sent to ServiceNow and Cloud Observability.

🎉

### Optional: Run the OpenTelemetry demo

If you just want to see how OpenTelemetry monitoring works in an otherwise empty or test cluster, the [OpenTelemetry demo](https://github.com/open-telemetry/opentelemetry-demo) is an example microservice environment with real-world metrics, logs, events and traces from a variety of microservices.

#### 1. Add OpenTelemetry helm repository

We use the OpenTelemetry Helm charts to install the OpenTelemetry Demo. If you haven't already added the repo, run:

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo update
```

#### 2. Edit the demo config to use your access token

Download the file in the `collector/config-k8s/values-oteldemo.yaml` directory and replace `YOUR_TOKEN` with your [Cloud Observability access token](https://docs.lightstep.com/docs/create-and-manage-access-tokens).

#### 3. Deploy the demo environment

This will deploy a microservice environment instrumented for OpenTelemetry metrics, logs, and traces.

```sh
helm upgrade --install my-otel-demo open-telemetry/opentelemetry-demo -f collector/config-k8s/values-oteldemo.yaml
```

#### 4. See data in ServiceNow

In Cloud Observability, you should see metrics, logs, and traces from the demo environment after a few minutes.

🎉