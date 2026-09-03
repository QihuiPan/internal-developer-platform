# Versioned Service Templates

The template catalogue is part of the control-plane binary and maps each semantic template identifier to deterministic renderer code. The service descriptor and generated repository both retain the selected identifier.

Publishing a new version requires:

1. a new immutable identifier in `catalog.yaml` and domain validation;
2. renderer behavior that does not modify an existing version;
3. contract tests for every required output;
4. a changelog entry and compatibility notes;
5. a canary generation and build before the version becomes stable.

In the production GitHub adapter, each semantic version must also resolve to a Git commit SHA. Existing services are never silently regenerated; upgrades arrive as pull requests.
