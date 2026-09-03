# Contributing

## Required workflow

1. Create a focused branch.
2. Add or update tests for behavioral changes.
3. Keep code comments, annotations, commit messages, review notes, and documentation in English.
4. Add an English entry under `Unreleased` in `CHANGELOG.md` for every change, including documentation and configuration changes.
5. Run the verification commands from the README.
6. Open a pull request describing the user outcome, risk, test evidence, and rollback plan.

Pull requests without a changelog update fail the `changelog` CI job. A release moves entries from `Unreleased` into a dated semantic-version section and restores an empty `Unreleased` section.

## Engineering rules

- Preserve idempotency across retries and duplicated requests.
- Keep metric labels low-cardinality.
- Never commit secrets, generated state, credentials, or Terraform state.
- Require a recorded reason for privileged or retry mutations.
- Prefer immutable image digests and immutable template versions.
- Add an ADR when a decision changes a durable boundary, security assumption, or operational model.
