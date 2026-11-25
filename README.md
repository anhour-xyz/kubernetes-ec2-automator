# Kubernetes EC2 Infrastructure Operator ☁️

## Overview

Kubernetes EC2 Automator is a Kubernetes infrastructure operator built with Go and Kubebuilder. It provides two custom resources:

- `EC2Instance` manages the creation, monitoring, and termination of AWS EC2 instances through the AWS SDK for Go.
- `ManagedWorkload` creates Kubernetes Deployments, monitors workload health, repairs replica drift, and recreates deleted resources.

The project also exposes Prometheus metrics and includes an automated failure-injection suite covering pod deletion, Deployment deletion, and replica drift.

<br>

<ins> Tech Stack: Go, Kubernetes, Docker, Prometheus, Linux, AWS EC2 </ins>

<br>

## Infrastructure Architecture

```text
                         Developer
                             │
              kubectl apply -f ec2instance.yaml
                             │
                             ▼
                ┌──────────────────────┐
                │ Kubernetes API Server│
                │                      │
                │ • Validate CRD       │
                │ • Authentication/RBAC│
                └──────────┬───────────┘
                           │
                           ▼
                      ┌────────┐
                      │  etcd  │
                      │        │
                      │ spec   │
                      │ status │
                      └────┬───┘
                           │
                     Watch Event
                           │
                           ▼
        ┌──────────────────────────────────┐
        │      EC2Instance Controller      │
        │        Go + Kubebuilder          │
        │                                  │
        │       Reconciliation Loop        │
        │                                  │
        │  1. Read desired state           │
        │  2. Query actual AWS state       │
        │  3. Compare states               │
        │  4. Execute required action      │
        │  5. Update resource status       │
        └───────────────┬──────────────────┘
                        │
                   AWS SDK for Go
                        │
                        ▼
              ┌─────────────────────┐
              │      AWS EC2 API    │
              │                     │
              │ • RunInstances      │
              │ • DescribeInstances │
              │ • TerminateInstances│
              └──────────┬──────────┘
                         │
                         ▼
                  ┌─────────────┐
                  │EC2 Instances│
                  └──────┬──────┘
                         │
                   Actual State
                         │
                         ▼
              ┌────────────────────┐
              │ Status Sync        │
              │                    │
              │ instanceId         │
              │ state              │
              │ IP / conditions    │
              └─────────┬──────────┘
                        │
                        └──────────────► Kubernetes API


        ───────── Supporting Infrastructure ─────────

        Reliability                 Observability
        • Idempotency               • Prometheus
        • Finalizers                • Metrics
        • Retry / Backoff           • Logs
        • Error Handling            • Health Checks

                       Security
                    • Kubernetes RBAC
                    • AWS IAM

```

<br>

## API Reference

### ManagedWorkload

| Spec field | Behavior |
| --- | --- |
| `image` | Container image for the application. |
| `replicas` | Desired replica count. The controller restores this value after direct changes to the Deployment. |
| `port` | A positive port enables HTTP startup, readiness, and liveness probes on `/`. |
| `env` | Optional map of environment variable names to string values. |

Status includes the `Healthy` or `Recovering` phase, an `Available` condition, ready and available replica counts, and recovery count and timing fields. The controller considers a workload healthy when the number of available replicas meets the desired count. Initial startup can count as a recovery.

### EC2Instance

| Spec field | Behavior |
| --- | --- |
| `region` | AWS region used to launch, describe, and terminate the instance. |
| `amiID` | AMI used to launch the instance. Choose an AMI available in the specified region. |
| `instanceType` | EC2 instance type passed to the launch request. |
| `instanceName` | Required by the schema. The launch code currently leaves the EC2 Name tag unset. |
| `sshKey`, `subnet`, `tags`, `storage`, `additionalStorage` | Declared in the schema. |

<br>

## Results

![AWS EC2 Instance Provisioned by the Kubernetes Operator](image.png)

The `EC2Instance` controller provisioned an EC2 instance and synchronized its instance ID, state, IP addresses, and DNS information into Kubernetes custom-resource status.

<br>

![Prometheus CPU Monitoring Across 20 Managed Workloads](image-1.png)

Prometheus collected container CPU metrics from 20 workloads distributed across a three-node Kubernetes cluster.

<!-- Add the memory screenshot here. -->

<br>

![Automated Kubernetes Workload Recovery Tests](image-2.png)

The automated failure-injection suite completed **30/30 recovery tests successfully**:

- 10 Pod-deletion scenarios
- 10 Deployment-deletion scenarios
- 10 replica-drift scenarios




