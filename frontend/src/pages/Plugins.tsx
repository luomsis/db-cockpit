import { useEffect, useState } from 'react';
import { apiDelete, apiGet, apiPost, apiPut } from '../lib/api';
import type { McpServerConfig, SkillConfig } from '../lib/types';
import { relTime } from '../lib/dashboards';
import { useBreadcrumb } from '../App';
import { runOp, useOpDialog } from '../components/opDialog';
import type { PromptField } from '../components/opDialog';
import { IconEdit, IconTrash } from '../components/icons';
import { toast } from '../lib/toast';

/* ================= 插件中心：MCP 服务 / Skills ================= */

const ENABLED_OPTS = [
  { value: '1', label: '启用' },
  { value: '0', label: '停用' },
];

function parseOptionalJSON(raw: string): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw);
  } catch {
    throw new Error('不是合法 JSON');
  }
}

/* ---------- MCP 服务 ---------- */

function McpTab() {
  const [list, setList] = useState<McpServerConfig[] | null>(null);
  const [error, setError] = useState('');
  const op = useOpDialog();

  const reload = () => apiGet<McpServerConfig[]>('/api/mcp-servers')
    .then(r => { setList(r); setError(''); })
    .catch(e => { setList([]); setError(e.message); });
  useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  const fields = (m?: McpServerConfig): PromptField[] => [
    { key: 'name', label: '名称', required: true, initial: m?.name, placeholder: '如：db-meta-query' },
    { key: 'transport', label: '类型', type: 'select',
      options: [{ value: 'http', label: 'Streamable HTTP' }, { value: 'stdio', label: 'stdio（本地进程）' }],
      initial: m?.transport || 'http' },
    { key: 'command', label: '命令 / URL', initial: m?.command,
      placeholder: 'http 填服务地址，stdio 填启动命令', required: true },
    { key: 'args', label: '参数 JSON', type: 'textarea', rows: 2,
      initial: m?.args ? JSON.stringify(m.args) : '', placeholder: '可选，stdio 启动参数数组，如 ["--port", "8080"]' },
    { key: 'env', label: '环境变量 / 请求头 JSON', type: 'textarea', rows: 2,
      initial: m?.env ? JSON.stringify(m.env) : '', placeholder: '可选，如 {"API_TOKEN": "xxx"}' },
    { key: 'remark', label: '备注', initial: m?.remark },
    { key: 'enabled', label: '状态', type: 'select', options: ENABLED_OPTS, initial: m ? (m.enabled ? '1' : '0') : '1' },
  ];

  const body = (v: Record<string, string>) => {
    let args: unknown;
    let env: unknown;
    try {
      args = parseOptionalJSON(v.args);
      env = parseOptionalJSON(v.env);
    } catch (e) {
      toast.error((e as Error).message);
      return null;
    }
    return {
      name: v.name, transport: v.transport, command: v.command,
      args, env, remark: v.remark, enabled: v.enabled !== '0',
    };
  };

  const create = () => op.prompt('新建 MCP 服务', fields(), v => {
    const b = body(v);
    if (!b) return;
    runOp('MCP 服务已创建', () => apiPost('/api/mcp-servers', b), reload);
  }, '创建');

  const edit = (m: McpServerConfig) => op.prompt('编辑 MCP 服务', fields(m), v => {
    const b = body(v);
    if (!b) return;
    runOp('MCP 服务已更新', () => apiPut(`/api/mcp-servers/${m.id}`, b), reload);
  }, '保存');

  const del = (m: McpServerConfig) => op.confirm('删除 MCP 服务', `确定删除「${m.name}」吗？该操作不可恢复。`, () => {
    runOp(`已删除「${m.name}」`, () => apiDelete(`/api/mcp-servers/${m.id}`), reload);
  }, { okText: '删除', danger: true });

  return (
    <>
      <div className="cfg-head">
        <span className="cfg-count">共 {list?.length ?? 0} 个 MCP 服务</span>
        <button className="btn sm primary" onClick={create}>+ 新建服务</button>
      </div>
      {error && <div className="cfg-error">apiserver 不可达：{error}</div>}
      <div className="cfg-list">
        {list === null && <div className="cfg-empty">加载中…</div>}
        {list?.map(m => (
          <div className="cfg-item" key={m.id}>
            <div className="cfg-main">
              <div className="cfg-title">
                <b>{m.name}</b>
                <span className="pill info">{m.transport === 'stdio' ? 'stdio' : 'HTTP'}</span>
                <span className={`pill ${m.enabled ? 'ok' : 'err'}`}><i></i>{m.enabled ? '启用' : '停用'}</span>
              </div>
              <div className="cfg-meta">
                <code className="cfg-url">{m.command || '未配置命令/地址'}</code>
                <span className="muted">更新于 {relTime(m.updatedAt)}</span>
              </div>
            </div>
            <div className="cfg-ops">
              <button className="btn sm" onClick={() => edit(m)}><IconEdit size={13} />编辑</button>
              <button className="btn sm danger" onClick={() => del(m)}><IconTrash size={13} />删除</button>
            </div>
          </div>
        ))}
        {list?.length === 0 && !error && <div className="cfg-empty">暂无 MCP 服务，点击右上角「+ 新建服务」注册第一个 MCP 插件。</div>}
      </div>
      {op.view}
    </>
  );
}

