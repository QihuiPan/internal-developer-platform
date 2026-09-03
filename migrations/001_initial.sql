BEGIN;

CREATE TYPE operation_status AS ENUM ('PENDING', 'VALIDATING', 'PLANNING', 'APPLYING', 'VERIFYING', 'SUCCEEDED', 'FAILED', 'RETRYING');

CREATE TABLE services (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  owner_team_id TEXT NOT NULL,
  lifecycle TEXT NOT NULL DEFAULT 'experimental',
  template_version TEXT NOT NULL,
  descriptor JSONB NOT NULL,
  status TEXT NOT NULL,
  deleted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE environments (
  id UUID PRIMARY KEY,
  service_id UUID NOT NULL REFERENCES services(id),
  name TEXT NOT NULL,
  cluster TEXT NOT NULL,
  namespace TEXT NOT NULL,
  status TEXT NOT NULL,
  UNIQUE (service_id, name)
);

CREATE TABLE resources (
  id UUID PRIMARY KEY,
  environment_id UUID NOT NULL REFERENCES environments(id),
  type TEXT NOT NULL,
  desired_spec JSONB NOT NULL,
  desired_spec_hash TEXT NOT NULL,
  actual_ref TEXT,
  UNIQUE (environment_id, type, desired_spec_hash)
);

CREATE TABLE operations (
  id UUID PRIMARY KEY,
  type TEXT NOT NULL,
  target_id UUID NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  request_hash TEXT NOT NULL,
  status operation_status NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  error JSONB,
  lease_owner TEXT,
  lease_expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE operation_steps (
  operation_id UUID NOT NULL REFERENCES operations(id),
  name TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  error JSONB,
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  PRIMARY KEY (operation_id, name)
);

CREATE TABLE audit_events (
  id UUID PRIMARY KEY,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  target TEXT NOT NULL,
  reason TEXT NOT NULL,
  before_state JSONB,
  after_state JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox (
  id UUID PRIMARY KEY,
  topic TEXT NOT NULL,
  payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  published_at TIMESTAMPTZ
);

CREATE INDEX operations_claimable_idx ON operations (status, lease_expires_at) WHERE status IN ('PENDING', 'RETRYING');
CREATE INDEX outbox_unpublished_idx ON outbox (created_at) WHERE published_at IS NULL;

COMMIT;
