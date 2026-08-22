package model

import (
	"gorm.io/datatypes"
)

/* ================= 集群 / 实例 / 参数 ================= */

type Cluster struct {
	ID       string `gorm:"primaryKey;size:32" json:"id"`
	Name     string `gorm:"size:128" json:"name"`
	Type     string `gorm:"size:32;index" json:"type"`
	Version  string `gorm:"size:64" json:"version"`
	Desc     string `json:"desc"`
	AZ       string `gorm:"size:64" json:"az"`
	Biz      string `gorm:"size:64" json:"biz"`
	Nodes    int    `json:"nodes"`
	Mode     string `gorm:"size:128" json:"mode"`
	CPU      int    `json:"cpu"`
	Mem      int    `json:"mem"`
	Conn     int    `json:"conn"`
	QPS      int    `json:"qps"`
	SyncMode *string `gorm:"size:128" json:"syncMode,omitempty"`
	Zones    datatypes.JSON `json:"zones,omitempty"`
}

type Instance struct {
	ID        string  `gorm:"primaryKey;size:32" json:"id"`
	ClusterID string  `gorm:"size:32;index" json:"-"`
	Name      string  `gorm:"size:128" json:"name"`
	Role      string  `gorm:"size:64" json:"role"`
	IP        string  `gorm:"size:64" json:"ip"`
	Port      int     `json:"port"`
	Status    string  `gorm:"size:16" json:"status"`
	CPU       int     `json:"cpu"`
	Mem       int     `json:"mem"`
	Conn      int     `json:"conn"`
	Ver       string  `gorm:"size:32" json:"ver"`
	Zone      *string `gorm:"size:32" json:"zone,omitempty"`
	HostIP    string  `gorm:"size:64;index" json:"-"`
}

type ClusterParam struct {
	ID        uint    `gorm:"primaryKey" json:"-"`
	Scope     string  `gorm:"size:16;uniqueIndex:idx_param_scope,priority:1" json:"-"` // cluster | tenant
	ClusterID string  `gorm:"size:32;uniqueIndex:idx_param_scope,priority:2" json:"-"`
	TenantID  *string `gorm:"size:32;uniqueIndex:idx_param_scope,priority:3" json:"-"`
	Name      string  `gorm:"size:128;uniqueIndex:idx_param_scope,priority:4" json:"name"`
	Value     string  `gorm:"size:256" json:"value"`
	Range     string  `gorm:"column:param_range;size:128" json:"range"`
	Desc      string  `gorm:"column:param_desc;size:256" json:"desc"`
	Status    string  `gorm:"size:16" json:"status"`
}

type ParamHistory struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ParamID   uint   `gorm:"index" json:"-"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
	ChangedAt int64  `json:"changedAt"`
}

/* ================= PostgreSQL 扩展 ================= */

type PgDatabase struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;uniqueIndex:idx_pgdb_cluster_name,priority:1" json:"-"`
	Name      string `gorm:"size:128;uniqueIndex:idx_pgdb_cluster_name,priority:2" json:"name"`
	Owner     string `gorm:"size:128" json:"owner"`
	Size      string `gorm:"size:32" json:"size"`
	Tables    int    `json:"tables"`
	Conn      int    `json:"conn"`
	ConnLimit int    `json:"connLimit"`
	Status    string `gorm:"size:16" json:"status"`
}

type PgReplica struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;index" json:"-"`
	Instance  string `gorm:"size:128" json:"instance"`
	Role      string `gorm:"size:64" json:"role"`
	DelayMs   int    `json:"delayMs"`
	WalLag    string `gorm:"size:32" json:"walLag"`
	Status    string `gorm:"size:16" json:"status"`
}

/* ================= OceanBase 租户体系 ================= */

type ObTenant struct {
	ID           string          `gorm:"primaryKey;size:32" json:"id"`
	ClusterID    string          `gorm:"size:32;index" json:"-"`
	Name         string          `gorm:"size:128" json:"name"`
	Kind         string          `gorm:"size:16" json:"kind"`
	Mode         string          `gorm:"column:mode;size:16" json:"mode"`
	PrimaryZone  string          `gorm:"size:64" json:"primaryZone"`
	Locality     string          `gorm:"size:128" json:"locality"`
	UnitNum      int             `json:"unitNum"`
	MaxCpu       float64         `json:"maxCpu"`
	UsedCpu      float64         `json:"usedCpu"`
	MaxMemGb     float64         `json:"maxMemGb"`
	UsedMemGb    float64         `json:"usedMemGb"`
	StorageUsed  string          `gorm:"size:32" json:"storageUsed"`
	StorageTotal string          `gorm:"size:32" json:"storageTotal"`
	Units        datatypes.JSON  `json:"units"`
	Whitelist    datatypes.JSON  `json:"whitelist"`
	ConnHint     string          `gorm:"size:256" json:"connHint"`
	Status       string          `gorm:"size:16" json:"status"`
}

