# KOM Operator

[The Kubernetes Operator for Microservices aka KOM Operator](https://operatorhub.io/operator/kom-operator),  is designed to manage all the lifecycle of a HTTP based microservice including deploying/updating, load balancing and auto scaling.

The KOM Operator is using these services under the hood to manage deployments:
- [Traefik proxy](https://github.com/traefik/traefik) for load balancing 
- The kubernate horizontal pod autoscaler for autoscaling.

## Quick start

### Install
 The KOM Operator can be installed through many ways:
 - Helm:  [see instructions here](https://artifacthub.io/packages/helm/kaiso/kom-operator)
 - Olm **(Deprecated)**:  [see instructions here](https://artifacthub.io/packages/olm/community-operators/kom-operator)
### Use the KOM Operator:
  In order to deploy your microservice using the KOM Operator simply create a yaml descriptor file, the following example
deploys nginx. routing, volumes & autoscaling are optional.
```
apiVersion: kom.kaiso.github.io/v1alpha1
kind: Microservice
metadata:
    name: kom-nginx
    namespace: default
    labels:
        deploymentName: kaiso-nginx
spec:
  nodeSelector:
    kubernetes.io/os: linux
  container:
    image: nginx:latest
    args:
        - 'nginx-debug'
        -  '-g'
        - 'daemon off;'
    routing:
          http:
            - port: 
                containerPort: 8019
                name: "https"
              rule: "PathPrefix(`/nginxhttps`)"
            - port: 
                containerPort: 80
                name: "http"
              rule: "Host(`nginx.kaiso.local`)"
    volumeMounts:
            - mountPath: /var/log/
              name: logvolume
            - mountPath: /usr/share/nginx/html/
              name: contentvolume
    resources:
      requests:
          memory: 50Mi
          cpu: 100m
      limits:
          memory: 200Mi
          cpu: 500m
  autoscaling:
      min: 2
      max: 5
      scaler:
          - resource: CPU
            type: Utilization
            value: "75"
  volumes:
      - name: logvolume
        hostPath:
            path: /Volumes/DATA/tmp/log/
            type: DirectoryOrCreate
      - name: contentvolume
        hostPath:
            path: /Volumes/DATA/tmp/mdrn/engine/buildprod/
            type: DirectoryOrCreate
```
Check the created objects:
 ```
  kubectl get pods
  kubectl get hpa
 ```
 Go to your browser to see if the routing was applied on [http://localhost/web](http://localhost/web)
## Development
### Local run
To run the operator locally outside the cluster
```
# Install CRDs
make install
# run locally
make run
```
Uninstall CRDs
```
make uninstall
```
### Release
#### Update Version
the version must be updated in three files:
- version/version.go
- Makefile
- helm/kom/operator/Chart.yaml
#### Build docker image
```
  make docker-build
```
#### Install helm chart locally
```
  cd helm
  helm lint kom-operator
  helm install --set loadbalancer.replicaCount=2 kom-operator ./kom-operator
```
#### Build helm chart
```
  cd helm
  helm lint kom-operator
  helm package kom-operator
```
