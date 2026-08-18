import { Link } from 'react-router-dom';
import { STATUS_MAP, DB_TYPES, EXTRA_TYPE_NAME } from '../lib/mockData';

export function Pill({ st, text }: { st: string; text?: string }) {
  const m = STATUS_MAP[st] || STATUS_MAP.info;
  return <span className={`pill ${m[0]}`}><i></i>{text || m[1]}</span>;
}

export function TypeTag({ t }: { t: string }) {
  const d = DB_TYPES.find(x => x.type === t);
  const name = d ? d.name : (EXTRA_TYPE_NAME[t] || t);
  return <span className={`tag ${t}`}>{name}</span>;
}

export function Bar({ value, hot }: { value: number; hot?: boolean }) {
  const isHot = hot ?? (value > 85);
  return <div className="bar"><i className={isHot ? 'hot' : ''} style={{ width: `${value}%` }}></i></div>;
}

export function Stat({ num, lbl, cls }: { num: any; lbl: string; cls?: string }) {
  return <div className="stat"><div className={`num ${cls || ''}`}>{num}</div><div className="lbl">{lbl}</div></div>;
}

export function LinkA({ to, children }: { to: string; children: React.ReactNode }) {
  return <Link to={to} style={{ color: 'var(--blue)', textDecoration: 'none' }}>{children}</Link>;
}
