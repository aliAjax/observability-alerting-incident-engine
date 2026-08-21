CREATE TABLE IF NOT EXISTS ingestion_events (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    source text NOT NULL,
    event_type text NOT NULL,
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    numeric_value double precision NOT NULL DEFAULT 0,
    text_value text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    dedupe_key text,
    trace_id text NOT NULL DEFAULT '',
    UNIQUE (dedupe_key)
);
CREATE INDEX IF NOT EXISTS idx_ingestion_source_time ON ingestion_events(source, event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingestion_labels ON ingestion_events USING gin(labels);

CREATE TABLE IF NOT EXISTS rules (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    data_source text NOT NULL,
    event_type text NOT NULL DEFAULT 'metric',
    rule_type text NOT NULL,
    condition jsonb NOT NULL DEFAULT '{}'::jsonb,
    severity text NOT NULL DEFAULT 'warning',
    enabled boolean NOT NULL DEFAULT true,
    mode text NOT NULL DEFAULT 'active',
    window_seconds bigint NOT NULL DEFAULT 300,
    interval_seconds bigint NOT NULL DEFAULT 60,
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    group_by jsonb NOT NULL DEFAULT '[]'::jsonb,
    suppress_by jsonb NOT NULL DEFAULT '[]'::jsonb,
    channels jsonb NOT NULL DEFAULT '[]'::jsonb,
    escalation jsonb NOT NULL DEFAULT '[]'::jsonb,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (tenant, name)
);
CREATE INDEX IF NOT EXISTS idx_rules_eval ON rules(tenant, enabled, mode, updated_at);
CREATE INDEX IF NOT EXISTS idx_rules_source ON rules(tenant, data_source);

CREATE TABLE IF NOT EXISTS alerts (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    rule_id text NOT NULL,
    fingerprint text NOT NULL UNIQUE,
    scope text NOT NULL DEFAULT '',
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending',
    severity text NOT NULL DEFAULT 'warning',
    message text NOT NULL DEFAULT '',
    current_value double precision NOT NULL DEFAULT 0,
    started_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    last_fired_at timestamptz NOT NULL,
    resolved_at timestamptz,
    acknowledged_at timestamptz,
    silenced_at timestamptz,
    fired_count integer NOT NULL DEFAULT 0,
    version integer NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_alerts_tenant_status ON alerts(tenant, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_rule ON alerts(tenant, rule_id);

CREATE TABLE IF NOT EXISTS alert_observations (
    id text PRIMARY KEY,
    alert_id text NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    observed_at timestamptz NOT NULL,
    value double precision NOT NULL DEFAULT 0,
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_event_id text NOT NULL DEFAULT '',
    sequence integer NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_alert_observations_time ON alert_observations(alert_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS alert_transitions (
    id text PRIMARY KEY,
    alert_id text NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
    from_status text NOT NULL,
    to_status text NOT NULL,
    reason text NOT NULL DEFAULT '',
    actor text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL,
    trace_id text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_alert_transitions_time ON alert_transitions(alert_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS notification_channels (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    name text NOT NULL,
    type text NOT NULL,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_templates (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    name text NOT NULL,
    channel_type text NOT NULL,
    subject text NOT NULL DEFAULT '',
    body text NOT NULL,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS notification_tasks (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    alert_id text NOT NULL,
    rule_id text NOT NULL,
    channel_id text NOT NULL,
    template_id text NOT NULL DEFAULT '',
    receivers jsonb NOT NULL DEFAULT '[]'::jsonb,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    available_at timestamptz NOT NULL,
    last_error text NOT NULL DEFAULT '',
    last_attempt_at timestamptz,
    cooldown_until timestamptz,
    escalation_step integer NOT NULL DEFAULT 0,
    locked_by text NOT NULL DEFAULT '',
    locked_until timestamptz,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_status ON notification_tasks(status, available_at, created_at);
CREATE INDEX IF NOT EXISTS idx_notification_tasks_tenant ON notification_tasks(tenant, created_at DESC);

CREATE TABLE IF NOT EXISTS schedules (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    timezone text NOT NULL DEFAULT 'UTC',
    enabled boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS schedule_shifts (
    id text PRIMARY KEY,
    schedule_id text NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    assignee text NOT NULL,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_schedule_shifts_time ON schedule_shifts(starts_at, ends_at);

CREATE TABLE IF NOT EXISTS silences (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    rule_id text NOT NULL DEFAULT '',
    scope text NOT NULL DEFAULT '',
    matchers jsonb NOT NULL DEFAULT '{}'::jsonb,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    created_by text NOT NULL DEFAULT '',
    comment text NOT NULL DEFAULT '',
    active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_silences_active ON silences(tenant, active, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS incidents (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    alert_id text NOT NULL DEFAULT '',
    title text NOT NULL,
    description text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'open',
    severity text NOT NULL DEFAULT 'medium',
    assignee text NOT NULL DEFAULT '',
    labels jsonb NOT NULL DEFAULT '{}'::jsonb,
    opened_at timestamptz NOT NULL,
    closed_at timestamptz,
    updated_at timestamptz NOT NULL,
    version integer NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_incidents_tenant_status ON incidents(tenant, status, opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_alert ON incidents(tenant, alert_id);

CREATE TABLE IF NOT EXISTS incident_actions (
    id text PRIMARY KEY,
    incident_id text NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    action text NOT NULL,
    actor text NOT NULL DEFAULT '',
    comment text NOT NULL DEFAULT '',
    occurred_at timestamptz NOT NULL,
    trace_id text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_incident_actions_time ON incident_actions(incident_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS audit_records (
    id text PRIMARY KEY,
    tenant text NOT NULL DEFAULT 'default',
    entity text NOT NULL,
    entity_id text NOT NULL DEFAULT '',
    action text NOT NULL,
    actor text NOT NULL DEFAULT '',
    detail jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL,
    trace_id text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_audit_records_lookup ON audit_records(tenant, entity, entity_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS queue_items (
    id text PRIMARY KEY,
    topic text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    max_attempts integer NOT NULL DEFAULT 5,
    available_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL,
    locked_by text NOT NULL DEFAULT '',
    locked_until timestamptz,
    last_error text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_queue_items_poll ON queue_items(topic, status, available_at, created_at);
