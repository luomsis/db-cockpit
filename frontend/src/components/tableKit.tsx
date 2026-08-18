/* ================= 列表页通用小部件：搜索框 / 列筛选 / 分页 / 操作图标 ================= */
import { useEffect, useRef, useState } from 'react';
import type { ReactNode } from 'react';

export function EyeIcon({ size = 15 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

function FunnelIcon({ size = 12 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      <path d="M22 3H2l8 9.46V19l4 2v-8.54L22 3z" />
    </svg>
  );
}

/* 关键字搜索框（带清空按钮） */
export function SearchInput({ value, onChange, placeholder }:
  { value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <div className="tbl-search">
      <span className="ico">⌕</span>
      <input value={value} placeholder={placeholder} onChange={e => onChange(e.target.value)} />
      {!!value && <button className="clear" title="清空" onClick={() => onChange('')}>×</button>}
    </div>
  );
}

/* ---------- 列筛选下拉 ---------- */
export interface FilterOption { value: string; label: string; }

export function uniqOpts(vals: string[]): FilterOption[] {
  return Array.from(new Set(vals)).sort().map(v => ({ value: v, label: v }));
}

export function FilterSelect({ label, value, options, onChange }:
  { label: string; value: string; options: FilterOption[]; onChange: (v: string) => void }) {
  return (
    <label className="filter-item">
      <span className="filter-lbl">{label}</span>
      <select value={value} onChange={e => onChange(e.target.value)}>
        <option value="">全部</option>
        {options.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
      </select>
    </label>
  );
}

/* ---------- 列头过滤：列名旁漏斗图标，点击弹出下拉；选中后图标高亮 ---------- */
export function Th({ children, filter }: {
  children?: ReactNode;
  filter?: { value: string; options: FilterOption[]; onChange: (v: string) => void };
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open]);

  return (
    <th>
      <div className="th-inner" ref={ref}>
        <span className="th-label">{children}</span>
        {filter && (
          <span className="th-filter-slot">
            <button className={`th-filter${filter.value ? ' active' : ''}`} title="筛选"
              onClick={e => { e.stopPropagation(); setOpen(o => !o); }}>
              <FunnelIcon />
            </button>
            {open && (
              <div className="th-filter-pop">
                <button className={`th-filter-opt${!filter.value ? ' cur' : ''}`}
                  onClick={() => { filter.onChange(''); setOpen(false); }}>全部</button>
                {filter.options.map(o => (
                  <button key={o.value} className={`th-filter-opt${filter.value === o.value ? ' cur' : ''}`}
                    onClick={() => { filter.onChange(o.value); setOpen(false); }}>{o.label}</button>
                ))}
              </div>
            )}
          </span>
        )}
      </div>
    </th>
  );
}

/* ---------- 活跃过滤条件 chips（× 删除） ---------- */
export interface ActiveFilter { key: string; label: string; value: string; onRemove: () => void }

export function FilterChips({ items }: { items: ActiveFilter[] }) {
  if (!items.length) return null;
  return (
    <div className="filter-chips">
      {items.map(f => (
        <span className="chip" key={f.key}>
          {f.label}: <b>{f.value}</b>
          <button title="删除该过滤条件" onClick={f.onRemove}>×</button>
        </span>
      ))}
    </div>
  );
}

/* ---------- 分页条：共 N 条 · 每页 x 条 · 页码 ---------- */
export function Pagination({ total, page, pageSize, onPage, onPageSize }:
  { total: number; page: number; pageSize: number; onPage: (p: number) => void; onPageSize: (s: number) => void }) {
  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const nums: (number | '…')[] = [];
  if (pageCount <= 7) {
    for (let i = 1; i <= pageCount; i++) nums.push(i);
  } else {
    nums.push(1);
    if (page > 3) nums.push('…');
    for (let i = Math.max(2, page - 1); i <= Math.min(pageCount - 1, page + 1); i++) nums.push(i);
    if (page < pageCount - 2) nums.push('…');
    nums.push(pageCount);
  }
  const start = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const end = Math.min(page * pageSize, total);
  return (
    <div className="pager">
      <span className="pager-info">共 {total} 条 · 第 {start}-{end} 条</span>
      <span className="pager-size">
        每页
        <select value={pageSize} onChange={e => { onPageSize(Number(e.target.value)); onPage(1); }}>
          {[10, 20, 50].map(n => <option key={n} value={n}>{n}</option>)}
        </select>
        条
      </span>
      <div className="pager-btns">
        <button className="pg" disabled={page <= 1} onClick={() => onPage(page - 1)}>‹</button>
        {nums.map((n, i) => n === '…'
          ? <span key={`e${i}`} className="pg-ell">…</span>
          : <button key={n} className={`pg${n === page ? ' cur' : ''}`} onClick={() => onPage(n)}>{n}</button>)}
        <button className="pg" disabled={page >= pageCount} onClick={() => onPage(page + 1)}>›</button>
      </div>
    </div>
  );
}
