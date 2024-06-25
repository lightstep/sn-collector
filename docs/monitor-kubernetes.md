## Monitor Kubernetes with the ServiceNow Collector and CNO

Below are instructions on monitoring one of the following Kubernetes cluster environments with ServiceNow.

| Kuberenetes Distibution                        | Support Status            | Architecture |
| ---------------------------------------------- | ------------------------- | ------------ |
| GKE (Google Cloud)                             | last three major versions | ARM, AMD     |
| EKS (AWS)                                      | last three major versions | ARM, AMD     |
| AKS (Azure)                                    | last three major versions | ARM, AMD     |
| Kubernetes                                     | last three major versions | ARM, AMD     |

**Note:** We recommend Red Hat OpenShift customers use the [Red Hat OpenTelemetry Distribution](https://docs.openshift.com/container-platform/4.15/otel/otel-installing.html).


### Deploy the collector and CNO

To monitor the cluster, make sure you have the following before proceeding:

* a ServiceNow Washingtion DC instance with Discovery and Service Mapping Patterns and Agent Client Collector for Visibility version 3.4.0 or higher
* `helm` v3 installed locally to deploy charts
* Kubernetes cluster with local access via `kubectl` with at least 6 GB of memory and active workloads running in your cluster (no workloads or a test cluster? [See below for deploying the OpenTelemetry demo](#optional-run-the-opentelemetry-demo))
* ability to pull from the public Docker image repository `ghcr.io/lightstep/sn-collector`
* `ClusterRole` permissions in your cluster
* a ServiceNow user with `discovery_admin`, `evt_admin`, and `mid_server` roles.

#### 1. Add OpenTelemetry and ServiceNow helm repositories

We use the OpenTelemetry Helm charts to configure collectors for Kubernetes monitoring. Helm charts make it easy to deploy and configure Kubernetes manifests.

```sh
helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
helm repo add servicenow https://install.service-now.com/glide/distribution/builds/package/informer/informer-helm/repo
helm repo update
```

#### 2. Create a ServiceNow namespace

This namespace is where the OpenTelemetry and CNO components will live in your cluster.

```sh
kubectl create namespace servicenow
```

#### 3. Set credentials

Multiple instance URLs and credentials are needed to send data to HLA, Event Management, Cloud Observability, and the MID server.

First, set username, password and URL credentials for sending events to Event Management with a user that has the `evt_admin` role. The URL __must__ be network accessible from the cluster. We recommend using the generic event endpoint: `/api/global/em/jsonv2`.

```sh
    kubectl create configmap servicenow-events -n servicenow --from-literal=url=https://INSTANCE_NAME.service-now.com/api/global/em/jsonv2

    kubectl create secret generic servicenow-events -n servicenow --from-literal=.user=USERNAME --from-literal=.password=PASSWORD 
```

Next, set the username and password with a user with the `discovery_admin` role for CNO.

```sh
    kubectl create secret generic k8s-informer-cred-INSTANCE_NAME -n servicenow --from-literal=.user=USERNAME --from-literal=.password=PASSWORD
```

Next, set the password of the user for the mid server using a file.

```sh
    echo "mid.instance.password=<YOUR_MID_USER_PASSWORD>" > mid.properties
    kubectl create secret generic servicenow-mid-secret --from-file=mid.properties -n servicenow
```

Next, set the username and password for the MID **webserver** extension user. Note these are webserver-only basic auth credentials are **different** from your instance user credentials.

```sh
    kubectl create secret generic servicenow-mid-webserver -n servicenow --from-literal=.user=USERNAME --from-literal=.password=PASSWORD 
```

Finally, set a Cloud Observability token. [Visit Cloud Observability docs for instructions](https://docs.lightstep.com/docs/create-and-manage-access-tokens) on generating an access token for your project.

```sh
kubectl create secret generic servicenow-cloudobs-token -n servicenow --from-literal=token=YOUR_CLOUDOBS_TOKEN
```

#### 4. Deploy the MID server and configure Metric Intelligence

```sh
    # update the template file with your instance name and MID username and create a new manifest file.
    sed -e 's/__INSTANCE_NAME__/YOUR_INSTANCE_NAME/' -e 's/__USERNAME__/YOUR_USERNAME/' mid-statefulset.yaml > mid.yaml
    kubectl apply -f mid.yaml
```

The MID server should appear on your instance after a few minutes. After it does, perform validation, setup metric intelligence and setup a REST Listener in the MI Metric extension.

#### 5. Deploy ServiceNow Collector for Cluster Monitoring and CNO for Visibility

ServiceNow CMDB generally requires a Kubernetes cluster name to be set. Since this varies depending on the type of cluster, set the name manually in a configuration map:

```sh
    kubectl create configmap cluster-info -n servicenow --from-literal=name=YOUR_CLUSTER_NAME
```

You're now ready to deploy a collector to your cluster to collect cluster-level metrics and events. To preview the generated manifest before deploying, add the `--dry-run` option to the below command:

```sh
helm upgrade otel-collector-cluster open-telemetry/opentelemetry-collector --install --namespace servicenow --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-cluster.yaml
```

Next, install CNO for visibility. Additional install instructions for CNO are on the ServiceNow documentation [portal](https://docs.servicenow.com/bundle/washingtondc-it-operations-management/page/product/cloud-native-operations-visibility/task/cnov-deploy-install.html). By sending `Y` you accept the terms and conditions of ServiceNow CNO.

```sh
helm upgrade k8s-informer servicenow/k8s-informer-chart \ 
    --set acceptEula=Y --set instance.name=INSTANCE_NAME --set clusterName="CLUSTER_NAME" \
    --install --namespace servicenow
```

The pod will deploy after a few seconds, to check status and for errors, run:

```sh
kubectl get pods -n servicenow
```

#### 6. Deploy ServiceNow Collector for Node and Workloads Monitoring

Next, deploy collectors to each Kubernetes host to get workload metrics (via Kubelet). To preview the generated manifest before deploying, add the `--dry-run` option to the below command:

```sh
helm upgrade otel-collector open-telemetry/opentelemetry-collector --install --namespace servicenow --values https://raw.githubusercontent.com/lightstep/sn-collector/main/collector/config-k8s/values-node.yaml
```

#### 6. See events in ServiceNow

If all went well, Kubernetes events will be sent to ServiceNow and Cloud Observability. To send Kubernetes metrics, see instructions below on deploying a MID server.

🎉

## Run the OpenTelemetry demo in your cluster

If you just want to see how OpenTelemetry monitoring works in an otherwise empty or test cluster, the [OpenTelemetry demo](https://github.com/open-telemetry/opentelemetry-demo) is an example microservice environment with real-world metrics, logs, events and traces from a variety of microservices.

### 1. Add OpenTelemetry helm repository

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

## Inject failures into a demo/test cluster 

To simulate some interesting events in the demo cluster, you can use the [chaoskube](https://github.com/linki/chaoskube?tab=readme-ov-file#helm) Helm chart.

## Experimental: Deploy the MID Server to a cluster configured for Metric Intelligence

Set the password for a user that can connect to the MID on your instance.
```sh
    echo "mid.instance.password=<YOUR_MID_USER_PASSWORD>" > mid.secret
    kubectl create secret generic servicenow-mid-secret --from-file=mid.secret -n servicenow
```

Manually download and edit the file to specify your username and instance URL, then apply. Note the cluster must have at least 4GB of free memory and 2 CPUs.

```sh
    # edit the username and instance URL before applying
    kubectl apply -f collector/config-k8s/mid-statefulset.yaml
```

After a few minutes the MID server should appear under MID > Servers on your instance. Validate and [Enable the REST Listener](https://docs.servicenow.com/bundle/washingtondc-it-operations-management/page/product/event-management/task/auto-setup.html) so the MID Server can accept metrics.

If all goes well, the following command should return a 401 error:

```sh
    kubectl port-forward servicenow-mid-statefulset-0 8097:8097 -n servicenow
    curl http://localhost:8097/api/mid/sa/metrics
```

Set a configuration map variable to reference the MID server URL:

```sh
    kubectl create configmap servicenow-mid-url -n servicenow --from-literal=url=http://servicenow-mid:8097/api/mid/sa/metrics
```

Set the MID webserver username:

```sh
    kubectl create configmap servicenow-mid-webserver-user -n servicenow --from-literal=username=WEBSERVER_USERNAME
```

Set the MID webserver password:

```sh
    kubectl create secret generic servicenow-mid-webserver-pass -n servicenow --from-literal="password=YOUR_PASSWORD"
```
