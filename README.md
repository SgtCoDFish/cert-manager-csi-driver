# cert-manager-csi

cert-manager-csi is a Container Storage Interface (CSI) driver plugin for
Kubernetes to work along cert-manager. The goal for this plugin is to facilitate
requesting and mounting certificate key pairs to pods seamlessly. This is useful
for facilitating mTLS, or otherwise securing connections of pods with guaranteed
present certificates whilst having all of the features that cert-manager
provides.

This project is experimental.

## Requirements and Installation

This CSI driver plugin makes use of the 'CSI inline volume' feature - Alpha as
of v1.15. As such, Kubernetes must be of at least version v1.15 and have the
following feature gate set;
```
--feature-gates=CSIInlineVolume=true
```

You must have a working installation of cert-manager present on the cluster.
Instructions on how to install cert-manager can be found
[here](https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html).

To install the cert-manager-csi driver, apply the deployment manifests to your
cluster.

```
 $ kubectl apply -f deploy/cert-manager-csi-driver.yaml
```

You can verify the installation has completed correctly by checking the presence
of the CSIDriver resource as well as a CSINode resource present for each node,
referencing `csi.cert-manager.io`.

```
$ kubectl get csidrivers
NAME                     CREATED AT
csi.cert-manager.io   2019-09-06T16:55:19Z

$ kubectl get csinodes -o yaml
apiVersion: v1
items:
- apiVersion: storage.k8s.io/v1beta1
  kind: CSINode
  metadata:
    name: kind-control-plane
    ownerReferences:
    - apiVersion: v1
      kind: Node
      name: kind-control-plane
...
  spec:
    drivers:
    - name: csi.cert-manager.io
      nodeID: kind-control-plane
      topologyKeys: null
...
```

The CSI driver is now installed and is ready to be used for pods in the cluster.

## Requesting and Mounting Certificates

To request certificates from cert-manager, simply define a volume mount where
the key and certificate will be written to, along with a volume with attributes
that define the cert-manager request. The following is a dummy app that mounts a
key certificate pair to `/tls` and has been signed by the `ca-issuer` with a
DNS name valid for `my-service.sandbox.svc.cluster.local`.

```
apiVersion: v1
kind: Pod
metadata:
  name: my-csi-app
  namespace: sandbox
  labels:
    app: my-csi-app
spec:
  containers:
    - name: my-frontend
      image: busybox
      volumeMounts:
      - mountPath: "/tls"
        name: tls
      command: [ "sleep", "1000000" ]
  volumes:
    - name: tls
      csi:
        driver: csi.cert-manager.io
        volumeAttributes:
              csi.cert-manager.io/issuer-name: ca-issuer
              csi.cert-manager.io/dns-names: my-service.sandbox.svc.cluster.local
```

Once created, the CSI driver will generate a private key locally, request a
certificate from cert-manager based on the given attributes, then store both
locally to be mounted to the pod. The pod will remain in a pending state until
this process has been completed.

For more information on how to set up issuers for your cluster, refer to the
cert-manager documentation
[here](https://docs.cert-manager.io/en/latest/tasks/issuers/index.html).

## Supported Volume Attributes

The cert-manager-csi driver aims to have complete feature parity with all
possible values availble through the cert-manager API however currently supports
the following values;

| Attribute                            | Description                                                                                          | Default         | Example                        |
|--------------------------------------|------------------------------------------------------------------------------------------------------|-----------------|--------------------------------|
| csi.cert-manager.io/issuer-name      | The Issuer name to sign the certificate request.                                                     |                 | ca-issuer                      |
| csi.cert-manager.io/issuer-kind      | The Issuer kind to sign the certificate request.                                                     | Issuer          | ClusterIssuer                  |
| csi.cert-manager.io/issuer-group     | The group name the Issuer belongs to.                                                                | cert-manager.io | out.of.tree.foo                |
| csi.cert-manager.io/common-name      | Certificate common name. A common name or at least one DNS name must be set.                         |                 | my-cert.foo                    |
| csi.cert-manager.io/dns-names        | DNS names the certificate will be requested for. At least a common name or one DNS name must be set. |                 | a.b.foo.com,c.d.foo.com        |
| csi.cert-manager.io/ip-sans          | IP addresses the certificate will be requested for.                                                  |                 | 192.0.0.1,192.0.0.2            |
| csi.cert-manager.io/uri-sans         | URI names the certificate will be requested for.                                                     |                 | spiffe://foo.bar.cluster.local |
| csi.cert-manager.io/duration         | Requested duration the signed certificate will be valid for.                                         | 720h            | 1880h                          |
| csi.cert-manager.io/is-ca            | Mark the certificate as a certificate authority.                                                     | false           | true                           |
| csi.cert-manager.io/certificate-file | File name to store the certificate file at.                                                          | crt.pem         | bar/foo.crt                    |
| csi.cert-manager.io/privatekey-file  | File name to store the key file at.                                                                  | key.pem         | bar/foo.key                    |
| csi.cert-manager.io/namespace        | The namespace to use when creating the CertificateRequest.                                           | $PodNamespace   | my-namespace                   |

## Design Documents
 - [Certificate Renewal](./docs/design/20190914.certificaterenewal.md)
