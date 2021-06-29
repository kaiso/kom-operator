# The KOM Operator
## Production grade kubernetes operator for microservices
 The Kubernetes Operator for Microservices aka KOM Operator is designed to manage all the lifecycle of a HTTP based
microservice including deploying/updating, load balancing and auto scaling.

### Usage
#### Prerequisites
 The KOM Operator runs on the kube-system namespace which requires that the user have the required privileges.
#### Installing
  The KOM Operator may be installed using OLM or directly applying the deploy.yaml
#### Example usage
  After deploying the KOM Operator you can submit a microservice following the example above which uses nginx
as a microservice
