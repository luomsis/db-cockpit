/* =====================================================================
 * 数据库智能运维平台 · 项目汇报 PPT 生成器（pptxgenjs）
 * 产出：../数据库智能运维平台-项目汇报.pptx（18 页，16:9 宽屏）
 * 风格：平台同源蓝白科技风（深色封面/尾页 + 浅色内容页三明治）
 * ===================================================================== */
const pptxgen = require('pptxgenjs');
const React = require('react');
const ReactDOMServer = require('react-dom/server');
const Fi = require('react-icons/fi');
const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

/* ---------- 调色板（与平台 UI 同源） ---------- */
const C = {
  navy: '1D2B45', primary: '0052D9', blue: '006AFF', ice: 'CADCFC',
  light: 'E8F2FF', lighter: 'F4F8FF', line: 'D8E4F5',
  accent: 'FF9500', amberBg: 'FFF4E5',
  text: '1D2B45', text2: '4E5D78', muted: '8A97AD',
  white: 'FFFFFF', green: '00B365', greenBg: 'E6F7EF', red: 'F53F3F', redBg: 'FFECE8',
};
const FONT = 'Microsoft YaHei';
const W = 13.333, H = 7.5;

/* ---------- 图标（react-icons → 白色 PNG dataURL） ---------- */
const ICON_DEFS = {
  target: Fi.FiTarget, layers: Fi.FiLayers, cpu: Fi.FiCpu, users: Fi.FiUsers,
  alert: Fi.FiAlertTriangle, db: Fi.FiDatabase, chat: Fi.FiMessageSquare,
  chart: Fi.FiBarChart2, shield: Fi.FiShield, check: Fi.FiCheckCircle,
  calendar: Fi.FiCalendar, risk: Fi.FiAlertCircle, tool: Fi.FiTool,
  layout: Fi.FiLayout, zap: Fi.FiZap, link: Fi.FiLink, shuffle: Fi.FiShuffle,
  monitor: Fi.FiMonitor, rocket: Fi.FiNavigation, box: Fi.FiBox, clock: Fi.FiClock,
  file: Fi.FiFileText, refresh: Fi.FiRefreshCw, git: Fi.FiGitBranch,
  eye: Fi.FiEye, lock: Fi.FiLock, send: Fi.FiSend, flag: Fi.FiFlag,
  key: Fi.FiKey, list: Fi.FiList, activity: Fi.FiActivity, search: Fi.FiSearch,
};
const ICONS = {}; // key -> dataURL

async function buildIcons() {
  for (const [key, Comp] of Object.entries(ICON_DEFS)) {
    let svg = ReactDOMServer.renderToStaticMarkup(React.createElement(Comp, { size: 256 }));
    svg = svg.replace(/currentColor/g, '#FFFFFF');
    const png = await sharp(Buffer.from(svg), { density: 300 }).resize(256, 256).png().toBuffer();
    ICONS[key] = 'image/png;base64,' + png.toString('base64');
  }
}

/* ---------- 通用绘制助手 ---------- */
let pageNo = 0;
function newSlide(pres, dark = false) {
  const s = pres.addSlide();
  pageNo += 1;
  s.background = { color: dark ? C.navy : C.white };
  if (!dark) {
    s.addText(`${String(pageNo).padStart(2, '0')}`, { x: W - 0.75, y: H - 0.42, w: 0.45, h: 0.3, fontFace: FONT, fontSize: 9, color: C.muted, align: 'right', margin: 0 });
    s.addText('DB Copilot · 项目汇报', { x: 0.55, y: H - 0.42, w: 3, h: 0.3, fontFace: FONT, fontSize: 9, color: C.muted, margin: 0 });
  }
  return s;
}
function header(s, tag, title, sub) {
  s.addText(tag, { x: 0.55, y: 0.34, w: 11, h: 0.32, fontFace: FONT, fontSize: 11, bold: true, color: C.blue, charSpacing: 2, margin: 0 });
  s.addText(title, { x: 0.55, y: 0.64, w: 12.2, h: 0.55, fontFace: FONT, fontSize: 25, bold: true, color: C.text, margin: 0 });
  if (sub) s.addText(sub, { x: 0.55, y: 1.2, w: 12.2, h: 0.32, fontFace: FONT, fontSize: 11, color: C.muted, margin: 0 });
}
function card(s, { x, y, w, h, fill = C.white, line = C.line, radius = 0.07, shadow = true }) {
  const opt = { x, y, w, h, fill: { color: fill }, rectRadius: radius, line: line ? { color: line, width: 0.75 } : { type: 'none' } };
  if (shadow) opt.shadow = { type: 'outer', color: '1D2B45', opacity: 0.07, blur: 5, offset: 2, angle: 90, rotateWithShape: false };
  s.addShape('roundRect', opt);
}
function chip(s, { x, y, w, h = 0.34, text, fill = C.light, color = C.primary, size = 9.5, bold = false, line = null }) {
  s.addShape('roundRect', { x, y, w, h, fill: { color: fill }, rectRadius: h / 2, line: line ? { color: line, width: 0.75 } : { type: 'none' } });
  s.addText(text, { x, y: y - 0.015, w, h, fontFace: FONT, fontSize: size, bold, color, align: 'center', valign: 'middle', margin: 0 });
}
function iconCircle(s, key, x, y, d = 0.44, bg = C.primary) {
  s.addShape('ellipse', { x, y, w: d, h: d, fill: { color: bg }, line: { type: 'none' } });
  const pad = d * 0.24;
  s.addImage({ data: ICONS[key], x: x + pad, y: y + pad, w: d - pad * 2, h: d - pad * 2 });
}
function bullets(s, { x, y, w, h, items, size = 12, color = C.text2, gap = 7, boldColor = null }) {
  const arr = items.map((it, i) => {
    const [head, desc] = Array.isArray(it) ? it : [null, it];
    if (head) {
      return { text: `${head}　`, options: { bold: true, color: boldColor || C.text, bullet: { code: '25AA', indent: 10 }, breakLine: false } };
    }
    return { text: desc, options: { color, bullet: { code: '25AA', indent: 10 }, breakLine: true, paraSpaceAfter: i === items.length - 1 ? 0 : gap } };
  });
  s.addText(arr, { x, y, w, h, fontFace: FONT, fontSize: size, valign: 'top', margin: 0, lineSpacingMultiple: 1.12 });
}
function arrow(s, x1, y1, x2, y2, { color = C.muted, width = 1.2, dash = 'solid', end = 'triangle' } = {}) {
  s.addShape('line', { x: x1, y: y1, w: x2 - x1, h: y2 - y1, line: { color, width, dashType: dash, endArrowType: end, beginArrowType: 'none' } });
}

