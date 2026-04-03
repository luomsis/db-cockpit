-- Instance metadata table
-- Stores database instance information including configuration, ownership, and infrastructure details

CREATE TABLE IF NOT EXISTS public.instance_meta (
    id bigserial NOT NULL,
    db_type text NOT NULL,
    entity_name text NOT NULL,
    chinese_desc text NOT NULL,
    org_code text NOT NULL,
    service_user text NOT NULL,
    opr_dba text NOT NULL,
    business_owner text NOT NULL,
    alert_subscriber text NOT NULL,
    infra_type text NOT NULL,
    req_cpu numeric(10, 2) NOT NULL,
    req_memory_gb numeric(10, 2) NOT NULL,
    req_storage_gb numeric(10, 2) NOT NULL,
    created_date timestamptz NOT NULL,
    environment text NOT NULL,
    opr_dba_ii text NOT NULL,
    ins_created_date timestamptz NOT NULL,
    ins_updated_date timestamptz NOT NULL,
    host_environment1 text NOT NULL,
    host_environment2 text NOT NULL,
    le_name text NOT NULL,
    instance_endpoint text NOT NULL,
    subsys_code text NOT NULL,
    source_sys text NOT NULL,
    attach_db text NOT NULL,
    host_namel text NOT NULL,
    host_name2 text NOT NULL,
    default_role text NOT NULL,
    "role" text NOT NULL,
    status text NOT NULL,
    version_detail text NOT NULL,
    instance_name text NOT NULL,
    is_created_by_cloud text NOT NULL,
    character_set text NOT NULL,
    instance_vip text NOT NULL,
    instance_port int8 NOT NULL,
    user_name text NOT NULL,
    host_ip1 text NOT NULL,
    host_infra_type1 text NOT NULL,
    os_name text NOT NULL,
    host_ip2 text NOT NULL,
    host_infra_type2 text NOT NULL,
    ha_type text NOT NULL,
    backup_method text NOT NULL,
    failover_type text NOT NULL,
    ins_uuid text NOT NULL,
    ccm_name text NOT NULL,
    CONSTRAINT instance_meta_pkey PRIMARY KEY (id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_instance_meta_db_type ON public.instance_meta(db_type);
CREATE INDEX IF NOT EXISTS idx_instance_meta_environment ON public.instance_meta(environment);
CREATE INDEX IF NOT EXISTS idx_instance_meta_status ON public.instance_meta(status);
CREATE INDEX IF NOT EXISTS idx_instance_meta_instance_name ON public.instance_meta(instance_name);
CREATE INDEX IF NOT EXISTS idx_instance_meta_org_code ON public.instance_meta(org_code);

-- Comments for documentation
COMMENT ON TABLE public.instance_meta IS 'Database instance metadata including configuration, ownership, and infrastructure details';
COMMENT ON COLUMN public.instance_meta.id IS 'Primary key';
COMMENT ON COLUMN public.instance_meta.db_type IS 'Database type (e.g., PostgreSQL, MySQL, Oracle)';
COMMENT ON COLUMN public.instance_meta.entity_name IS 'Entity/business name';
COMMENT ON COLUMN public.instance_meta.chinese_desc IS 'Chinese description';
COMMENT ON COLUMN public.instance_meta.org_code IS 'Organization code';
COMMENT ON COLUMN public.instance_meta.service_user IS 'Service user account';
COMMENT ON COLUMN public.instance_meta.opr_dba IS 'Operational DBA';
COMMENT ON COLUMN public.instance_meta.business_owner IS 'Business owner';
COMMENT ON COLUMN public.instance_meta.alert_subscriber IS 'Alert subscriber';
COMMENT ON COLUMN public.instance_meta.infra_type IS 'Infrastructure type';
COMMENT ON COLUMN public.instance_meta.req_cpu IS 'Requested CPU cores';
COMMENT ON COLUMN public.instance_meta.req_memory_gb IS 'Requested memory in GB';
COMMENT ON COLUMN public.instance_meta.req_storage_gb IS 'Requested storage in GB';
COMMENT ON COLUMN public.instance_meta.created_date IS 'Record creation date';
COMMENT ON COLUMN public.instance_meta.environment IS 'Environment (e.g., prod, staging, dev)';
COMMENT ON COLUMN public.instance_meta.opr_dba_ii IS 'Secondary operational DBA';
COMMENT ON COLUMN public.instance_meta.ins_created_date IS 'Instance creation date';
COMMENT ON COLUMN public.instance_meta.ins_updated_date IS 'Instance last update date';
COMMENT ON COLUMN public.instance_meta.host_environment1 IS 'Primary host environment';
COMMENT ON COLUMN public.instance_meta.host_environment2 IS 'Secondary host environment';
COMMENT ON COLUMN public.instance_meta.le_name IS 'Legal entity name';
COMMENT ON COLUMN public.instance_meta.instance_endpoint IS 'Instance connection endpoint';
COMMENT ON COLUMN public.instance_meta.subsys_code IS 'Subsystem code';
COMMENT ON COLUMN public.instance_meta.source_sys IS 'Source system';
COMMENT ON COLUMN public.instance_meta.attach_db IS 'Attached database name';
COMMENT ON COLUMN public.instance_meta.host_namel IS 'Primary hostname';
COMMENT ON COLUMN public.instance_meta.host_name2 IS 'Secondary hostname';
COMMENT ON COLUMN public.instance_meta.default_role IS 'Default database role';
COMMENT ON COLUMN public.instance_meta.role IS 'Current role';
COMMENT ON COLUMN public.instance_meta.status IS 'Instance status (e.g., active, inactive, maintenance)';
COMMENT ON COLUMN public.instance_meta.version_detail IS 'Database version details';
COMMENT ON COLUMN public.instance_meta.instance_name IS 'Instance name';
COMMENT ON COLUMN public.instance_meta.is_created_by_cloud IS 'Whether created by cloud platform';
COMMENT ON COLUMN public.instance_meta.character_set IS 'Database character set';
COMMENT ON COLUMN public.instance_meta.instance_vip IS 'Instance virtual IP';
COMMENT ON COLUMN public.instance_meta.instance_port IS 'Instance port';
COMMENT ON COLUMN public.instance_meta.user_name IS 'Username';
COMMENT ON COLUMN public.instance_meta.host_ip1 IS 'Primary host IP';
COMMENT ON COLUMN public.instance_meta.host_infra_type1 IS 'Primary host infrastructure type';
COMMENT ON COLUMN public.instance_meta.os_name IS 'Operating system name';
COMMENT ON COLUMN public.instance_meta.host_ip2 IS 'Secondary host IP';
COMMENT ON COLUMN public.instance_meta.host_infra_type2 IS 'Secondary host infrastructure type';
COMMENT ON COLUMN public.instance_meta.ha_type IS 'High availability type';
COMMENT ON COLUMN public.instance_meta.backup_method IS 'Backup method';
COMMENT ON COLUMN public.instance_meta.failover_type IS 'Failover type';
COMMENT ON COLUMN public.instance_meta.ins_uuid IS 'Instance unique identifier';
COMMENT ON COLUMN public.instance_meta.ccm_name IS 'CCM configuration name';