type ObTenantDb struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	TenantID string `gorm:"size:32;uniqueIndex:idx_obdb_tenant_name,priority:1" json:"-"`
	Name     string `gorm:"size:128;uniqueIndex:idx_obdb_tenant_name,priority:2" json:"name"`
	Tables   int    `json:"tables"`
	Size     string `gorm:"size:32" json:"size"`
	Conn     int    `json:"conn"`
	Status   string `gorm:"size:16" json:"status"`
}

/* ================= 主机 / 账号 / 会话 / 事务 / 慢SQL ================= */

type Host struct {
	IP        string `gorm:"primaryKey;size:64" json:"ip"`
	Zone      string `gorm:"size:64" json:"zone"`
	Spec      string `gorm:"size:64" json:"spec"`
	OS        string `gorm:"column:os;size:64" json:"os"`
	CPU       int    `json:"cpu"`
	Mem       int    `json:"mem"`
	Disk      int    `json:"disk"`
	DiskTotal int    `json:"diskTotal"`
	ClusterID string `gorm:"size:32" json:"-"`
	Status    string `gorm:"size:16" json:"status"`
}

type InstanceUser struct {
	ID         uint   `gorm:"primaryKey" json:"-"`
	InstanceID string `gorm:"size:32;index" json:"-"` // '' = 全局演示数据
	Username   string `gorm:"size:128" json:"user"`
	Host       string `gorm:"size:64" json:"host"`
	Priv       string `gorm:"size:256" json:"priv"`
	LastLogin  string `gorm:"size:32" json:"lastLogin"`
	Status     string `gorm:"size:16" json:"status"`
}

type RuntimeSession struct {
	SessionID  int64  `gorm:"column:id;primaryKey" json:"id"`
	InstanceID string `gorm:"size:32;index" json:"-"`
	Username   string `gorm:"size:128" json:"user"`
	Host       string `gorm:"size:64" json:"host"`
	Db         string `gorm:"size:128" json:"db"`
	Cmd        string `gorm:"size:32" json:"cmd"`
	Time       string `gorm:"size:32" json:"time"`
	State      string `gorm:"size:64" json:"state"`
	LockInfo   string `gorm:"column:lock_info;size:64" json:"lock"`
	Status     string `gorm:"size:16" json:"status"`
}

type Trx struct {
	ID         string `gorm:"primaryKey;size:32" json:"id"`
	InstanceID string `gorm:"size:32;index" json:"-"`
	Session    int64  `json:"session"`
	Username   string `gorm:"size:128" json:"user"`
	Dur        string `gorm:"size:32" json:"dur"`
	Undo       string `gorm:"size:32" json:"undo"`
	LockRows   string `gorm:"size:32" json:"lockRows"`
	Waiting    string `gorm:"size:8" json:"waiting"`
	Sql        string `gorm:"column:sql_text" json:"sql"`
	Status     string `gorm:"size:16" json:"status"`
}

type SlowSql struct {
	ID         uint   `gorm:"primaryKey" json:"-"`
	InstanceID string `gorm:"size:32;index" json:"-"` // '' = 全局
	Sql        string `gorm:"column:sql_text" json:"sql"`
	Db         string `gorm:"size:128" json:"db"`
	Time       string `gorm:"size:16" json:"time"`
	Rows       string `gorm:"column:rows_scanned;size:32" json:"rows"`
	Count      int    `json:"count"`
}

/* ================= 报告 / 告警 / 指标 / 元统计 ================= */

type Report struct {
	ID        uint   `gorm:"primaryKey" json:"-"`
	ClusterID string `gorm:"size:32;index" json:"-"` // '' = 平台级
	Ico       string `gorm:"size:8" json:"ico"`
	Title     string `gorm:"size:128" json:"title"`
	Desc      string `gorm:"size:256" json:"desc"`
	Date      string `gorm:"size:32" json:"date"`
	Size      string `gorm:"size:16" json:"size"`
}

