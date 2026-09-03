# Twelve-Minute Demo

## 0–2 minutes: product and architecture

Explain the developer, platform engineer, service owner, and security reviewer outcomes. Show the architecture diagram and the rule that desired state is committed before side effects.

## 2–5 minutes: create a service

Start the API and portal, submit `payments-notifier`, and inspect the operation. Open the generated repository, CI workflow, CODEOWNERS, and recorded service descriptor.

## 5–7 minutes: plan and GitOps

Show the deterministic resource plan and Kubernetes manifest. Point out that the production adapters are explicit follow-up work, not simulated as successful cloud mutations.

## 7–9 minutes: policy gate

Apply the Kyverno policy in a disposable cluster and submit a privileged Pod. Show admission rejecting the workload, then show the compliant generated deployment.

## 9–11 minutes: fault and recovery

Restart with `PLATFORM_FAIL_AT_STEP=render`, submit a second service, and show `FAILED`. Restart without the failpoint, retry with a reason, and show the earlier completed checkpoints being preserved.

## 11–12 minutes: evidence and judgment

Show test output, benchmark evidence, SLO measurement gaps, accepted MVP risks, and the production-evolution list.