/* ===================================================================== */
async function main() {
  await buildIcons();
  const pres = new pptxgen();
  pres.layout = 'LAYOUT_WIDE';
  pres.author = 'DB Copilot';
  pres.title = '数据库智能运维平台 · 项目汇报';

  /* ============ S1 封面（深色） ============ */
  {
    const s = newSlide(pres, true);
    // 背景装饰：右上大圆 + 层叠架构暗示
    s.addShape('ellipse', { x: 9.2, y: -2.2, w: 7, h: 7, fill: { color: C.primary, transparency: 82 }, line: { type: 'none' } });
    s.addShape('ellipse', { x: 10.6, y: 4.9, w: 5.2, h: 5.2, fill: { color: C.blue, transparency: 86 }, line: { type: 'none' } });
    // 右侧层叠卡片（架构栈暗示）
    const stack = [
      { y: 2.05, c: C.blue, t: '006AFF' }, { y: 2.85, c: C.primary, t: '0052D9' }, { y: 3.65, c: C.navy, t: '0E1A30' },
    ];
    stack.forEach((it, i) => {
      s.addShape('roundRect', { x: 8.85 + i * 0.28, y: it.y, w: 2.9, h: 0.62, rectRadius: 0.08, fill: { color: it.t }, line: { color: '3A5480', width: 0.75 } });
    });
    s.addText('PRESENTATION · LLM-NATIVE DB OPS', { x: 8.85, y: 1.62, w: 3.4, h: 0.3, fontFace: 'Arial', fontSize: 9, color: C.ice, charSpacing: 2, margin: 0 });
    // 品牌区
    s.addShape('roundRect', { x: 0.85, y: 0.7, w: 0.52, h: 0.52, rectRadius: 0.1, fill: { color: C.blue }, line: { type: 'none' } });
    s.addText('DB', { x: 0.85, y: 0.7, w: 0.52, h: 0.52, fontFace: 'Arial', fontSize: 13, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
    s.addText('DB Copilot', { x: 1.52, y: 0.7, w: 3, h: 0.3, fontFace: FONT, fontSize: 14, bold: true, color: C.white, margin: 0 });
    s.addText('数据库智能运维平台', { x: 1.52, y: 1.0, w: 3, h: 0.25, fontFace: FONT, fontSize: 9.5, color: C.ice, margin: 0 });
    // 主标题
    s.addText('数据库智能运维平台', { x: 0.85, y: 2.6, w: 11.6, h: 0.95, fontFace: FONT, fontSize: 44, bold: true, color: C.white, margin: 0 });
    s.addText('DB Copilot · 项目规划 / 架构设计 / 实施进展 / 任务分派', { x: 0.85, y: 3.62, w: 11.6, h: 0.42, fontFace: FONT, fontSize: 16, color: C.ice, margin: 0 });
    s.addText([
      { text: '监控大盘 · 智能诊断 · 多 Agent 对话 · 统一数据底座', options: { color: '9FB6DF', breakLine: false } },
    ], { x: 0.85, y: 4.18, w: 11.6, h: 0.34, fontFace: FONT, fontSize: 11.5, margin: 0 });
    // 底部
    s.addText('数据智能运维团队　·　2026 年 8 月', { x: 0.85, y: 6.55, w: 6, h: 0.3, fontFace: FONT, fontSize: 11, color: C.muted, margin: 0 });
    s.addText('MVP 汇报版 v1.0', { x: 9.9, y: 6.55, w: 2.6, h: 0.3, fontFace: FONT, fontSize: 11, color: C.muted, align: 'right', margin: 0 });
  }

  /* ============ S2 汇报框架 ============ */
  {
    const s = newSlide(pres);
    header(s, 'AGENDA', '汇报框架');
    const items = [
      { icon: 'target', n: '01', t: '目标与范围', d: '最终形态与建设目标\nMVP 边界与北极星场景' },
      { icon: 'layers', n: '02', t: '总体架构', d: '分层架构与技术选型\n核心流程时序与数据流' },
      { icon: 'cpu', n: '03', t: '关键模块', d: 'Agent 执行框架 / 异步任务闭环\n工具注册表 / 卡片协议 / 前端进展' },
      { icon: 'users', n: '04', t: '分工与验收', d: '三个任务系列与排期\n契约优先验收机制与风险' },
    ];
    items.forEach((it, i) => {
      const x = 0.55 + i * 3.13;
      card(s, { x, y: 2.0, w: 2.93, h: 3.9 });
      iconCircle(s, it.icon, x + 0.28, y0(2.35), 0.56, i === 3 ? C.accent : C.primary);
      s.addText(it.n, { x: x + 2.0, y: 2.32, w: 0.75, h: 0.55, fontFace: 'Arial', fontSize: 26, bold: true, color: C.light, align: 'right', margin: 0 });
      s.addText(it.t, { x: x + 0.28, y: 3.25, w: 2.4, h: 0.45, fontFace: FONT, fontSize: 17, bold: true, color: C.text, margin: 0 });
      s.addText(it.d, { x: x + 0.28, y: 3.78, w: 2.45, h: 1.7, fontFace: FONT, fontSize: 11, color: C.text2, margin: 0, lineSpacingMultiple: 1.3 });
    });
    function y0(v) { return v; }
    s.addText('本页为全局导航：四部分依次对应「为什么做 → 怎么设计 → 关键如何实现 → 谁来做如何验收」', { x: 0.55, y: 6.35, w: 12.2, h: 0.3, fontFace: FONT, fontSize: 10.5, color: C.muted, margin: 0 });
  }

  /* ============ S3 背景与痛点 ============ */
  {
    const s = newSlide(pres);
    header(s, '01 · 目标与范围', '背景与痛点：存量割裂，能力待统一收口');
    const pains = [
      { icon: 'alert', c: C.red, bg: C.redBg, t: '存量系统各自割裂', d: '告警系统、工单系统、DBaaS 平台独立建设，入口分散、体验不一，排障需在多系统间来回切换' },
      { icon: 'monitor', c: C.accent, bg: C.amberBg, t: '外采诊断能力受限', d: '已外采诊断 Agent 覆盖大部分国产数据库，API 可编程；但自带界面不满足需求，难以融入统一平台' },
      { icon: 'shuffle', c: C.primary, bg: C.light, t: '数据体系与 API 风格不一', d: '指标、元数据、告警的数据模型与接口风格各异，新能力建设反复适配、重复投入' },
    ];
    pains.forEach((p, i) => {
      const y = 1.72 + i * 1.62;
      card(s, { x: 0.55, y, w: 7.15, h: 1.46 });
      iconCircle(s, p.icon, 0.85, y + 0.42, 0.6, p.bg);
      s.addShape('ellipse', { x: 0.85, y: y + 0.42, w: 0.6, h: 0.6, fill: { color: p.c }, line: { type: 'none' } });
      s.addText(p.t, { x: 1.7, y: y + 0.22, w: 5.8, h: 0.4, fontFace: FONT, fontSize: 14.5, bold: true, color: C.text, margin: 0 });
      s.addText(p.d, { x: 1.7, y: y + 0.62, w: 5.75, h: 0.75, fontFace: FONT, fontSize: 10.5, color: C.text2, margin: 0, lineSpacingMultiple: 1.2 });
    });
    // 右侧：建设契机（渐进替换）
    card(s, { x: 8.0, y: 1.72, w: 4.78, h: 4.86, fill: C.lighter });
    s.addText('建设契机 · 渐进替换策略', { x: 8.3, y: 2.0, w: 4.2, h: 0.4, fontFace: FONT, fontSize: 14, bold: true, color: C.text, margin: 0 });
    s.addText('Strangler Fig：新建统一平台，旧系统经防腐层集成、按模块分批吸收，最终按节奏下线', { x: 8.3, y: 2.42, w: 4.2, h: 0.6, fontFace: FONT, fontSize: 10.5, color: C.text2, margin: 0, lineSpacingMultiple: 1.25 });
    const legacy = ['告警系统', 'DBaaS', '工单系统'];
    legacy.forEach((t, i) => {
      s.addShape('roundRect', { x: 8.35, y: 3.25 + i * 0.62, w: 1.45, h: 0.46, rectRadius: 0.06, fill: { color: C.white }, line: { color: C.line, width: 0.75 } });
      s.addText(t, { x: 8.35, y: 3.25 + i * 0.62, w: 1.45, h: 0.46, fontFace: FONT, fontSize: 10, color: C.text2, align: 'center', valign: 'middle', margin: 0 });
    });
    arrow(s, 9.95, 4.2, 10.55, 4.2, { color: C.primary, width: 1.6 });
    s.addShape('roundRect', { x: 10.65, y: 3.55, w: 1.85, h: 1.3, rectRadius: 0.08, fill: { color: C.primary }, line: { type: 'none' } });
    s.addText('DB Copilot\n统一平台', { x: 10.65, y: 3.55, w: 1.85, h: 1.3, fontFace: FONT, fontSize: 12, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0, lineSpacingMultiple: 1.2 });
    s.addText('防腐层收口：旧系统变更只波及适配层', { x: 8.35, y: 5.35, w: 4.2, h: 0.55, fontFace: FONT, fontSize: 10, color: C.primary, margin: 0, lineSpacingMultiple: 1.2 });
  }

  /* ============ S4 建设目标与最终形态 ============ */
  {
    const s = newSlide(pres);
    header(s, '01 · 目标与范围', '建设目标与最终形态：双范式智能运维平台');
    // 左：5 大目标
    s.addText('五大建设目标', { x: 0.55, y: 1.62, w: 5.5, h: 0.35, fontFace: FONT, fontSize: 14, bold: true, color: C.text, margin: 0 });
    const goals = [
      ['综合型智能运维平台', '覆盖多种数据库类型的统一入口'],
      ['保留并重构传统能力', '概览 / 集群 / 监控大盘 / 实例详情'],
      ['重点建设下钻分析', '概览→集群→实例→会话/慢SQL/锁'],
      ['对话式智能运维', 'LLM 多 Agent：诊断 / 问数 / 知识'],
      ['兼容旧体系渐进替换', '适配旧数据体系与 API，支持分批吸收'],
    ];
    goals.forEach((g, i) => {
      const y = 2.05 + i * 0.92;
      s.addShape('ellipse', { x: 0.58, y: y + 0.05, w: 0.4, h: 0.4, fill: { color: C.light }, line: { type: 'none' } });
      s.addText(String(i + 1), { x: 0.58, y: y + 0.05, w: 0.4, h: 0.4, fontFace: 'Arial', fontSize: 13, bold: true, color: C.primary, align: 'center', valign: 'middle', margin: 0 });
      s.addText(g[0], { x: 1.16, y, w: 4.6, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
      s.addText(g[1], { x: 1.16, y: y + 0.32, w: 4.6, h: 0.3, fontFace: FONT, fontSize: 10, color: C.muted, margin: 0 });
    });
    // 右：双范式
    card(s, { x: 6.1, y: 1.62, w: 6.68, h: 2.6 });
    s.addText('双范式并列 · 双向流转', { x: 6.4, y: 1.84, w: 5, h: 0.35, fontFace: FONT, fontSize: 14, bold: true, color: C.text, margin: 0 });
    const paradigms = [
      { x: 6.4, t: '浏览式', icon: 'chart', items: '监控大盘（Grafana 式编辑器）\n集群 / 实例详情 · 下钻导航' },
      { x: 9.62, t: '对话式', icon: 'chat', items: 'Copilot 智能对话\nAI 输出结构化卡片渲染' },
    ];
    paradigms.forEach(p => {
      s.addShape('roundRect', { x: p.x, y: 2.3, w: 2.95, h: 1.62, rectRadius: 0.08, fill: { color: C.lighter }, line: { color: C.line, width: 0.75 } });
      iconCircle(s, p.icon, p.x + 0.22, 2.52, 0.46);
      s.addText(p.t, { x: p.x + 0.82, y: 2.56, w: 1.9, h: 0.38, fontFace: FONT, fontSize: 13.5, bold: true, color: C.text, margin: 0 });
      s.addText(p.items, { x: p.x + 0.24, y: 3.08, w: 2.6, h: 0.75, fontFace: FONT, fontSize: 10, color: C.text2, margin: 0, lineSpacingMultiple: 1.25 });
    });
    arrow(s, 9.4, 3.1, 9.6, 3.1, { color: C.muted, width: 1.4, end: 'triangle' });
    arrow(s, 9.6, 3.42, 9.4, 3.42, { color: C.muted, width: 1.4, end: 'triangle' });
    // 愿景功能 chips
    card(s, { x: 6.1, y: 4.42, w: 6.68, h: 2.16, fill: C.lighter });
    s.addText('愿景能力（二期+ 设计占位已入架构）', { x: 6.4, y: 4.62, w: 5.5, h: 0.32, fontFace: FONT, fontSize: 12, bold: true, color: C.text, margin: 0 });
    const chips = [
      ['选中页面元素 · 直接智能诊断', 3.16], ['平台说明书知识库 · 回答附跳转 URL', 3.36],
      ['自治服务中心 · 任务+建议+手动触发', 3.36], ['⌘K 全局自然语言导航', 2.6],
    ];
    let cx = 6.4, cy = 5.05;
    chips.forEach(([t, w]) => {
      if (cx + w > 12.55) { cx = 6.4; cy += 0.52; }
      chip(s, { x: cx, y: cy, w, h: 0.4, text: t, fill: C.light, color: C.primary, size: 10 });
      cx += w + 0.18;
    });
    chip(s, { x: 6.4, y: 6.12, w: 6.08, h: 0.4, text: '一期北极星功能：数据库实例智能诊断', fill: C.amberBg, color: 'D46B08', size: 10.5, bold: true });
  }

  /* ============ S5 MVP 范围 ============ */
  {
    const s = newSlide(pres);
    header(s, '01 · 目标与范围', 'MVP 范围：3 个月 · 3 人 · 真实数据接入');
    card(s, { x: 0.55, y: 1.6, w: 12.23, h: 0.78, fill: C.light });
    iconCircle(s, 'rocket', 0.82, 1.77, 0.44, C.primary);
    s.addText([
      { text: '一句话目标：', options: { bold: true, color: C.text, breakLine: false } },
      { text: '以真实数据跑通五项架构核心假设 —— 统一查询协议 · 多 Agent 编排 · 卡片协议 · 异步任务闭环 · 防腐层集成', options: { color: C.text2, breakLine: false } },
    ], { x: 1.42, y: 1.6, w: 11.2, h: 0.78, fontFace: FONT, fontSize: 12.5, valign: 'middle', margin: 0 });
    // 做 / 不做
    card(s, { x: 0.55, y: 2.62, w: 6.0, h: 3.06 });
    iconCircle(s, 'check', 0.85, 2.86, 0.42, C.green);
    s.addText('MVP 交付（做）', { x: 1.42, y: 2.9, w: 4, h: 0.35, fontFace: FONT, fontSize: 13.5, bold: true, color: C.text, margin: 0 });
    bullets(s, {
      x: 0.9, y: 3.42, w: 5.5, h: 2.2, size: 11, gap: 6, items: [
        '监控大盘：完整编辑器复刻（多大盘 / 双 Y 轴 / 阈值 / 事件标注）',
        '智能对话：多会话 + 流式 + 5 种结构化卡片',
        'Agent 诊断：LangGraph ReAct + 8 个自建工具',
        'Agent 问数：四类白名单 NL2Metric',
        '真实接入：公司 AI 平台 / 旧监控 / DBaaS / 告警',
        '外采诊断：适配器规范就绪，契约到手即接入',
      ],
    });
    card(s, { x: 6.78, y: 2.62, w: 6.0, h: 3.06 });
    iconCircle(s, 'lock', 7.08, 2.86, 0.42, C.muted);
    s.addText('明确不做（防范围蔓延）', { x: 7.65, y: 2.9, w: 4.5, h: 0.35, fontFace: FONT, fontSize: 13.5, bold: true, color: C.text, margin: 0 });
    bullets(s, {
      x: 7.13, y: 3.42, w: 5.5, h: 2.2, size: 11, gap: 6, items: [
        '鉴权 / SSO（权限桩，二期接 RBAC）',
        '页面上下文注入 · 元素选中诊断（二期）',
        '自治服务页 · 知识库问答（二期）',
        'L1 / L2 动作执行（Agent 只读提议）',
        'Redis / 消息队列 / 对象存储（PG 单库）',
        'MCP / CLI / Skills 插件生态（二期+）',
      ],
    });
    // 北极星场景 flow
    s.addText('北极星验收场景', { x: 0.55, y: 5.92, w: 2.2, h: 0.3, fontFace: FONT, fontSize: 11.5, bold: true, color: C.text, margin: 0 });
    const steps = ['实例变慢', '大盘发现异常', 'chat 发起诊断', '真实证据采集', '异步深扫任务', '报告卡 + 追问'];
    let sx = 2.75;
    steps.forEach((t, i) => {
      const w = 1.42;
      chip(s, { x: sx, y: 5.86, w, h: 0.4, text: t, fill: i === 5 ? C.amberBg : C.light, color: i === 5 ? 'D46B08' : C.primary, size: 9.5, bold: i === 5 });
      if (i < 5) arrow(s, sx + w + 0.02, 6.06, sx + w + 0.24, 6.06, { color: C.muted, width: 1.1 });
      sx += w + 0.28;
    });
  }

  /* ============ S6 总体架构图 ============ */
  {
    const s = newSlide(pres);
    header(s, '02 · 总体架构', '分层架构：逻辑分层 × 物理归属（React / Go / Python / PG）');
    const bands = [
      { y: 1.62, h: 0.66, t: '前端层', badge: 'React', items: ['概览 / 集群 / 主机', '监控大盘（完整编辑器）', '智能对话（多会话）', '卡片渲染器注册表'] },
      { y: 2.38, h: 0.54, t: '接入层', badge: 'Go', items: ['API 网关', 'SSO（二期）', '权限网关（MVP 桩）'] },
      { y: 3.02, h: 0.84, t: '应用服务层', badge: 'Go', items: ['大盘 / 元数据 / Issue 服务', '会话与档案服务', '任务总线调度（轮询 / 对账 / wake）', '内部数据 API（供 Agent 工具）'], strong: true },
      { y: 3.96, h: 0.84, t: 'AI 层', badge: 'Python', items: ['路由 Agent（意图分发）', '诊断 / 问数专家', '工具注册表', '卡片生成器'], strong: true, ai: true },
      { y: 4.9, h: 0.66, t: '数据服务层', badge: 'Go', items: ['指标查询代理（缓存 / 白名单）', 'Probe Executor（三通道路由）', '实例接入网关（只读 / 审计 / 熔断）'] },
      { y: 5.66, h: 0.54, t: '存储层', badge: 'PG', items: ['PostgreSQL 单库：元数据 · Issue · 会话轨迹 · 任务 · 缓存'] },
      { y: 6.3, h: 0.62, t: '防腐层', badge: 'Go', items: ['DBaaS 适配器', '告警适配器', '监控 API 适配器', '外采诊断适配器（契约后接入）'] },
    ];
    bands.forEach(b => {
      s.addShape('roundRect', {
        x: 0.55, y: b.y, w: 10.6, h: b.h, rectRadius: 0.06,
        fill: { color: b.ai ? C.light : C.lighter },
        line: { color: b.strong ? C.blue : C.line, width: b.strong ? 1.2 : 0.75 },
      });
      s.addText(b.t, { x: 0.75, y: b.y, w: 1.35, h: b.h, fontFace: FONT, fontSize: 11.5, bold: true, color: C.text, valign: 'middle', margin: 0 });
      // 模块 chips
      let cx = 2.2;
      b.items.forEach(it => {
        const w = 0.28 + it.length * 0.108;
        s.addShape('roundRect', { x: cx, y: b.y + (b.h - 0.36) / 2, w, h: 0.36, rectRadius: 0.05, fill: { color: C.white }, line: { color: C.line, width: 0.75 } });
        s.addText(it, { x: cx, y: b.y + (b.h - 0.36) / 2 - 0.01, w, h: 0.36, fontFace: FONT, fontSize: 9, color: C.text2, align: 'center', valign: 'middle', margin: 0 });
        cx += w + 0.14;
      });
      // 物理归属 badge
      s.addShape('roundRect', { x: 10.06, y: b.y + (b.h - 0.32) / 2, w: 0.92, h: 0.32, rectRadius: 0.05, fill: { color: b.badge === 'React' ? '0E1A30' : b.badge === 'Python' ? '0F5132' : b.badge === 'PG' ? '6E3A07' : C.blue }, line: { type: 'none' } });
      s.addText(b.badge, { x: 10.06, y: b.y + (b.h - 0.32) / 2 - 0.01, w: 0.92, h: 0.32, fontFace: 'Arial', fontSize: 9, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
    });
    // 右侧：存量系统 + AI 平台
    s.addShape('roundRect', { x: 11.35, y: 6.3, w: 1.45, h: 0.62, rectRadius: 0.06, fill: { color: C.white }, line: { color: C.line, width: 1 } });
    s.addText('存量系统', { x: 11.35, y: 6.36, w: 1.45, h: 0.3, fontFace: FONT, fontSize: 9.5, bold: true, color: C.text2, align: 'center', margin: 0 });
    arrow(s, 11.15, 6.61, 11.35, 6.61, { color: C.muted, width: 1.2 });
    s.addShape('roundRect', { x: 11.35, y: 4.06, w: 1.45, h: 0.64, rectRadius: 0.06, fill: { color: C.primary }, line: { type: 'none' } });
    s.addText('公司统一\nAI 平台', { x: 11.35, y: 4.06, w: 1.45, h: 0.64, fontFace: FONT, fontSize: 9, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0, lineSpacingMultiple: 1.1 });
    arrow(s, 11.35, 4.38, 11.15, 4.38, { color: C.primary, width: 1.4, dash: 'dash' });
    s.addText('OpenAI 兼容 · tool calling · 流式（Python 直连）', { x: 11.28, y: 4.78, w: 1.6, h: 0.6, fontFace: FONT, fontSize: 7.5, color: C.muted, margin: 0, lineSpacingMultiple: 1.15 });
  }

  /* ============ S7 技术选型与部署 ============ */
  {
    const s = newSlide(pres);
    header(s, '02 · 总体架构', '技术选型与部署形态：Compose 四容器起步');
    // 左：选型表
    const rows = [
      [{ text: '层', options: { bold: true, color: C.white, fill: { color: C.primary } } }, { text: '选型', options: { bold: true, color: C.white, fill: { color: C.primary } } }, { text: '要点', options: { bold: true, color: C.white, fill: { color: C.primary } } }],
      ['前端', 'React 18 + TS + Vite', '卡片渲染器 = 组件注册，ECharts'],
      ['apiserver', 'Go', '收口入口与存储，SSE 透传'],
      ['Agent 集群', 'Python + LangGraph', '四端口封装，无状态多副本'],
      ['LLM', '公司统一 AI 平台', 'OpenAI 兼容 · tool calling · 流式'],
      ['存储', 'PostgreSQL 单库', 'JSONB 承载卡片 / 轨迹；不引入中间件'],
      ['部署', 'Docker Compose', '生产 K8s 后置'],
    ].map(r => r.map(cell => typeof cell === 'string' ? { text: cell, options: { color: C.text2 } } : cell));
    s.addTable(rows, {
      x: 0.55, y: 1.7, w: 5.9, colW: [1.1, 2.15, 2.65], fontFace: FONT, fontSize: 10,
      border: { type: 'solid', color: C.line, pt: 0.75 }, rowH: 0.42, valign: 'middle', margin: 0.06, fill: { color: C.white },
    });
    // 右：部署图
    card(s, { x: 6.85, y: 1.62, w: 5.93, h: 3.6, fill: C.lighter });
    s.addText('物理部署与调用关系', { x: 7.1, y: 1.8, w: 4, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
    const bx = (x, y, w, h, t, fill, tc, fs = 10) => {
      s.addShape('roundRect', { x, y, w, h, rectRadius: 0.07, fill: { color: fill }, line: fill === C.white ? { color: C.line, width: 0.75 } : { type: 'none' } });
      s.addText(t, { x, y: y - 0.01, w, h, fontFace: FONT, fontSize: fs, bold: true, color: tc, align: 'center', valign: 'middle', margin: 0, lineSpacingMultiple: 1.05 });
    };
    bx(7.15, 2.25, 1.5, 0.55, '用户', C.white, C.text2);
    bx(9.1, 2.25, 1.9, 0.55, 'frontend\n(React·Nginx)', C.white, C.text2, 9);
    bx(11.15, 2.25, 1.4, 0.55, 'PG', '6E3A07', C.white);
    bx(9.1, 3.3, 1.9, 0.75, 'apiserver (Go)\nREST·SSE·内部API·调度', C.blue, C.white, 9);
    bx(11.15, 3.35, 1.4, 0.65, 'agentcluster\n(Python)', '0F5132', C.white, 8.5);
    bx(7.15, 4.35, 1.9, 0.55, '存量系统\n监控/DBaaS/告警', C.white, C.text2, 8.5);
    bx(9.6, 4.35, 1.4, 0.55, '外采诊断\n(契约后)', C.white, C.muted, 8.5);
    bx(11.15, 4.35, 1.4, 0.55, 'AI 平台', C.primary, C.white, 9.5);
    arrow(s, 8.65, 2.52, 9.1, 2.52, { color: C.muted, width: 1.2 });
    arrow(s, 10.05, 2.8, 10.05, 3.3, { color: C.muted, width: 1.2 });
    arrow(s, 11.0, 3.6, 11.15, 3.6, { color: C.muted, width: 1 }); // go->pg? adjust below
    arrow(s, 9.1, 3.86, 8.4, 4.35, { color: C.muted, width: 1 });
    arrow(s, 10.6, 4.05, 9.95, 4.35, { color: C.muted, width: 1, dash: 'dash' });
    arrow(s, 11.5, 4.0, 11.5, 4.35, { color: C.primary, width: 1.4, dash: 'dash' });
    // go<->agent 双向
    arrow(s, 11.0, 3.55, 11.15, 3.55, { color: C.green, width: 1.4 });
    arrow(s, 11.15, 3.8, 11.0, 3.8, { color: C.green, width: 1.4 });
    s.addText('SSE 透传 / 内部数据 API / wake 回调', { x: 10.55, y: 2.98, w: 2.3, h: 0.3, fontFace: FONT, fontSize: 7.5, color: C.green, margin: 0 });
    // 底部三原则
    const principles = [
      ['入口收口', '前端一切请求（含 chat SSE）只打 apiserver'],
      ['存储收口', '全部表归 Go 持有，Python 无状态'],
      ['防腐收口', '全部旧系统调用在 Go 适配层'],
    ];
    principles.forEach((p, i) => {
      const x = 0.55 + i * 4.18;
      card(s, { x, y: 5.55, w: 3.98, h: 1.15 });
      iconCircle(s, 'shield', x + 0.22, 5.78, 0.42, C.primary);
      s.addText(p[0], { x: x + 0.78, y: 5.72, w: 3, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
      s.addText(p[1], { x: x + 0.78, y: 6.05, w: 3.05, h: 0.55, fontFace: FONT, fontSize: 9.5, color: C.text2, margin: 0, lineSpacingMultiple: 1.15 });
    });
    s.addText('Go ↔ Python 边界三原则（硬约束）', { x: 6.85, y: 5.28, w: 5.9, h: 0.3, fontFace: FONT, fontSize: 11, bold: true, color: C.text, align: 'center', margin: 0 });
  }

  /* ============ S8 核心流程时序图 ============ */
  {
    const s = newSlide(pres);
    header(s, '02 · 总体架构', '核心流程：智能诊断全链路时序');
    const lanes = [
      { t: '用户(DBA)', x: 1.15 }, { t: '前端 React', x: 2.85 }, { t: 'Go apiserver', x: 4.55 },
      { t: 'Agent 专家', x: 6.25 }, { t: '内部数据 API', x: 7.95 }, { t: '任务总线', x: 9.65 }, { t: '外采诊断·二期', x: 11.35 },
    ];
    const top = 2.05, bottom = 6.75;
    lanes.forEach(l => {
      const w = 1.5;
      s.addShape('roundRect', { x: l.x - w / 2, y: 1.5, w, h: 0.44, rectRadius: 0.06, fill: { color: l.t.includes('二期') ? C.white : C.blue }, line: { color: C.line, width: 0.75 } });
      s.addText(l.t, { x: l.x - w / 2, y: 1.49, w, h: 0.44, fontFace: FONT, fontSize: 9.5, bold: true, color: l.t.includes('二期') ? C.muted : C.white, align: 'center', valign: 'middle', margin: 0 });
      s.addShape('line', { x: l.x, y: top, w: 0, h: bottom - top, line: { color: C.line, width: 1, dashType: 'dash' } });
    });
    const msg = (from, to, y, label, opt = {}) => {
      const x1 = lanes[from].x, x2 = lanes[to].x;
      arrow(s, x1, y, x2, y, { color: opt.color || C.text2, width: 1.2, dash: opt.dash ? 'dash' : 'solid' });
      s.addText(label, { x: Math.min(x1, x2) - 0.15, y: y - 0.27, w: Math.abs(x2 - x1) + 0.3, h: 0.24, fontFace: FONT, fontSize: 8.5, color: opt.color || C.text2, align: 'center', margin: 0 });
      s.addText(String(opt.n || ''), { x: Math.min(x1, x2) - 0.42, y: y - 0.1, w: 0.3, h: 0.22, fontFace: 'Arial', fontSize: 8, bold: true, color: C.primary, align: 'right', margin: 0 });
    };
    msg(0, 1, 2.45, '「实例 X 变慢，帮我诊断」', { n: 1 });
    msg(1, 2, 2.85, '发起会话轮次', { n: 2 });
    msg(2, 3, 3.25, 'SSE 透传（绑定会话）', { n: 3 });
    msg(3, 4, 3.68, '同步工具：get_metrics / session_snapshot', { n: 4 });
    msg(4, 3, 4.08, '标准化结果（Schema 校验）', { n: 5, dash: true, color: C.muted });
    msg(3, 5, 4.5, '提交深度诊断任务（异步 · 轮次终止）', { n: 6, color: C.accent });
    msg(3, 1, 4.92, '流式回复「已启动」+ 进度卡（经 Go 透传）', { n: 7, color: C.blue });
    msg(5, 6, 5.32, 'submit / poll（契约到手后接入）', { n: 8, dash: true, color: C.muted });
    msg(5, 3, 5.75, 'wake 唤醒：任务结果注入会话', { n: 9, color: C.accent });
    msg(3, 1, 6.15, '诊断报告卡（二次推理 · 交叉验证）', { n: 10, color: C.blue });
    msg(1, 0, 6.55, '卡片渲染 · 支持追问', { n: 11 });
    s.addText('任务完成 ≠ 直接弹结果：唤醒专家做「二次推理」，将任务结果与会话既有证据交叉验证后出结论', { x: 0.55, y: 6.95, w: 12.2, h: 0.3, fontFace: FONT, fontSize: 10, color: C.primary, margin: 0, align: 'center' });
  }

  /* ============ S9 数据流 ============ */
  {
    const s = newSlide(pres);
    header(s, '02 · 总体架构', '数据架构：三类通道 · 双数据流演进');
    const chans = [
      { t: 'a · 本地沉淀通道', c: C.primary, bg: C.light, items: ['元数据 / 告警 / Issue', '定时拉取 + 同步水位（幂等）', '告警统一 Issue 化（平台内流转）'] },
      { t: 'b · 实时直连通道', c: C.green, bg: C.greenBg, items: ['会话 / 锁 / 慢SQL / 执行计划', '实例接入网关：只读账号', '操作审计 · 超时熔断（不缓存）'] },
      { t: 'c · 旧系统 API 通道', c: C.accent, bg: C.amberBg, items: ['时序指标经查询代理（带缓存）', '限流 / 排队 / 降级返最近缓存值', '二期被自建采集替换，上层无感'] },
    ];
    chans.forEach((ch, i) => {
      const x = 0.55 + i * 4.18;
      card(s, { x, y: 1.66, w: 3.98, h: 2.5 });
      s.addShape('roundRect', { x: x + 0.24, y: 1.9, w: 2.1, h: 0.42, rectRadius: 0.06, fill: { color: ch.bg }, line: { type: 'none' } });
      s.addText(ch.t, { x: x + 0.24, y: 1.89, w: 2.1, h: 0.42, fontFace: FONT, fontSize: 11.5, bold: true, color: ch.c, align: 'center', valign: 'middle', margin: 0 });
      bullets(s, { x: x + 0.28, y: 2.5, w: 3.5, h: 1.5, size: 10.5, gap: 7, items: ch.items });
    });
    // 下：诊断结论沉淀 + 指标白名单
    card(s, { x: 0.55, y: 4.42, w: 6.0, h: 2.3 });
    iconCircle(s, 'db', 0.85, 4.66, 0.44, C.primary);
    s.addText('诊断档案 = 一等公民资产', { x: 1.45, y: 4.7, w: 4.5, h: 0.35, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    bullets(s, {
      x: 0.9, y: 5.22, w: 5.4, h: 1.4, size: 10.5, gap: 6, items: [
        '会话 / 轮次 / 工具轨迹 / 报告全量落库',
        '任一会话可完整回放（故障复盘 + 提示词回归）',
        '大结果存 JSONB 引用，控制库体积',
      ],
    });
    card(s, { x: 6.78, y: 4.42, w: 6.0, h: 2.3 });
    iconCircle(s, 'list', 7.08, 4.66, 0.44, C.primary);
    s.addText('指标归一化 · 最小白名单策略（MVP）', { x: 7.68, y: 4.7, w: 5, h: 0.35, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    bullets(s, {
      x: 7.13, y: 5.22, w: 5.4, h: 1.4, size: 10.5, gap: 6, items: [
        '只归一大盘与诊断所需 20~30 个核心指标',
        '映射表落库：映射是数据不是代码，新增 = 插一行',
        '未命中明确报错并给出注册指引，不猜测映射',
      ],
    });
  }

  /* ============ S10 模块① Agent 执行框架 ============ */
  {
    const s = newSlide(pres);
    header(s, '03 · 关键模块 ①', 'Agent 执行框架：四端口 + ReAct 循环（LangGraph 落地）');
    // 左：四端口图
    const cx = 3.15, cy = 4.25;
    s.addShape('roundRect', { x: cx - 1.25, y: cy - 0.62, w: 2.5, h: 1.24, rectRadius: 0.1, fill: { color: C.primary }, line: { type: 'none' }, shadow: { type: 'outer', color: '0052D9', opacity: 0.25, blur: 8, offset: 3, angle: 90 } });
    s.addText('AgentLoop\nReAct 执行循环', { x: cx - 1.25, y: cy - 0.62, w: 2.5, h: 1.24, fontFace: FONT, fontSize: 14, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0, lineSpacingMultiple: 1.15 });
    const ports = [
      { t: 'LLM 端口', d: '公司 AI 平台\nOpenAI 兼容·流式', x: cx - 1.05, y: 1.95 },
      { t: '工具端口', d: '统一工具注册表\nSchema 校验·审计', x: cx + 1.35, y: 2.9 },
      { t: '上下文端口', d: '预算制装配\ncheckpoint 持久化', x: cx - 1.05, y: 5.75 },
      { t: '任务端口', d: '异步任务提交\nwake 续聊', x: cx + 1.35, y: 4.8 },
    ];
    ports.forEach(p => {
      s.addShape('roundRect', { x: p.x, y: p.y, w: 1.85, h: 0.92, rectRadius: 0.08, fill: { color: C.white }, line: { color: C.blue, width: 1.2 } });
      s.addText(p.t, { x: p.x, y: p.y + 0.1, w: 1.85, h: 0.3, fontFace: FONT, fontSize: 11, bold: true, color: C.primary, align: 'center', margin: 0 });
      s.addText(p.d, { x: p.x, y: p.y + 0.4, w: 1.85, h: 0.48, fontFace: FONT, fontSize: 8.5, color: C.text2, align: 'center', margin: 0, lineSpacingMultiple: 1.1 });
      const px = p.x < cx ? p.x + 1.85 : p.x, py = p.y + 0.46;
      const tx = p.x < cx ? cx - 1.25 : cx + 1.25, ty = cy + (p.y < cy ? -0.2 : 0.2);
      arrow(s, px, py, tx, ty, { color: C.blue, width: 1.1 });
    });
    s.addText('专家 = 声明式配置（AgentDefinition），基础设施全部挂接端口之下可替换', { x: 0.55, y: 6.85, w: 5.6, h: 0.3, fontFace: FONT, fontSize: 9.5, color: C.muted, margin: 0 });
    // 右：要点 + LangGraph 映射
    bullets(s, {
      x: 6.6, y: 1.75, w: 6.2, h: 1.5, size: 11.5, gap: 7, items: [
        ['一套运行时，所有专家复用', '新专家 = 新配置，不改框架代码'],
        ['AgentDefinition 版本化', '提示词 / 模型 / 工具策略 / 预算，支持灰度 shadow 双跑'],
        ['有界执行', '步数 / token / 时长 / 费用预算 + 同工具同参循环熔断'],
      ],
    });
    // LangGraph 映射表
    s.addText('LangGraph 落地映射（Python 承载）', { x: 6.6, y: 3.42, w: 6, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
    const map = [
      ['AgentLoop 循环', 'StateGraph：agent→tools→agent'],
      ['流式输出', 'astream_events → SSE 六类事件'],
      ['上下文端口', '自定义 Checkpointer（经 Go API 落 PG）'],
      ['任务续聊', '异步任务 END + wake 回调续跑'],
      ['预算护栏', 'recursion_limit + guard 节点'],
    ];
    map.forEach((r, i) => {
      const y = 3.85 + i * 0.56;
      s.addShape('roundRect', { x: 6.6, y, w: 2.6, h: 0.46, rectRadius: 0.06, fill: { color: C.light }, line: { type: 'none' } });
      s.addText(r[0], { x: 6.72, y: y - 0.01, w: 2.4, h: 0.46, fontFace: FONT, fontSize: 10, bold: true, color: C.primary, valign: 'middle', margin: 0 });
      arrow(s, 9.25, y + 0.23, 9.55, y + 0.23, { color: C.muted, width: 1 });
      s.addText(r[1], { x: 9.65, y: y - 0.01, w: 3.15, h: 0.46, fontFace: FONT, fontSize: 10, color: C.text2, valign: 'middle', margin: 0 });
    });
  }

  /* ============ S11 模块② 异步任务续聊闭环 ============ */
  {
    const s = newSlide(pres);
    header(s, '03 · 关键模块 ②', '异步任务与续聊闭环：DB 为唯一事实源');
    // 左：状态机
    s.addText('DIAG_TASK 状态机', { x: 0.55, y: 1.66, w: 4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    const st = { pending: [1.35, 2.7], running: [3.55, 2.7], done: [5.75, 2.25], failed: [5.75, 3.25], cancelled: [3.55, 4.35] };
    const stBox = (k, x, y, color) => {
      s.addShape('roundRect', { x, y, w: 1.15, h: 0.5, rectRadius: 0.25, fill: { color }, line: { type: 'none' } });
      s.addText(k, { x, y: y - 0.01, w: 1.15, h: 0.5, fontFace: 'Arial', fontSize: 10.5, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
    };
    stBox('pending', st.pending[0], st.pending[1], C.muted);
    stBox('running', st.running[0], st.running[1], C.blue);
    stBox('done', st.done[0], st.done[1], C.green);
    stBox('failed', st.failed[0], st.failed[1], C.red);
    stBox('cancelled', st.cancelled[0], st.cancelled[1], C.muted);
    arrow(s, 2.5, 2.95, 3.55, 2.95, { color: C.text2, width: 1.2 });
    arrow(s, 4.7, 2.85, 5.75, 2.5, { color: C.green, width: 1.2 });
    arrow(s, 4.7, 3.1, 5.75, 3.4, { color: C.red, width: 1.2 });
    arrow(s, 5.75, 3.65, 4.85, 3.7, { color: C.red, width: 1, dash: 'dash' });
    s.addText('重试(限次)', { x: 4.62, y: 3.72, w: 1.2, h: 0.24, fontFace: FONT, fontSize: 8, color: C.red, margin: 0 });
    arrow(s, 3.0, 3.2, 3.55, 4.35, { color: C.muted, width: 1, dash: 'dash' });
    s.addText('进度更新不改状态\n（progress / stage）', { x: 1.2, y: 2.1, w: 2.6, h: 0.5, fontFace: FONT, fontSize: 8.5, color: C.muted, margin: 0 });
    card(s, { x: 0.55, y: 5.05, w: 6.4, h: 1.75, fill: C.lighter });
    bullets(s, {
      x: 0.85, y: 5.3, w: 5.9, h: 1.3, size: 10.5, gap: 6, items: [
        '每次状态迁移写 task_event 流水（前态/后态/时间戳）',
        '内存调度仅加速器：重启后 pending 重调度、running 对账改判',
        'call_id 全局唯一 + 工具限流：防 Agent 循环重复提交',
      ],
    });
    // 右：闭环五步
    s.addText('续聊闭环（分钟级任务不阻塞对话）', { x: 7.35, y: 1.66, w: 5.4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    const steps2 = [
      ['专家提交异步任务', '本轮结束：回复「已启动」+ 进度卡', C.blue],
      ['进度流式更新', 'progress/stage 定向更新进度卡（幂等合并）', C.blue],
      ['完成 wake 回调', 'Go 调度器 → agentcluster 唤醒绑定专家', C.accent],
      ['系统轮次注入', '任务结果作为输入创建 system turn', C.accent],
      ['二次推理出报告', '与既有证据交叉验证 → 诊断报告卡', C.green],
    ];
    steps2.forEach((it, i) => {
      const y = 2.12 + i * 0.94;
      s.addShape('ellipse', { x: 7.35, y: y + 0.06, w: 0.44, h: 0.44, fill: { color: it[2] }, line: { type: 'none' } });
      s.addText(String(i + 1), { x: 7.35, y: y + 0.06, w: 0.44, h: 0.44, fontFace: 'Arial', fontSize: 13, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
      if (i < 4) s.addShape('line', { x: 7.57, y: y + 0.52, w: 0, h: 0.44, line: { color: C.line, width: 1.5 } });
      s.addText(it[0], { x: 7.98, y, w: 4.8, h: 0.3, fontFace: FONT, fontSize: 12, bold: true, color: C.text, margin: 0 });
      s.addText(it[1], { x: 7.98, y: y + 0.31, w: 4.85, h: 0.3, fontFace: FONT, fontSize: 9.5, color: C.text2, margin: 0 });
    });
  }

  /* ============ S12 模块③ 统一工具注册表 ============ */
  {
    const s = newSlide(pres);
    header(s, '03 · 关键模块 ③', '统一工具注册表：能力插件化，注册即接入');
    // 左上：Schema 要点
    card(s, { x: 0.55, y: 1.62, w: 5.3, h: 2.62 });
    s.addText('ToolDefinition 关键字段', { x: 0.85, y: 1.82, w: 4.5, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
    const fields = [
      ['description', '给 LLM 看的说明书：何时用 / 不适用'],
      ['input/output_schema', 'JSON Schema 双向强校验'],
      ['execution_mode', 'sync / async 一等公民'],
      ['risk_level', 'L0 只读自主 · L1/L2 需确认审批'],
      ['db_types + routing', '按实例库类型路由候选工具'],
      ['rate_limit', '防 Agent 循环打挂下游'],
    ];
    fields.forEach((f, i) => {
      const y = 2.24 + i * 0.33;
      s.addText(f[0], { x: 0.88, y, w: 1.9, h: 0.3, fontFace: 'Courier New', fontSize: 9, bold: true, color: C.primary, margin: 0 });
      s.addText(f[1], { x: 2.85, y, w: 2.95, h: 0.3, fontFace: FONT, fontSize: 9, color: C.text2, margin: 0 });
    });
    // 左下：vendor 适配器
    card(s, { x: 0.55, y: 4.44, w: 5.3, h: 2.28 });
    s.addText('外采诊断接入：三段式适配器', { x: 0.85, y: 4.64, w: 4.6, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
    const seg = [['submit', '提交 + ID 映射'], ['poll', '轮询进度/状态'], ['normalize', '报告归一化']];
    seg.forEach((g, i) => {
      const x = 0.85 + i * 1.62;
      s.addShape('roundRect', { x, y: 5.1, w: 1.45, h: 0.72, rectRadius: 0.08, fill: { color: C.light }, line: { type: 'none' } });
      s.addText(g[0], { x, y: 5.18, w: 1.45, h: 0.28, fontFace: 'Courier New', fontSize: 10, bold: true, color: C.primary, align: 'center', margin: 0 });
      s.addText(g[1], { x, y: 5.46, w: 1.45, h: 0.26, fontFace: FONT, fontSize: 8, color: C.text2, align: 'center', margin: 0 });
      if (i < 2) arrow(s, x + 1.46, 5.46, x + 1.62, 5.46, { color: C.muted, width: 1.1 });
    });
    s.addText('契约未到已落地规范与 shadow 注册位——契约到手仅实现一个 Go 适配器即接入，注册表 / 专家 / 前端零改动', { x: 0.85, y: 5.98, w: 4.75, h: 0.65, fontFace: FONT, fontSize: 9.5, color: C.text2, margin: 0, lineSpacingMultiple: 1.2 });
    // 右：MVP 8 工具
    card(s, { x: 6.1, y: 1.62, w: 6.68, h: 5.1 });
    s.addText('MVP 自建工具清单（全部 L0 只读）', { x: 6.4, y: 1.84, w: 5.5, h: 0.32, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
    const tools = [
      ['builtin_get_metrics', 'metrics', '问数/诊断', '指标查询代理'],
      ['builtin_metric_anomaly', 'metrics', '诊断', '阈值/环比检测'],
      ['builtin_list_alerts', 'alert', '问数/诊断', '本地 Issue 记录'],
      ['builtin_list_instances', 'topology', '问数', 'DBaaS 同步元数据'],
      ['builtin_top_slow_queries', 'slow_sql', '问数/诊断', '慢日志 TOP'],
      ['builtin_session_snapshot', 'session', '诊断', '网关只读直连'],
      ['builtin_lock_analysis', 'lock', '诊断', '锁等待链'],
      ['builtin_metric_deep_scan', 'metrics·异步', '诊断', '验证异步闭环'],
    ];
    const trows = [[
      { text: '工具', options: { bold: true, color: C.white, fill: { color: C.primary } } },
      { text: '类别', options: { bold: true, color: C.white, fill: { color: C.primary } } },
      { text: '消费方', options: { bold: true, color: C.white, fill: { color: C.primary } } },
      { text: '数据来源', options: { bold: true, color: C.white, fill: { color: C.primary } } },
    ]].concat(tools.map(t => t.map((cell, j) => ({ text: cell, options: { fontFace: j === 0 ? 'Courier New' : FONT, fontSize: j === 0 ? 8 : 9, color: j === 0 ? C.primary : C.text2 } }))));
    s.addTable(trows, {
      x: 6.4, y: 2.3, w: 6.1, colW: [2.15, 1.05, 1.15, 1.75], fontFace: FONT, fontSize: 9,
      border: { type: 'solid', color: C.line, pt: 0.75 }, rowH: 0.4, valign: 'middle', margin: 0.05,
    });
    s.addText('问数专家仅绑定 metrics / alert / slow_sql / topology 四类（NL2Metric 白名单）', { x: 6.4, y: 6.15, w: 6.1, h: 0.3, fontFace: FONT, fontSize: 9.5, color: C.muted, margin: 0 });
  }

  /* ============ S13 模块④ 卡片协议 ============ */
  {
    const s = newSlide(pres);
    header(s, '03 · 关键模块 ④', 'Generative UI 卡片协议：AI 输出即界面');
    // 左：信封
    s.addText('统一信封（Envelope）', { x: 0.55, y: 1.62, w: 4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    card(s, { x: 0.55, y: 2.05, w: 5.15, h: 3.3, fill: C.lighter });
    const env = [
      ['card_id', '全局唯一 · 流式幂等合并'],
      ['card_type', '类型枚举（只增不改）'],
      ['status', 'streaming → final（定稿只读）'],
      ['source', '溯源：会话/轮次/Agent/工具'],
      ['context', '运维上下文：实例 + 时间窗'],
      ['payload', '类型特化数据（Schema 定义）'],
      ['interactions', 'ask / drilldown / open_link…'],
      ['fallback_text', '降级 / 复制 / 纯文本场景'],
    ];
    env.forEach((e, i) => {
      const y = 2.28 + i * 0.36;
      s.addText(e[0], { x: 0.85, y, w: 1.65, h: 0.3, fontFace: 'Courier New', fontSize: 9.5, bold: true, color: C.primary, margin: 0 });
      s.addText(e[1], { x: 2.6, y, w: 3.0, h: 0.3, fontFace: FONT, fontSize: 9, color: C.text2, margin: 0 });
    });
    card(s, { x: 0.55, y: 5.55, w: 5.15, h: 1.2, fill: C.amberBg, line: 'FFD9A0' });
    s.addText([
      { text: '向前兼容：', options: { bold: true, color: 'D46B08', breakLine: false } },
      { text: '未知 card_type 走 FallbackRenderer（标题 + 摘要），绝不白屏；版本不兼容降级为 fallback_text', options: { color: '8A5A00', breakLine: false } },
    ], { x: 0.85, y: 5.55, w: 4.6, h: 1.2, fontFace: FONT, fontSize: 10, valign: 'middle', margin: 0, lineSpacingMultiple: 1.25 });
    // 右：5 卡片 grid
    s.addText('MVP 五种卡片', { x: 6.1, y: 1.62, w: 4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    const cards5 = [
      ['text', '富文本回复', 'Markdown 容器'],
      ['metric_chart', '图表卡', '时序 + 类目分布'],
      ['data_table', '数据表卡', '行级「问 AI」追问'],
      ['task_progress', '进度卡', '异步任务步骤条'],
      ['diagnosis_report', '诊断报告卡', '根因置信度/证据链'],
    ];
    cards5.forEach((cd, i) => {
      const x = 6.1 + (i % 2) * 3.4, y = 2.05 + Math.floor(i / 2) * 1.14;
      card(s, { x, y, w: 3.2, h: 0.98 });
      iconCircle(s, i === 0 ? 'file' : i === 1 ? 'chart' : i === 2 ? 'list' : i === 3 ? 'clock' : 'activity', x + 0.2, y + 0.26, 0.46, i === 4 ? C.accent : C.blue);
      s.addText(cd[1], { x: x + 0.8, y: y + 0.18, w: 2.3, h: 0.3, fontFace: FONT, fontSize: 12, bold: true, color: C.text, margin: 0 });
      s.addText(cd[0] + ' · ' + cd[2], { x: x + 0.8, y: y + 0.5, w: 2.35, h: 0.3, fontFace: FONT, fontSize: 8.5, color: C.muted, margin: 0 });
    });
    card(s, { x: 6.1, y: 5.55, w: 6.68, h: 1.2, fill: C.light });
    iconCircle(s, 'layout', 6.38, 5.83, 0.46, C.primary);
    s.addText([
      { text: '渲染器注册表（前端）：', options: { bold: true, color: C.text, breakLine: false } },
      { text: '新增卡片类型 = 注册渲染器组件，协议与 AI 层零互改（依赖倒置）；卡片容器统一处理信封，业务渲染器只消费 payload', options: { color: C.text2, breakLine: false } },
    ], { x: 7.0, y: 5.55, w: 5.6, h: 1.2, fontFace: FONT, fontSize: 10, valign: 'middle', margin: 0, lineSpacingMultiple: 1.25 });
  }

  /* ============ S14 当前进展 ============ */
  {
    const s = newSlide(pres);
    header(s, '03 · 关键模块 ⑤', '当前进展：前端 MVP 已完成并通过验收');
    const feats = [
      'React 18 + TS + Vite 全新实现（约 4200 行 / 25 文件）',
      '大盘完整复刻：多大盘管理 · 面板编辑器 · 实时预览',
      '智能对话：多会话 + SSE 六类事件流式渲染',
      '五种卡片渲染器注册表（协议对齐，联调即验收）',
      'Mock / Api 双 Provider：后端就绪换 URL，页面零改动',
      '修复 echarts visualMap 缺陷（原型遗留隐藏 bug）',
    ];
    feats.forEach((f, i) => {
      const x = 0.55 + (i % 2) * 6.25, y = 1.72 + Math.floor(i / 2) * 0.52;
      iconCircle(s, 'check', x, y, 0.32, C.green);
      s.addText(f, { x: x + 0.45, y: y - 0.02, w: 5.7, h: 0.36, fontFace: FONT, fontSize: 11, color: C.text2, valign: 'middle', margin: 0 });
    });
    const shots = [
      ['shot-overview.png', '运维概览：类型/告警/异常 TOP5/锁分析'],
      ['shot-editor.png', '大盘面板编辑器：多指标 + 实时预览'],
      ['shot-chat.png', '智能诊断全流程：轨迹 + 卡片 + 报告'],
    ];
    shots.forEach((sh, i) => {
      const x = 0.55 + i * 4.32;
      s.addShape('roundRect', { x: x - 0.04, y: 3.5, w: 4.16, h: 2.72, rectRadius: 0.06, fill: { color: C.white }, line: { color: C.line, width: 1 }, shadow: { type: 'outer', color: '1D2B45', opacity: 0.12, blur: 6, offset: 2, angle: 90 } });
      s.addImage({ path: path.join(__dirname, 'assets', sh[0]), x, y: 3.54, w: 4.08, h: 2.55 });
      s.addText(sh[1], { x, y: 6.14, w: 4.08, h: 0.28, fontFace: FONT, fontSize: 9.5, color: C.muted, align: 'center', margin: 0 });
    });
    s.addText('截图为 React 真实运行界面（mock 数据）——对 fixtures 契约开发，后端就绪后直接切换', { x: 0.55, y: 6.62, w: 12.2, h: 0.3, fontFace: FONT, fontSize: 10, color: C.muted, align: 'center', margin: 0 });
  }

  /* ============ S15 任务分派总览 ============ */
  {
    const s = newSlide(pres);
    header(s, '04 · 分工与验收', '任务分派：三个任务系列（本人 + 工程师 A + 工程师 B）');
    const series = [
      {
        x: 0.55, who: '本人', line: '系列 1 · 前端 + apiserver', color: C.blue, icon: 'monitor',
        goal: '平台入口与数据枢纽',
        tasks: ['1.1 骨架与契约桩（W1-2）', '1.2 会话与档案服务（W3-4）', '1.3 内部工具数据 API（W3-5）', '1.4 任务总线调度（W5-6）', '1.5 SSE 透传对接（W6）', '1.6 元数据/告警同步编排（W4-6）', '1.7 前端切换与联调（W7-10）', '1.8 Compose 验收交付（W11-12）'],
        exit: '北极星全链路 + 前端零改动切换 + 一键拉起',
      },
      {
        x: 4.68, who: '工程师 A', line: '系列 2 · Agent 集群（Python）', color: '0F5132', icon: 'cpu',
        goal: '多专家对话服务',
        tasks: ['2.1 契约样品库 fixtures（W2-3）', '2.2 Agent 运行时基座（W3-6）', '2.3 路由+诊断专家+续聊闭环（W5-8）', '2.4 问数专家（W9-10）'],
        exit: 'fixtures 回放=真实链路 · 无状态 2 副本 50 并发',
      },
      {
        x: 8.81, who: '工程师 B', line: '系列 3 · 数据与集成', color: '6E3A07', icon: 'db',
        goal: '库表基座 + 防腐 + 外采',
        tasks: ['3.1 PG Schema + 部署编排（W1-2）', '3.2 旧系统适配器 ×3（W3-6）', '3.3 外采契约跟进 + 适配器（契约后 1 周）'],
        exit: '迁移幂等 · 适配器注入即用 · 外采 shadow→active',
      },
    ];
    series.forEach(se => {
      card(s, { x: se.x, y: 1.66, w: 3.95, h: 5.05 });
      iconCircle(s, se.icon, se.x + 0.24, 1.9, 0.5, se.color);
      s.addText(se.line, { x: se.x + 0.88, y: 1.92, w: 3.0, h: 0.3, fontFace: FONT, fontSize: 12.5, bold: true, color: C.text, margin: 0 });
      s.addText(se.who + ' · ' + se.goal, { x: se.x + 0.88, y: 2.22, w: 3.0, h: 0.26, fontFace: FONT, fontSize: 9.5, color: C.muted, margin: 0 });
      se.tasks.forEach((t, i) => {
        const y = 2.68 + i * 0.42;
        s.addShape('roundRect', { x: se.x + 0.24, y, w: 3.5, h: 0.35, rectRadius: 0.05, fill: { color: C.lighter }, line: { color: C.line, width: 0.75 } });
        s.addText(t, { x: se.x + 0.36, y: y - 0.01, w: 3.3, h: 0.35, fontFace: FONT, fontSize: 9, color: C.text2, valign: 'middle', margin: 0 });
      });
      const ey = 2.68 + se.tasks.length * 0.42 + 0.08;
      s.addShape('roundRect', { x: se.x + 0.24, y: ey, w: 3.5, h: 0.72, rectRadius: 0.06, fill: { color: C.light }, line: { type: 'none' } });
      s.addText([
        { text: '系列出口验收：', options: { bold: true, color: C.primary, breakLine: true } },
        { text: se.exit, options: { color: C.text2, breakLine: false } },
      ], { x: se.x + 0.36, y: ey + 0.04, w: 3.3, h: 0.66, fontFace: FONT, fontSize: 8.5, valign: 'top', margin: 0, lineSpacingMultiple: 1.15 });
    });
  }

  /* ============ S16 排期与集成关卡 ============ */
  {
    const s = newSlide(pres);
    header(s, '04 · 分工与验收', '排期总览：三线并行 · 三个集成关卡');
    const gx = 1.75, gw = 11.0, weeks = 12, colw = gw / weeks, gy = 2.3, laneH = 1.06, laneGap = 0.34;
    // 周刻度
    for (let i = 0; i < weeks; i++) {
      s.addText('W' + (i + 1), { x: gx + i * colw, y: 1.98, w: colw, h: 0.26, fontFace: 'Arial', fontSize: 8.5, color: C.muted, align: 'center', margin: 0 });
      if (i > 0) s.addShape('line', { x: gx + i * colw, y: gy - 0.05, w: 0, h: 3 * (laneH + laneGap) + 0.15, line: { color: 'EEF3FA', width: 0.75 } });
    }
    // 月份分隔 M1/M2/M3
    [4, 8].forEach(w => {
      s.addShape('line', { x: gx + w * colw, y: 1.9, w: 0, h: 3 * (laneH + laneGap) + 0.28, line: { color: C.line, width: 1, dashType: 'dash' } });
    });
    ['M1 · 第 1 月', 'M2 · 第 2 月', 'M3 · 第 3 月'].forEach((m, i) => {
      s.addText(m, { x: gx + i * 4 * colw, y: 5.62, w: 4 * colw, h: 0.26, fontFace: FONT, fontSize: 9.5, bold: true, color: C.text2, align: 'center', margin: 0 });
    });
    // 三条泳道
    const lanes = [
      { name: '系列1 · 本人', color: C.blue, bars: [['1.1 骨架与桩', 1, 2], ['1.2-1.3/1.6 数据服务', 3, 6], ['1.4-1.5 任务与 SSE', 5, 6], ['1.7 前端联调', 7, 10], ['1.8 交付', 11, 12]] },
      { name: '系列2 · 工程师A', color: '0F5132', bars: [['2.1 契约 fixtures', 2, 3], ['2.2 运行时基座', 3, 6], ['2.3 诊断专家+续聊', 5, 8], ['2.4 问数专家', 9, 10]] },
      { name: '系列3 · 工程师B', color: '6E3A07', bars: [['3.1 Schema+编排', 1, 2], ['3.2 旧系统适配器', 3, 6], ['3.3 外采适配器', 9, 11, true]] },
    ];
    lanes.forEach((ln, li) => {
      const y = gy + li * (laneH + laneGap);
      s.addText(ln.name, { x: 0.4, y: y + laneH / 2 - 0.2, w: 1.3, h: 0.4, fontFace: FONT, fontSize: 9.5, bold: true, color: C.text2, align: 'right', valign: 'middle', margin: 0, lineSpacingMultiple: 1.1 });
      s.addShape('roundRect', { x: gx, y, w: gw, h: laneH, rectRadius: 0.05, fill: { color: C.lighter }, line: { color: C.line, width: 0.75 } });
      ln.bars.forEach(b => {
        const [label, w1, w2, dashed] = b;
        const bx = gx + (w1 - 1) * colw + 0.04, bw = (w2 - w1 + 1) * colw - 0.08;
        s.addShape('roundRect', { x: bx, y: y + 0.18, w: bw, h: 0.5, rectRadius: 0.06, fill: { color: ln.color, transparency: dashed ? 55 : 0 }, line: { type: 'none' } });
        s.addText(label, { x: bx, y: y + 0.17, w: bw, h: 0.5, fontFace: FONT, fontSize: 8.5, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
      });
    });
    // 集成关卡旗标
    const gates = [
      [6, '集成① W6', 'apiserver↔Agent 换 URL 互连，fixtures 双端一致'],
      [8, '集成② W8', '北极星前半段真实链路（诊断→报告→追问）'],
      [10, '集成③ W10', '问数接入 + 全场景串联'],
    ];
    gates.forEach(([wk, t, d]) => {
      const x = gx + wk * colw;
      iconCircle(s, 'flag', x - 0.17, 6.05, 0.34, C.accent);
      s.addText(t, { x: x - 1.05, y: 6.42, w: 2.1, h: 0.26, fontFace: FONT, fontSize: 9.5, bold: true, color: 'D46B08', align: 'center', margin: 0 });
      s.addText(d, { x: x - 1.5, y: 6.66, w: 3.0, h: 0.45, fontFace: FONT, fontSize: 8, color: C.text2, align: 'center', margin: 0, lineSpacingMultiple: 1.1 });
    });
    s.addText('依赖要点：系列2 只依赖契约 fixtures（不依赖本人进度）；3.1 零外部依赖 W1 即可开工；3.3 外采契约为唯一外部依赖（虚线示意）', { x: 0.55, y: 7.08, w: 12.2, h: 0.3, fontFace: FONT, fontSize: 9.5, color: C.muted, align: 'center', margin: 0 });
  }

  /* ============ S17 验收机制与风险 ============ */
  {
    const s = newSlide(pres);
    header(s, '04 · 分工与验收', '验收机制：契约优先，集成 = 换 URL');
    // 左：机制流程
    card(s, { x: 0.55, y: 1.66, w: 6.1, h: 3.0 });
    s.addText('契约优先工作法', { x: 0.85, y: 1.86, w: 4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    const flow = [
      ['文档契约 v1.1', '内部 API / SSE 事件 / 卡片 JSON / 工具清单 / 状态机'],
      ['contracts/ fixtures', 'JSON 样品 + Schema 校验，进 CI'],
      ['三系列各自对 fixtures 自测', '不依赖对方进度'],
      ['集成 = 换 URL / 注入实现', '与前端已验证的 Mock→Api 模式同构'],
    ];
    flow.forEach((f, i) => {
      const y = 2.3 + i * 0.6;
      s.addShape('ellipse', { x: 0.88, y: y + 0.03, w: 0.34, h: 0.34, fill: { color: C.blue }, line: { type: 'none' } });
      s.addText(String(i + 1), { x: 0.88, y: y + 0.03, w: 0.34, h: 0.34, fontFace: 'Arial', fontSize: 11, bold: true, color: C.white, align: 'center', valign: 'middle', margin: 0 });
      s.addText(f[0], { x: 1.38, y, w: 2.55, h: 0.4, fontFace: FONT, fontSize: 10.5, bold: true, color: C.text, valign: 'middle', margin: 0 });
      s.addText(f[1], { x: 3.95, y, w: 2.6, h: 0.4, fontFace: FONT, fontSize: 8.5, color: C.text2, valign: 'middle', margin: 0, lineSpacingMultiple: 1.1 });
      if (i < 3) s.addShape('line', { x: 1.05, y: y + 0.38, w: 0, h: 0.22, line: { color: C.line, width: 1.2 } });
    });
    s.addShape('roundRect', { x: 0.55, y: 4.86, w: 6.1, h: 0.62, rectRadius: 0.06, fill: { color: C.amberBg }, line: { type: 'none' } });
    s.addText('纪律：契约变更必须先改 fixture 并三方会签，再改实现', { x: 0.85, y: 4.86, w: 5.6, h: 0.62, fontFace: FONT, fontSize: 10.5, bold: true, color: '8A5A00', valign: 'middle', margin: 0 });
    // 左下：验收标准示例
    card(s, { x: 0.55, y: 5.66, w: 6.1, h: 1.35 });
    s.addText('验收标准示例（全部可执行 / 可量化）', { x: 0.85, y: 5.82, w: 5, h: 0.3, fontFace: FONT, fontSize: 11, bold: true, color: C.text, margin: 0 });
    ['pytest 契约回放一致 · LLM 路由准确率 ≥80%', '迁移幂等两遍无差异 · 2 副本 50 并发无串扰', '越界问数 100% 拒答 · 重启恢复对账通过'].forEach((t, i) => {
      s.addText('✓ ' + t, { x: 0.88, y: 6.14 + i * 0.27, w: 5.5, h: 0.26, fontFace: FONT, fontSize: 9, color: C.text2, margin: 0 });
    });
    // 右：风险表
    s.addText('风险与缓解', { x: 7.0, y: 1.66, w: 4, h: 0.32, fontFace: FONT, fontSize: 13, bold: true, color: C.text, margin: 0 });
    const risks = [
      ['R1', '外采契约未到', '适配器抽象 + 自建工具兜底 + 周会跟踪索取'],
      ['R2', '网络连通性未确认', 'W1 确认部署位与旧系统/AI 平台网络'],
      ['R3', 'AI 平台模型未定名', 'LLM 端口配置化，任一兼容端点可开发'],
      ['R4', '契约漂移', 'fixtures 进 CI + 三方会签纪律'],
      ['R5', '系列2 关键路径', '2.3 延后时本人以内部 API 桩提前联调'],
    ];
    risks.forEach((r, i) => {
      const y = 2.1 + i * 0.98;
      card(s, { x: 7.0, y, w: 5.78, h: 0.86, shadow: false });
      s.addShape('roundRect', { x: 7.18, y: y + 0.24, w: 0.55, h: 0.38, rectRadius: 0.06, fill: { color: i === 0 ? C.redBg : C.light }, line: { type: 'none' } });
      s.addText(r[0], { x: 7.18, y: y + 0.23, w: 0.55, h: 0.38, fontFace: 'Arial', fontSize: 11, bold: true, color: i === 0 ? C.red : C.primary, align: 'center', valign: 'middle', margin: 0 });
      s.addText(r[1], { x: 7.88, y: y + 0.1, w: 4.7, h: 0.3, fontFace: FONT, fontSize: 10.5, bold: true, color: C.text, margin: 0 });
      s.addText(r[2], { x: 7.88, y: y + 0.42, w: 4.75, h: 0.36, fontFace: FONT, fontSize: 9, color: C.text2, margin: 0 });
    });
  }

  /* ============ S18 结尾（深色） ============ */
  {
    const s = newSlide(pres, true);
    s.addShape('ellipse', { x: -2.4, y: 3.9, w: 7, h: 7, fill: { color: C.primary, transparency: 84 }, line: { type: 'none' } });
    s.addShape('ellipse', { x: 10.8, y: -2.6, w: 6.4, h: 6.4, fill: { color: C.blue, transparency: 86 }, line: { type: 'none' } });
    s.addText('NORTH STAR', { x: 0.85, y: 1.35, w: 6, h: 0.35, fontFace: 'Arial', fontSize: 12, bold: true, color: C.ice, charSpacing: 3, margin: 0 });
    s.addText('实例变慢 → 大盘发现异常 → 一句话发起诊断', { x: 0.85, y: 1.85, w: 11.8, h: 0.75, fontFace: FONT, fontSize: 30, bold: true, color: C.white, margin: 0 });
    s.addText('→ 真实证据采集 → 异步深度扫描 → 可追问的诊断报告', { x: 0.85, y: 2.62, w: 11.8, h: 0.7, fontFace: FONT, fontSize: 30, bold: true, color: C.ice, margin: 0 });
    s.addText('MVP 北极星场景——一次链路验证五项架构核心假设', { x: 0.85, y: 3.5, w: 10, h: 0.35, fontFace: FONT, fontSize: 13, color: '9FB6DF', margin: 0 });
    // 需要的支持
    s.addText('需要支持的事项', { x: 0.85, y: 4.35, w: 5, h: 0.35, fontFace: FONT, fontSize: 14, bold: true, color: C.white, margin: 0 });
    const asks = [
      ['users', '工程师 A / B 于 W1 到岗', 'Python AI 方向 ×1 · 数据集成方向 ×1'],
      ['link', '网络连通性 W1 内确认', '部署位 ↔ 旧监控 / DBaaS / AI 平台 / 外采'],
      ['send', '外采 API 契约本周索取', '唯一外部硬依赖，行动项已列'],
    ];
    asks.forEach((a, i) => {
      const x = 0.85 + i * 4.05;
      s.addShape('roundRect', { x, y: 4.85, w: 3.85, h: 1.35, rectRadius: 0.08, fill: { color: '16223A' }, line: { color: '32456B', width: 0.75 } });
      iconCircle(s, a[0], x + 0.22, 5.05, 0.44, C.blue);
      s.addText(a[1], { x: x + 0.8, y: 5.02, w: 2.95, h: 0.55, fontFace: FONT, fontSize: 11, bold: true, color: C.white, margin: 0, lineSpacingMultiple: 1.1 });
      s.addText(a[2], { x: x + 0.24, y: 5.62, w: 3.4, h: 0.45, fontFace: FONT, fontSize: 9, color: '9FB6DF', margin: 0, lineSpacingMultiple: 1.15 });
    });
    s.addText('DB Copilot · 数据库智能运维平台　—　谢谢', { x: 0.85, y: 6.7, w: 11.6, h: 0.35, fontFace: FONT, fontSize: 11, color: C.muted, margin: 0 });
  }

  await pres.writeFile({ fileName: path.join(__dirname, '..', '数据库智能运维平台-项目汇报.pptx') });
  console.log('DONE, slides =', pageNo);
}

main().catch(e => { console.error(e); process.exit(1); });