type AlertRecord struct {
	ID       uint   `gorm:"primaryKey" json:"-"`
	Name     string `gorm:"size:128" json:"name"`
	Severity string `gorm:"size:8" json:"severity"`
	Title    string `gorm:"size:256" json:"title"`
	Time     string `gorm:"size:32" json:"time"`
	Count    int    `json:"count"`
}

type MetricDef struct {
	ID     string  `gorm:"primaryKey;size:32" json:"id"`
	Name   string  `gorm:"size:64" json:"name"`
	Unit   string  `gorm:"size:16" json:"unit"`
	Base   float64 `json:"base"`
	Jitter float64 `json:"jitter"`
}

// meta_stats：概览页的非表格型 mock 常量（db_types / sql_issues / lock_summary / top_anomaly）
type MetaStat struct {
	Key   string         `gorm:"primaryKey;size:32" json:"key"`
	Value datatypes.JSON `json:"value"`
}

/* ================= 监控大盘 ================= */

type Dashboard struct {
	ID          string         `gorm:"primaryKey;size:64" json:"id"`
	Title       string         `gorm:"size:128" json:"title"`
	Description string         `gorm:"size:256" json:"description"`
	Cfg         datatypes.JSON `json:"cfg"`
	Panels      datatypes.JSON `json:"panels"`
	CreatedAt   int64          `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt   int64          `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

/* ================= Chat 会话呈现域（Go 独占读写；agentcluster 只读） ================= */

type ChatSession struct {
	ID        string `gorm:"primaryKey;size:64" json:"id"`
	UserID    string `gorm:"column:user_id;size:64;index" json:"userId"` // MVP 无鉴权统一 anonymous，二期接 SSO 落真实账号
	Title     string `gorm:"size:128" json:"title"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type ChatMessage struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	SessionID string         `gorm:"size:64;index" json:"-"`
	TurnID    string         `gorm:"column:turn_id;size:64;index" json:"turnId,omitempty"` // 空 = 历史导入消息
	Seq       int            `gorm:"index" json:"-"`
	Role      string         `gorm:"size:16" json:"role"`
	Text      string         `json:"text"`
	Thoughts  datatypes.JSON `json:"thoughts"`
	Cards     datatypes.JSON `json:"cards"`
	Status    string         `gorm:"size:16" json:"status"`
}

// ChatTurn：轮次状态机 + 提交幂等 + 配置版本留痕（docs《交互时序与生命周期》§3.2/§5.1）。
// status: running | done | failed | cancelled | interrupted(二期续跑)。
type ChatTurn struct {
	ID              string         `gorm:"primaryKey;size:64" json:"id"`
	SessionID       string         `gorm:"size:64;index" json:"-"`
	Seq             int            `gorm:"index" json:"-"`
	Kind            string         `gorm:"size:16" json:"kind"` // user | system_resume
	ResumeOf        string         `gorm:"column:resume_of;size:64;index" json:"resumeOf,omitempty"` // system_resume 轮指向的任务 ID
	Status          string         `gorm:"size:16;index" json:"status"`
	ClientRequestID *string        `gorm:"column:client_request_id;size:64;uniqueIndex" json:"-"` // 可空：无幂等键(NULL)不参与去重，PG 唯一索引允许多个 NULL
	ConfigVersion   string         `gorm:"column:config_version;size:32" json:"configVersion"`
	UserMsg         string         `gorm:"column:user_msg" json:"userMsg"`
	Usage           datatypes.JSON `json:"usage"`
	ErrorCode       string         `gorm:"column:error_code;size:32" json:"errorCode,omitempty"`
	ErrorMsg        string         `gorm:"column:error_msg;size:512" json:"errorMsg,omitempty"`
	CreatedAt       int64          `json:"createdAt"`
	UpdatedAt       int64          `json:"updatedAt"`
}

type ChatTurnEvent struct {
	SessionID string         `gorm:"primaryKey;size:64" json:"-"` // 复合主键 (session_id, seq)：seq 仅会话内单调
	Seq       int64          `gorm:"primaryKey" json:"seq"`
	TurnID    string         `gorm:"size:64" json:"turnId"`
	Event     datatypes.JSON `json:"event"`
}

/* ================= 执行内核域（Go 统一建模建表；agentcluster 直连读写，Go 只读供前端轨迹视图） =================
 * 域权限（目标形态独立 PG 角色，MVP 同用户起步）：
 *   agentcluster → chat_* 四表 SELECT；本域四表 SELECT/INSERT/UPDATE；其余表（model_configs 等）不可见。
 */

// ToolCall：工具调用轨迹（agentcluster 执行中直写；工具注册表 §3 调用信封落点）。
type ToolCall struct {
	ID          string         `gorm:"primaryKey;size:64" json:"id"` // tc_
	TurnID      string         `gorm:"size:64;index" json:"turnId"`
	SessionID   string         `gorm:"size:64;index" json:"sessionId"`
	CallID      string         `gorm:"column:call_id;size:64;uniqueIndex" json:"callId"` // 幂等防重
	ToolName    string         `gorm:"column:tool_name;size:128" json:"toolName"`
	ToolVersion string         `gorm:"column:tool_version;size:32" json:"toolVersion"`
	Provider    string         `gorm:"size:32" json:"provider"` // builtin | vendor_agent | mcp | cli
	CallerAgent string         `gorm:"column:caller_agent;size:64" json:"callerAgent"`
	InputJSON   datatypes.JSON `gorm:"column:input_json" json:"input"`
	OutputJSON  datatypes.JSON `gorm:"column:output_json" json:"output"` // 大结果存引用
	RiskLevel   string         `gorm:"column:risk_level;size:8" json:"riskLevel"`
	Status      string         `gorm:"size:16" json:"status"` // success | failed | timeout
	DurationMs  int64          `gorm:"column:duration_ms" json:"durationMs"`
	Error       string         `gorm:"size:512" json:"error"`
	CreatedAt   int64          `json:"createdAt"`
}

// LlmCall：LLM 调用计量（agentcluster 直写；预算治理数据源）。
type LlmCall struct {
	ID               string `gorm:"primaryKey;size:64" json:"id"`
	TurnID           string `gorm:"size:64;index" json:"turnId"`
	SessionID        string `gorm:"size:64;index" json:"sessionId"`
	Model            string `gorm:"size:128" json:"model"`
	PromptTokens     int    `json:"promptTokens"`
	CompletionTokens int    `json:"completionTokens"`
	TotalTokens      int    `json:"totalTokens"`
	LatencyMs        int64  `gorm:"column:latency_ms" json:"latencyMs"`
	Status           string `gorm:"size:16" json:"status"`
	CreatedAt        int64  `json:"createdAt"`
}

// AgentCheckpoint：LangGraph checkpoint 持久化（agentcluster 直写；session_id 即 thread_id）。
type AgentCheckpoint struct {
	ID                 string         `gorm:"primaryKey;size:64" json:"id"`
	SessionID          string         `gorm:"size:64;uniqueIndex:idx_ckpt_session_cid,priority:1" json:"sessionId"`
	CheckpointID       string         `gorm:"column:checkpoint_id;size:64;uniqueIndex:idx_ckpt_session_cid,priority:2" json:"checkpointId"`
	ParentCheckpointID string         `gorm:"column:parent_checkpoint_id;size:64" json:"parentCheckpointId"`
	TurnID             string         `gorm:"column:turn_id;size:64;index" json:"turnId"`
	State              datatypes.JSON `json:"state"`
	CreatedAt          int64          `json:"createdAt"`
}

// AgentContextSummary：滚动摘要（agentcluster 轮末直写；装配上下文时按 session 取 up_to_turn_seq 最大一条）。
type AgentContextSummary struct {
	ID          string `gorm:"primaryKey;size:64" json:"id"`
	SessionID   string `gorm:"size:64;index" json:"sessionId"`
	UpToTurnSeq int    `gorm:"column:up_to_turn_seq;index" json:"upToTurnSeq"`
	Summary     string `json:"summary"`
	Model       string `gorm:"size:128" json:"model"`
	CreatedAt   int64  `json:"createdAt"`
}

/* ================= 管理面（Go 写·agent 只读：动态 subagent 与 workflow 定义） ================= */

// SubagentDef：动态 subagent 定义——sys_prompt + 工具集绑定 + workflow 引用 + 预算。
// 主 agent 运行时按 subagent_id+version 装配实例（docs《Agent集群开发规格》§4）。
type SubagentDef struct {
	ID           string         `gorm:"primaryKey;size:64" json:"id"` // sa_
	SubagentID   string         `gorm:"column:subagent_id;size:64;uniqueIndex:idx_sa_id_ver,priority:1" json:"subagentId"`
	Version      int            `gorm:"uniqueIndex:idx_sa_id_ver,priority:2" json:"version"`
	Status       string         `gorm:"size:16;index" json:"status"` // active | shadow | deprecated
	SysPrompt    string         `gorm:"column:sys_prompt" json:"sysPrompt"`
	Toolset      datatypes.JSON `json:"toolset"`                       // ["metrics-mcp.*", "instance-mcp.sessions", ...]
	WorkflowRef  string         `gorm:"column:workflow_ref;size:64" json:"workflowRef"` // 指向 workflow_defs.workflow_id（可空）
	ModelProfile datatypes.JSON `gorm:"column:model_profile" json:"modelProfile"`
	Budget       datatypes.JSON `json:"budget"`       // 步数/token/时长预算
	OutputCards  datatypes.JSON `json:"outputCards"`  // 输出卡片类型白名单
	RoutingHints datatypes.JSON `json:"routingHints"` // 主 agent 路由提示关键词
	Remark       string         `gorm:"size:256" json:"remark"`
	CreatedAt    int64          `json:"createdAt"`
	UpdatedAt    int64          `json:"updatedAt"`
}

// WorkflowDef：定制化工作流定义。L1_prompt=步骤清单渲染进 sys-prompt（MVP）；
// L2_graph=图 DSL 编译为 LangGraph 强约束执行图（二期）。
type WorkflowDef struct {
	ID         string         `gorm:"primaryKey;size:64" json:"id"` // wf_
	WorkflowID string         `gorm:"column:workflow_id;size:64;uniqueIndex:idx_wf_id_ver,priority:1" json:"workflowId"`
	Version    int            `gorm:"uniqueIndex:idx_wf_id_ver,priority:2" json:"version"`
	Name       string         `gorm:"size:128" json:"name"`
	Level      string         `gorm:"size:16" json:"level"` // L1_prompt | L2_graph
	Status     string         `gorm:"size:16" json:"status"` // active | deprecated
	Definition datatypes.JSON `json:"definition"`
	Remark     string         `gorm:"size:256" json:"remark"`
	CreatedAt  int64          `json:"createdAt"`
	UpdatedAt  int64          `json:"updatedAt"`
}

/* ================= 任务域（表契约：agent 写入/执行/推进度；Go 只读 + 限列写 notified/cancel_requested） ================= */

// AgentTask：异步任务表契约（依赖规则②）。agent 专家 INSERT 提交、worker 认领执行（claimed_by/lease_until）；
// apiserver 轮询本表：running 进度变化 → progress 事件直发会话总线；done 未消费 → 创建 system_resume 轮续跑。
// Go 限列写：cancel_requested（取消级联）、notified（终态消费标记）。
type AgentTask struct {
	ID              string         `gorm:"primaryKey;size:64" json:"id"` // atask_
	SessionID       string         `gorm:"size:64;index" json:"sessionId"`
	TurnID          string         `gorm:"size:64;index" json:"turnId"`
	SubagentID      string         `gorm:"column:subagent_id;size:64" json:"subagentId"`
	CallID          string         `gorm:"column:call_id;size:64;uniqueIndex" json:"callId"` // 幂等防重
	ToolName        string         `gorm:"column:tool_name;size:128" json:"toolName"`
	InputJSON       datatypes.JSON `gorm:"column:input_json" json:"input"`
	ResultRef       string         `gorm:"column:result_ref;size:512" json:"resultRef"` // 大结果引用（agent 侧解析）
	Status          string         `gorm:"size:16;index" json:"status"`                  // pending | running | done | failed | cancelled
	Progress        float64        `json:"progress"`
	Stage           string         `gorm:"size:256" json:"stage"`
	Error           string         `gorm:"size:512" json:"error"`
	CancelRequested bool           `gorm:"column:cancel_requested" json:"cancelRequested"` // Go 限列写
	Notified        bool           `gorm:"notified" json:"-"`                              // Go 限列写：终态已被轮询消费
	ClaimedBy       string         `gorm:"column:claimed_by;size:64" json:"-"`             // agent worker 认领标识
	LeaseUntil      int64          `gorm:"column:lease_until" json:"-"`                     // 认领租约（心跳续期）
	CreatedAt       int64          `json:"createdAt"`
	UpdatedAt       int64          `json:"updatedAt"`
}

/* ================= 模型与插件配置（设置中心 / 插件中心） ================= */

// ModelConfig：大模型（LLM）连接配置。APIKey 不出参，查询时由 handler 填充 APIKeyMask 脱敏展示。
type ModelConfig struct {
	ID          string         `gorm:"primaryKey;size:64" json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	Provider    string         `gorm:"size:64" json:"provider"` // openai 兼容 / anthropic / qwen ...
	BaseURL     string         `gorm:"column:base_url;size:256" json:"baseUrl"`
	APIKey      string         `gorm:"column:api_key;size:512" json:"-"`
	APIKeyMask  string         `gorm:"-" json:"apiKeyMask"`
	Model       string         `gorm:"size:128" json:"model"`
	Params      datatypes.JSON `json:"params"` // 温度 / max_tokens 等自由 JSON
	Remark      string         `gorm:"size:256" json:"remark"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   int64          `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt   int64          `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

// EmbeddingConfig：嵌入模型服务配置，APIKey 处理同 ModelConfig。
type EmbeddingConfig struct {
	ID          string         `gorm:"primaryKey;size:64" json:"id"`
	Name        string         `gorm:"size:128" json:"name"`
	BaseURL     string         `gorm:"column:base_url;size:256" json:"baseUrl"`
	APIKey      string         `gorm:"column:api_key;size:512" json:"-"`
	APIKeyMask  string         `gorm:"-" json:"apiKeyMask"`
	Model       string         `gorm:"size:128" json:"model"`
	Dimension   int            `json:"dimension"` // 0 = 未指定
	Params      datatypes.JSON `json:"params"`
	Remark      string         `gorm:"size:256" json:"remark"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   int64          `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt   int64          `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

// McpServerConfig：MCP 服务插件。Transport 为 stdio 时 Command 是启动命令，Args 是参数；
// 为 http 时 Command 是服务地址，Headers 存请求头。
type McpServerConfig struct {
	ID        string         `gorm:"primaryKey;size:64" json:"id"`
	Name      string         `gorm:"size:128" json:"name"`
	Transport string         `gorm:"size:16" json:"transport"` // stdio | http
	Command   string         `gorm:"size:512" json:"command"`
	Args      datatypes.JSON `json:"args"`    // stdio 启动参数数组 / http 附加参数
	Env       datatypes.JSON `json:"env"`     // stdio 环境变量 / http 自定义请求头
	Remark    string         `gorm:"size:256" json:"remark"`
	Enabled   bool           `json:"enabled"`
	CreatedAt int64          `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt int64          `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

// SkillConfig：Skill 插件（SKILL.md 知识/流程提示词）。
type SkillConfig struct {
	ID          string `gorm:"primaryKey;size:64" json:"id"`
	Name        string `gorm:"size:128" json:"name"`
	Description string `gorm:"size:512" json:"description"`
	Content     string `json:"content"` // SKILL.md 正文
	Enabled     bool   `json:"enabled"`
	CreatedAt   int64  `gorm:"autoCreateTime:false" json:"createdAt"`
	UpdatedAt   int64  `gorm:"autoUpdateTime:false" json:"updatedAt"`
}

/* ================= 审计 ================= */

type AuditLog struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	Ts         int64          `json:"ts"`
	Actor      string         `gorm:"size:64" json:"actor"`
	Action     string         `gorm:"size:64" json:"action"`
	TargetType string         `gorm:"size:32" json:"targetType"`
	TargetID   string         `gorm:"size:64" json:"targetId"`
	Detail     datatypes.JSON `json:"detail"`
}

func AllModels() []interface{} {
	return []interface{}{
		&Cluster{}, &Instance{}, &ClusterParam{}, &ParamHistory{},
		&DbCluster{}, &DbInstance{}, &DbInstanceNode{}, &DbSyncWatermark{},
		&AlertRaw{}, &ChangeTicket{}, &SlowQueryLog{},
		&PgDatabase{}, &PgReplica{}, &ObTenant{}, &ObTenantDb{},
		&Host{}, &InstanceUser{}, &RuntimeSession{}, &Trx{}, &SlowSql{},
		&Report{}, &AlertRecord{}, &MetricDef{}, &MetaStat{},
		&Dashboard{}, &ChatSession{}, &ChatMessage{}, &ChatTurn{}, &ChatTurnEvent{}, &AuditLog{},
		&ModelConfig{}, &EmbeddingConfig{}, &McpServerConfig{}, &SkillConfig{},
		&ToolCall{}, &LlmCall{}, &AgentCheckpoint{}, &AgentContextSummary{},
		&SubagentDef{}, &WorkflowDef{}, &AgentTask{},
	}
}
