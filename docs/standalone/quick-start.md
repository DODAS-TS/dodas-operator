# Standalone installation

## Requirements

- a k8s instance
  - kubectl available
  - for local test you can run k8s in docker with [KinD](https://kind.sigs.k8s.io/) or [k3d](https://github.com/rancher/k3d)
- access to a cloud provider

## Install the infrastructure manager

Setup the operator and the dodas infrastructure CustomResource definition as follow:

```bash
git clone https://github.com/dodas-ts/dodas-operator
cd dodas-operator
kubectl apply -f deploy/im.yaml
```

and check that there are no failures with:

```bash
kubectl get pod
```

## Install the operator

Setup the operator and the dodas infrastructure CustomResource definition as follow:

```bash
# create infrastructure custom resources
kubectl apply -f deploy/crds/infrastructures_crd.yaml

# create service account and roles for operator
kubectl apply -f deploy/service_account.yaml
kubectl apply -f deploy/role.yaml
kubectl apply -f deploy/role_binding.yaml

# deploy the operator
kubectl apply -f deploy/operator.yaml
```

And then just check if everythin went fine with:

```bash
$ kubectl get pod

NAME                              READY   STATUS    RESTARTS   AGE
dodas-operator-6ff5cbc4ff-kxttr   1/1     Running   0          10s
```

### Create a config map with your tosca template

Let's first try with a simple deployment of a 2 VMs k8s cluster on openstack resources.

> More complex examples and documentation for setting up end-to-end application can be found [here](https://dodas-ts.github.io/dodas-templates/)

First save the content below into `test-deployment.yaml` and fill up the fields between `<>`.

```yaml
tosca_definitions_version: tosca_simple_yaml_1_0

imports:
  - dodas_types: https://raw.githubusercontent.com/dodas-ts/dodas-templates/master/tosca-types/dodas_types.yml

description: TOSCA template for a complete CMS computing cluster on top of K8s orchestrator

topology_template:
  inputs:
    number_of_masters:
      type: integer
      default: 1

    num_cpus_master:
      type: integer
      default: 4

    mem_size_master:
      type: string
      default: "8 GB"

    number_of_slaves:
      type: integer
      default: 1

    num_cpus_slave:
      type: integer
      default: 4

    mem_size_slave:
      type: string
      default: "8 GB"

    server_image:
      type: string
      default: <your image here>
      # e.g. "ost://cloud.recas.ba.infn.it/1113d7e8-fc5d-43b9-8d26-61906d89d479"

  node_templates:
    k8s_master:
      type: tosca.nodes.DODAS.FrontEnd.Kubernetes
      properties:
        admin_token: testme
      requirements:
        - host: k8s-master-server

    k8s_slave:
      type: tosca.nodes.DODAS.WorkerNode.Kubernetes
      properties:
        front_end_ip: { get_attribute: [k8s-master-server, private_address, 0] }
      requirements:
        - host: k8s-slave-server

    k8s-master-server:
      type: tosca.nodes.indigo.Compute
      capabilities:
        endpoint:
          properties:
            network_name: PUBLIC
            ports:
              dashboard:
                protocol: tcp
                source: 30443
        scalable:
          properties:
            count: { get_input: number_of_masters }
        host:
          properties:
            num_cpus: { get_input: num_cpus_master }
            mem_size: { get_input: mem_size_master }
        os:
          properties:
            image: { get_input: server_image }

    k8s-slave-server:
      type: tosca.nodes.indigo.Compute
      capabilities:
        endpoint:
          properties:
            network_name: PRIVATE
        scalable:
          properties:
            count: { get_input: number_of_slaves }
        host:
          properties:
            num_cpus: { get_input: num_cpus_slave }
            mem_size: { get_input: mem_size_slave }
        os:
          properties:
            image: { get_input: server_image }

  outputs:
    k8s_endpoint:
      value:
        {
          concat:
            [
              "https://",
              get_attribute: [k8s-master-server, public_address, 0],
              ":30443",
            ],
        }
```

Then you can save it as Kubernetes ConfigMap (also for later use) with:

```bash
kubectl create configmap mytemplate --from-file=test-deployment.yaml
```

### Test a deployment

Create a manifest `my-infra.yml` with DODAS Infrastructure resource specifying the credentials for your cloud provider and the local InfrastructureManager.

> The sintax used is similar to what used for dodas client [here](https://dodas-ts.github.io/dodas-go-client/)

```yaml
apiVersion: dodas.infn.it/v1alpha1
kind: Infrastructure
metadata:
  name: example-infrastructure
spec:
  name: test-infra
  image: dodasts/dodas-client:v0.3.3
  template: mytemplate
  cloud:
    id: ost
    type: OpenStack
    username: <your username>
    password: <your password>
    host: <your cloud provider endpoint>
    # e.g. https://cloud.recas.ba.infn.it:5000/
    tenant: <your tenant>
    auth_version: 3.x_password
    service_region: recas-cloud
  im:
    id: im
    type: InfrastructureManager
    host: http://im-backend.default.svc.cluster.local:8800/infrastructures
    username: test
    password: test
```

Then create the resource in kubernetes with:

```bash
kubectl apply -f my-infra.yml
```

If everything went well you should be able to see you InfrastructureID of the deployment appearing here:

```bash
$ kubectl get infrastructures

NAME                     INFID                                  STATUS
example-infrastructure   9ca8a2ee-41ba-11ea-8ea8-0242ac150003   created
```

### Destroy the deployment

Just type:

```bash
$ kubectl delete infrastructure example-infrastructure

infrastructure.dodas.infn.it "example-infrastructure" deleted
```