/* ---------- Skills ---------- */

function SkillTab() {
  const [list, setList] = useState<SkillConfig[] | null>(null);
  const [error, setError] = useState('');
  const op = useOpDialog();

  const reload = () => apiGet<SkillConfig[]>('/api/skills')
    .then(r => { setList(r); setError(''); })
    .catch(e => { setList([]); setError(e.message); });
  useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  const fields = (s?: SkillConfig): PromptField[] => [
    { key: 'name', label: '名称', required: true, initial: s?.name, placeholder: '如：慢 SQL 诊断流程' },
    { key: 'description', label: '描述', type: 'textarea', rows: 2, initial: s?.description,
      placeholder: '触发时机与用途说明，如：当用户询问慢 SQL 优化时注入此技能' },
    { key: 'content', label: '内容（SKILL.md 正文）', type: 'textarea', rows: 10, initial: s?.content,
      placeholder: 'Markdown 格式的技能指令 / 领域知识 / 执行流程…' },
    { key: 'enabled', label: '状态', type: 'select', options: ENABLED_OPTS, initial: s ? (s.enabled ? '1' : '0') : '1' },
  ];

  const create = () => op.prompt('新建 Skill', fields(), v => {
    runOp('Skill 已创建', () => apiPost('/api/skills',
      { name: v.name, description: v.description, content: v.content, enabled: v.enabled !== '0' }), reload);
  }, '创建');

  const edit = (s: SkillConfig) => op.prompt('编辑 Skill', fields(s), v => {
    runOp('Skill 已更新', () => apiPut(`/api/skills/${s.id}`,
      { name: v.name, description: v.description, content: v.content, enabled: v.enabled !== '0' }), reload);
  }, '保存');

  const del = (s: SkillConfig) => op.confirm('删除 Skill', `确定删除「${s.name}」吗？该操作不可恢复。`, () => {
    runOp(`已删除「${s.name}」`, () => apiDelete(`/api/skills/${s.id}`), reload);
  }, { okText: '删除', danger: true });

  return (
    <>
      <div className="cfg-head">
        <span className="cfg-count">共 {list?.length ?? 0} 个 Skills</span>
        <button className="btn sm primary" onClick={create}>+ 新建 Skill</button>
      </div>
      {error && <div className="cfg-error">apiserver 不可达：{error}</div>}
      <div className="cfg-list">
        {list === null && <div className="cfg-empty">加载中…</div>}
        {list?.map(s => (
          <div className="cfg-item" key={s.id}>
            <div className="cfg-main">
              <div className="cfg-title">
                <b>{s.name}</b>
                <span className={`pill ${s.enabled ? 'ok' : 'err'}`}><i></i>{s.enabled ? '启用' : '停用'}</span>
              </div>
              <div className="cfg-desc">{s.description || <span className="muted">暂无描述</span>}</div>
              <div className="cfg-meta">
                <span className="muted">更新于 {relTime(s.updatedAt)}</span>
              </div>
            </div>
            <div className="cfg-ops">
              <button className="btn sm" onClick={() => edit(s)}><IconEdit size={13} />编辑</button>
              <button className="btn sm danger" onClick={() => del(s)}><IconTrash size={13} />删除</button>
            </div>
          </div>
        ))}
        {list?.length === 0 && !error && <div className="cfg-empty">暂无 Skills，点击右上角「+ 新建 Skill」添加第一个技能。</div>}
      </div>
      {op.view}
    </>
  );
}

/* ---------- 页面 ---------- */

export default function Plugins() {
  useBreadcrumb([{ label: '首页' }, { label: '插件中心' }]);
  const [tab, setTab] = useState<'mcp' | 'skills'>('mcp');
  return (
    <>
      <div className="page-title">插件中心</div>
      <div className="page-desc">管理 MCP 服务与 Skills 插件：MCP 以 stdio / Streamable HTTP 接入外部工具，Skills 以 SKILL.md 形式注入领域知识与流程。</div>
      <div className="tabs">
        <div className={`tab ${tab === 'mcp' ? 'active' : ''}`} onClick={() => setTab('mcp')}>MCP 服务</div>
        <div className={`tab ${tab === 'skills' ? 'active' : ''}`} onClick={() => setTab('skills')}>Skills</div>
      </div>
      {tab === 'mcp' ? <McpTab /> : <SkillTab />}
    </>
  );
}
