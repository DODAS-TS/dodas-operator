# DODAS Kubernetes operator

Create and delete DODAS clusters as Kubernetes resources (just like pod et al.).

Your deployments will be created and managed by the [InfrastructureManager](https://www.grycap.upv.es/im/index.php)(IM).
To start playing with the operatori we provide a quick start guide with two options:

- using the **[community instance of IM](https://dodas-ts.github.io/dodas-operator/enablingFac/quick-start/)** (required free registration for evaluation purpose [here](https://dodas-iam.cloud.cnaf.infn.it))
- a **[standalone setup](https://dodas-ts.github.io/dodas-operator/standalone/quick-start/)** where IM will be deployed together with the dodas-operator

> **N.B** All of the pre-compiled templates provided by DODAS use the helm charts defined and documented [here](https://github.com/DODAS-TS/helm_charts/tree/master/stable).
>
> Therefore **all the available applications can be installed as they are on top of any k8s instance with [Helm](https://helm.sh/)**

## Developer guide

If you want to contibute please consider the following

### Dev Requirements

- go > 1.12
- [operator-sdk](https://github.com/operator-framework/operator-sdk/blob/master/doc/user-guide.md)

### How to contribute

1. create a branch
2. upload your changes
3. create a pull request

Thanks!

### Render the page using Mkdocs

You will need [mkdocs](https://www.mkdocs.org/) installed on your machine. You can install it with pip:

```bash
pip install mkdocs mkdocs-material
```

To start a real time rendering of the doc just type:

```bash
mkdocs serve
```

The web page generated will be now update at each change you do on the local folder.

## Contact us

DODAS Team provides two support channels, email and Slack channel.

- **mailing list**: send a message to the following list dodas-support@lists.infn.it
- **slack channel**: join us on [Slack Channel](https://dodas-infn.slack.com/archives/CAJ6VG71A)
