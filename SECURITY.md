# Security Policy

## Supported versions

Security fixes are applied to the latest published release and the `main` branch.

## Reporting a vulnerability

Do not open a public issue for a suspected vulnerability. Use GitHub's **Report a vulnerability** option in the repository Security tab to send a private report. Include the affected version, deployment mode, reproduction steps, impact, and any suggested mitigation.

Maintainers should acknowledge a complete report within five business days. Public disclosure is coordinated after a fix or mitigation is available.

## Deployment boundary

The default demo authentication mode is for a trusted local machine only. Use token mode for a small shared evaluation and place it behind TLS. Internet-facing, multi-team, or regulated deployments require verified OIDC identities, centralized secrets, a multi-replica durable store, tenant authorization, and organization-specific security review.

Never commit API tokens, provider credentials, generated state, Terraform state, or customer data. Release archives contain binaries, the README, and the license only.
