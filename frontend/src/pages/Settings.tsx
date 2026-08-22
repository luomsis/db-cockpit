import { useEffect, useState } from 'react';
import { apiDelete, apiGet, apiPost, apiPut } from '../lib/api';
import type { EmbeddingConfig, ModelConfig, TestResult } from '../lib/types';
import { relTime } from '../lib/dashboards';
import { useBreadcrumb } from '../App';
import { runOp, useOpDialog } from '../components/opDialog';
import type { PromptField } from '../components/opDialog';
import { IconBolt, IconEdit, IconTrash } from '../components/icons';
import { toast } from '../lib/toast';

/* ================= 设置中心：模型设置 / 嵌入模型服务 ================= */

const ENABLED_OPTS = [
  { value: '1', label: '启用' },
  { value: '0', label: '停用' },
];

/* 解析用户输入的 JSON（空则 undefined），非法时抛错提示 */
function parseOptionalJSON(raw: string): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw);
  } catch {
    throw new Error('参数不是合法 JSON');
  }
}

/* ---------- 模型设置（大模型配置） ---------- */

function ModelConfigsTab() {
  const [list, setList] = useState<ModelConfig[] | null>(null);
  const [error, setError] = useState('');
  const [testing, setTesting] = useState<string | null>(null);
  const op = useOpDialog();

  const reload = () => apiGet<ModelConfig[]>('/api/model-configs')
    .then(r => { setList(r); setError(''); })
    .catch(e => { setList([]); setError(e.message); });
  useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  const fields = (m?: ModelConfig): PromptField[] => [
    { key: 'name', label: '名称', required: true, initial: m?.name, placeholder: '如：交易库诊断模型' },
    { key: 'provider', label: 'Provider', initial: m?.provider, placeholder: '如：openai / qwen / 自建网关' },
    { key: 'baseUrl', label: 'Base URL', initial: m?.baseUrl, placeholder: 'https://…（OpenAI 兼容地址）' },
    { key: 'model', label: '模型名', required: true, initial: m?.model, placeholder: '如：qwen-max' },
    { key: 'apiKey', label: 'API Key', placeholder: m ? '留空保持不变' : 'sk-…',
      hint: '仅存储在服务端，列表中脱敏展示' },
    { key: 'params', label: '参数 JSON', type: 'textarea', rows: 3,
      initial: m?.params ? JSON.stringify(m.params) : '', placeholder: '可选，如 {"temperature": 0.7, "max_tokens": 2048}' },
    { key: 'remark', label: '备注', initial: m?.remark },
    { key: 'enabled', label: '状态', type: 'select', options: ENABLED_OPTS, initial: m ? (m.enabled ? '1' : '0') : '1' },
  ];

  const body = (v: Record<string, string>) => {
    let params: unknown;
    try {
      params = parseOptionalJSON(v.params);
    } catch (e) {
      toast.error((e as Error).message);
      return null;
    }
    return {
      name: v.name, provider: v.provider, baseUrl: v.baseUrl, model: v.model,
      apiKey: v.apiKey, params, remark: v.remark, enabled: v.enabled !== '0',
    };
  };

  const create = () => op.prompt('新建模型配置', fields(), v => {
    const b = body(v);
    if (!b) return;
    runOp('模型配置已创建', () => apiPost('/api/model-configs', b), reload);
  }, '创建');

  const edit = (m: ModelConfig) => op.prompt('编辑模型配置', fields(m), v => {
    const b = body(v);
    if (!b) return;
    runOp('模型配置已更新', () => apiPut(`/api/model-configs/${m.id}`, b), reload);
  }, '保存');

  const del = (m: ModelConfig) => op.confirm('删除模型配置', `确定删除「${m.name}」吗？该操作不可恢复。`, () => {
    runOp(`已删除「${m.name}」`, () => apiDelete(`/api/model-configs/${m.id}`), reload);
  }, { okText: '删除', danger: true });

  const test = async (m: ModelConfig) => {
    setTesting(m.id);
    try {
      const r = await apiPost<TestResult>(`/api/model-configs/${m.id}/test`);
      if (r.ok) toast.success(`连接成功 · ${r.latencyMs}ms`);
      else toast.error(r.message || '连接失败');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : '测试失败');
    } finally {
      setTesting(null);
    }
  };

  return (
    <>
      <div className="cfg-head">
        <span className="cfg-count">共 {list?.length ?? 0} 个模型配置</span>
        <button className="btn sm primary" onClick={create}>+ 新建配置</button>
      </div>
      {error && <div className="cfg-error">apiserver 不可达：{error}</div>}
      <div className="cfg-list">
        {list === null && <div className="cfg-empty">加载中…</div>}
        {list?.map(m => (
          <div className="cfg-item" key={m.id}>
            <div className="cfg-main">
              <div className="cfg-title">
                <b>{m.name}</b>
                {m.provider && <span className="pill info">{m.provider}</span>}
                <span className={`pill ${m.enabled ? 'ok' : 'err'}`}><i></i>{m.enabled ? '启用' : '停用'}</span>
              </div>
              <div className="cfg-meta">
                <span className="cfg-model">{m.model || '未指定模型'}</span>
                {m.baseUrl && <code className="cfg-url">{m.baseUrl}</code>}
                <span className="muted">Key {m.apiKeyMask || '未配置'}</span>
                <span className="muted">更新于 {relTime(m.updatedAt)}</span>
              </div>
            </div>
            <div className="cfg-ops">
              <button className="btn sm" disabled={testing === m.id} onClick={() => test(m)}>
                <IconBolt size={13} />{testing === m.id ? '测试中…' : '测试'}
              </button>
              <button className="btn sm" onClick={() => edit(m)}><IconEdit size={13} />编辑</button>
              <button className="btn sm danger" onClick={() => del(m)}><IconTrash size={13} />删除</button>
            </div>
          </div>
        ))}
        {list?.length === 0 && !error && <div className="cfg-empty">暂无模型配置，点击右上角「+ 新建配置」添加第一个大模型。</div>}
      </div>
      {op.view}
    </>
  );
}

