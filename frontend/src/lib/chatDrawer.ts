/* ================= 全局聊天抽屉控制 =================
 * 页面各处的「问 AI」入口统一打开侧边浮窗（而非跳转 /chat 完整页）。
 * Shell 监听 open-chat-drawer 事件唤起抽屉；可选携带提问草稿（chatAskDraft），
 * 由 useChat 挂载时消费并自动发送。
 */
export const ASK_DRAFT_KEY = 'chatAskDraft';

export function openChatDrawer(askDraft?: string) {
  if (askDraft) sessionStorage.setItem(ASK_DRAFT_KEY, askDraft);
  window.dispatchEvent(new Event('open-chat-drawer'));
}
