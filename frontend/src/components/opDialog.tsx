/* ================= 页面写操作统一弹窗与反馈 =================
 * useOpDialog()：prompt(多字段输入) / confirm(危险确认) 的声明式弹窗，
 * 配合 runOp() 完成调用 → toast 反馈 → 刷新 的标准链路，
 * 替代原生的 window.prompt / window.confirm。
 */
import { useRef, useState } from 'react';
import { ConfirmDialog, Overlay } from './dialogs';
import { toast } from '../lib/toast';

export interface PromptField {
  key: string;
  label: string;
  placeholder?: string;
  initial?: string;
  required?: boolean;
  hint?: string;
}

export function PromptDialog({ title, fields, okText, onOk, onClose }: {
  title: string;
  fields: PromptField[];
  okText?: string;
  onOk: (values: Record<string, string>) => void;
  onClose: () => void;
}) {
  const refs = useRef<Record<string, HTMLInputElement | null>>({});
  const submit = () => {
    const values: Record<string, string> = {};
    for (const f of fields) {
      const v = (refs.current[f.key]?.value || '').trim();
      if (f.required && !v) { refs.current[f.key]?.focus(); return; }
      values[f.key] = v;
    }
    onClose();
    onOk(values);
  };
  return (
    <Overlay onClose={onClose}>
      <div className="dap-head">{title}</div>
      <div className="dap-body">
        {fields.map(f => (
          <label key={f.key}>{f.label}
            <input
              ref={el => { refs.current[f.key] = el; }}
              defaultValue={f.initial || ''}
              placeholder={f.placeholder || ''}
              onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); submit(); } }}
            />
            {f.hint && <span className="opd-hint">{f.hint}</span>}
          </label>
        ))}
      </div>
      <div className="dap-foot">
        <button className="btn sm" onClick={onClose}>取消</button>
        <button className="btn sm primary" onClick={submit}>{okText || '确定'}</button>
      </div>
    </Overlay>
  );
}

type OpDialogState =
  | {
      type: 'prompt';
      title: string;
      fields: PromptField[];
      okText?: string;
      onOk: (values: Record<string, string>) => void;
    }
  | {
      type: 'confirm';
      title: string;
      message: string;
      okText?: string;
      danger?: boolean;
      onOk: () => void;
    };

export function useOpDialog() {
  const [state, setState] = useState<OpDialogState | null>(null);
  const api = {
    prompt: (title: string, fields: PromptField[], onOk: (v: Record<string, string>) => void, okText?: string) =>
      setState({ type: 'prompt', title, fields, onOk, okText }),
    confirm: (title: string, message: string, onOk: () => void, opts?: { okText?: string; danger?: boolean }) =>
      setState({ type: 'confirm', title, message, onOk, okText: opts?.okText, danger: opts?.danger }),
    close: () => setState(null),
  };
  const view = !state ? null : state.type === 'prompt'
    ? <PromptDialog title={state.title} fields={state.fields} okText={state.okText} onOk={state.onOk} onClose={api.close} />
    : <ConfirmDialog title={state.title} message={state.message} okText={state.okText} danger={state.danger}
        onOk={state.onOk} onClose={api.close} />;
  return { ...api, view };
}

/* 标准执行链路：成功 toast + after（通常 reload），失败红 toast */
export async function runOp(okMsg: string, fn: () => Promise<unknown>, after?: () => void) {
  try {
    await fn();
    toast.success(okMsg);
    after?.();
  } catch (e) {
    toast.error(e instanceof Error ? e.message : '操作失败');
  }
}
