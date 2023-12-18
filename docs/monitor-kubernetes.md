## Monitor Kubernetes with the ServiceNow Collector

| Kuberenetes Distibution                        | Support Status            | Architecture |
| ---------------------------------------------- | ------------------ ------ | ------------ |
| GKE (Google Cloud)                             | last three major versions | ARM, AMD     |
| EKS (AWS)                                      | last three major versions | ARM, AMD     |
| AKS (Azure)                                    | last three major versions | ARM, AMD     |
| Kubernetes                                     | last three major versions | ARM, AMD     |

* **Note:** We recommend Red Hat OpenShift customers should use the [Red Hat OpenTelemetry Distribution](https://docs.openshift.com/container-platform/4.12/otel/otel-using.html).

### Deploy for Kubernetes monitoring with The OpenTelemetry Operator and Helm

> This is an example only. We recommend using the official OpenTelemetry Operator Helm chart for deploying to production.

All of the following assume you are installing the Operator in the `default` cluster namespace and you have `helm` v3 installed.

1. Add OpenTelemetry Helm Charts.
  - ```sh
    helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
    helm repo update
    ```

2. Install charts. This installs the Operator with an automatically-generated self-signed certificate. For other options, see ["TLS Certificate Requirement"](https://github.com/open-telemetry/opentelemetry-helm-charts/tree/main/charts/opentelemetry-operator#tls-certificate-requirement) in OpenTelemetry Operator documentation.
  - ```sh
    helm install \
        --set admissionWebhooks.certManager.enabled=false \
        --set admissionWebhooks.certManager.autoGenerateCert=true \
        opentelemetry-operator open-telemetry/opentelemetry-operator
    ```

3. Set credentials for your ServiceNow instance and Cloud Observability.
  - ```sh
    export LS_TOKEN='<your-token>'
    kubectl create secret generic ls-token-secret -n default --from-literal='LS_TOKEN=$LS_TOKEN'

    # Set password for Event Manangement user on your instances
    export MID_INSTANCE_EVENTS_PASSWORD='<your-mid-user-pw>'
    kubectl create secret generic mid-instance-events-password -n default --from-literal='MID_INSTANCE_EVENTS_PASSWORD=$MID_INSTANCE_EVENTS_PASSWORD'
    ```

4. Deploy an OpenTelemetry Collector. The following uses the deployment example from the `collector/examples/` directory. Before applying you *must* edit the `k8s-deployment.yaml` file and set your username, instance URL, and cluster name. 

  - ```sh
    kubectl apply -f collector/examples/k8s-deployment.yaml
    ```