/* ---------- 嵌入模型服务 ---------- */

function EmbeddingConfigsTab() {
  const [list, setList] = useState<EmbeddingConfig[] | null>(null);
  const [error, setError] = useState('');
  const [testing, setTesting] = useState<string | null>(null);
  const op = useOpDialog();

  const reload = () => apiGet<EmbeddingConfig[]>('/api/embedding-configs')
    .then(r => { setList(r); setError(''); })
    .catch(e => { setList([]); setError(e.message); });
  useEffect(() => { reload(); /* eslint-disable-next-line */ }, []);

  const fields = (e?: EmbeddingConfig): PromptField[] => [
    { key: 'name', label: '名称', required: true, initial: e?.name, placeholder: '如：知识库向量服务' },
    { key: 'baseUrl', label: 'Base URL', initial: e?.baseUrl, placeholder: 'https://…（OpenAI 兼容地址）' },
    { key: 'model', label: '模型名', required: true, initial: e?.model, placeholder: '如：text-embedding-v3' },
    { key: 'dimension', label: '向量维度', initial: e?.dimension ? String(e.dimension) : '', placeholder: '可选，如 1024', hint: '留空可由「测试连接」自动探测' },
    { key: 'apiKey', label: 'API Key', placeholder: e ? '留空保持不变' : 'sk-…',
      hint: '仅存储在服务端，列表中脱敏展示' },
    { key: 'remark', label: '备注', initial: e?.remark },
    { key: 'enabled', label: '状态', type: 'select', options: ENABLED_OPTS, initial: e ? (e.enabled ? '1' : '0') : '1' },
  ];

  const body = (v: Record<string, string>) => ({
    name: v.name, baseUrl: v.baseUrl, model: v.model,
    dimension: v.dimension ? Number(v.dimension) || 0 : 0,
    apiKey: v.apiKey, remark: v.remark, enabled: v.enabled !== '0',
  });

  const create = () => op.prompt('新建嵌入服务配置', fields(), v => {
    runOp('嵌入服务配置已创建', () => apiPost('/api/embedding-configs', body(v)), reload);
  }, '创建');

  const edit = (e: EmbeddingConfig) => op.prompt('编辑嵌入服务配置', fields(e), v => {
    runOp('嵌入服务配置已更新', () => apiPut(`/api/embedding-configs/${e.id}`, body(v)), reload);
  }, '保存');

  const del = (e: EmbeddingConfig) => op.confirm('删除嵌入服务配置', `确定删除「${e.name}」吗？该操作不可恢复。`, () => {
    runOp(`已删除「${e.name}」`, () => apiDelete(`/api/embedding-configs/${e.id}`), reload);
  }, { okText: '删除', danger: true });

  const test = async (e: EmbeddingConfig) => {
    setTesting(e.id);
    try {
      const r = await apiPost<TestResult>(`/api/embedding-configs/${e.id}/test`);
      if (r.ok) toast.success(`连接成功 · ${r.latencyMs}ms${r.dimension ? ` · 维度 ${r.dimension}` : ''}`);
      else toast.error(r.message || '连接失败');
    } catch (err) {
      toast.error(err instanceof Error ? err.message : '测试失败');
    } finally {
      setTesting(null);
    }
  };

  return (
    <>
      <div className="cfg-head">
        <span className="cfg-count">共 {list?.length ?? 0} 个嵌入服务</span>
        <button className="btn sm primary" onClick={create}>+ 新建配置</button>
      </div>
      {error && <div className="cfg-error">apiserver 不可达：{error}</div>}
      <div className="cfg-list">
        {list === null && <div className="cfg-empty">加载中…</div>}
        {list?.map(e => (
          <div className="cfg-item" key={e.id}>
            <div className="cfg-main">
              <div className="cfg-title">
                <b>{e.name}</b>
                <span className={`pill ${e.enabled ? 'ok' : 'err'}`}><i></i>{e.enabled ? '启用' : '停用'}</span>
              </div>
              <div className="cfg-meta">
                <span className="cfg-model">{e.model || '未指定模型'}</span>
                {e.baseUrl && <code className="cfg-url">{e.baseUrl}</code>}
                {e.dimension > 0 && <span className="pill info">维度 {e.dimension}</span>}
                <span className="muted">Key {e.apiKeyMask || '未配置'}</span>
                <span className="muted">更新于 {relTime(e.updatedAt)}</span>
              </div>
            </div>
            <div className="cfg-ops">
              <button className="btn sm" disabled={testing === e.id} onClick={() => test(e)}>
                <IconBolt size={13} />{testing === e.id ? '测试中…' : '测试'}
              </button>
              <button className="btn sm" onClick={() => edit(e)}><IconEdit size={13} />编辑</button>
              <button className="btn sm danger" onClick={() => del(e)}><IconTrash size={13} />删除</button>
            </div>
          </div>
        ))}
        {list?.length === 0 && !error && <div className="cfg-empty">暂无嵌入服务配置，点击右上角「+ 新建配置」添加。</div>}
      </div>
      {op.view}
    </>
  );
}

/* ---------- 页面 ---------- */

export default function Settings() {
  useBreadcrumb([{ label: '首页' }, { label: '设置' }]);
  const [tab, setTab] = useState<'models' | 'embeddings'>('models');
  return (
    <>
      <div className="page-title">设置</div>
      <div className="page-desc">管理大模型与嵌入模型服务配置。API Key 仅存储于服务端并在列表中脱敏展示；「测试」会用存储的密钥真实调用远端服务验证连通性。</div>
      <div className="tabs">
        <div className={`tab ${tab === 'models' ? 'active' : ''}`} onClick={() => setTab('models')}>LLM设置</div>
        <div className={`tab ${tab === 'embeddings' ? 'active' : ''}`} onClick={() => setTab('embeddings')}>Embedding设置</div>
      </div>
      {tab === 'models' ? <ModelConfigsTab /> : <EmbeddingConfigsTab />}
    </>
  );
}
