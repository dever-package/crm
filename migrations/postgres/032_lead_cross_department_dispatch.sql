BEGIN;

CREATE TABLE IF NOT EXISTS gjj_crm_lead_dispatch_route (
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT NOT NULL,
    status SMALLINT NOT NULL DEFAULT 2,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_lead_dispatch_route_workflow
    ON gjj_crm_lead_dispatch_route (workflow_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_lead_dispatch_route_status
    ON gjj_crm_lead_dispatch_route (status, workflow_id, id);

CREATE TABLE IF NOT EXISTS gjj_crm_lead_dispatch_handoff (
    id BIGSERIAL PRIMARY KEY,
    lead_id BIGINT NOT NULL,
    workflow_instance_id BIGINT NOT NULL,
    source_workflow_id BIGINT NOT NULL,
    source_stage_id BIGINT NOT NULL,
    source_department_id BIGINT NOT NULL,
    target_workflow_id BIGINT NOT NULL,
    target_stage_id BIGINT NOT NULL,
    target_department_id BIGINT NOT NULL,
    assignee_staff_id BIGINT NOT NULL DEFAULT 0,
    dispatch_type VARCHAR(32) NOT NULL DEFAULT '',
    operator_staff_id BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    completed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_gjj_crm_lead_dispatch_handoff_instance
    ON gjj_crm_lead_dispatch_handoff (workflow_instance_id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_lead_dispatch_handoff_lead_status
    ON gjj_crm_lead_dispatch_handoff (lead_id, status, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_lead_dispatch_handoff_source_status
    ON gjj_crm_lead_dispatch_handoff (
        source_department_id,
        source_workflow_id,
        status,
        created_at,
        id
    );
CREATE INDEX IF NOT EXISTS idx_gjj_crm_lead_dispatch_handoff_target_status
    ON gjj_crm_lead_dispatch_handoff (target_department_id, status, created_at, id);
CREATE INDEX IF NOT EXISTS idx_gjj_crm_lead_dispatch_handoff_assignee_complete
    ON gjj_crm_lead_dispatch_handoff (assignee_staff_id, completed_at, id);

COMMIT;
