/* ================= 构建期生成 version.json（左下角服务版本展示） =================
 * 输出 frontend/public/version.json：git 短哈希 + 构建时间。
 * 优先取环境变量 GIT_SHA / BUILD_TIME（docker 构建时由 ARG 注入），
 * 否则本地 git 命令 / 当前时间（非 git 环境 fallback unknown）。
 */
import { execSync } from 'node:child_process';
import { writeFileSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

let gitSHA = process.env.GIT_SHA || 'unknown';
if (!process.env.GIT_SHA) {
  try {
    gitSHA = execSync('git rev-parse --short HEAD', { stdio: ['ignore', 'pipe', 'ignore'] }).toString().trim() || 'unknown';
  } catch (e) { /* 非 git 环境 */ }
}
const buildTime = process.env.BUILD_TIME || new Date().toISOString();

const out = { name: 'frontend', gitSHA, buildTime };
mkdirSync(join(root, 'public'), { recursive: true });
writeFileSync(join(root, 'public', 'version.json'), JSON.stringify(out, null, 2) + '\n');
console.log(`[version] frontend ${out.gitSHA} @ ${out.buildTime}`);
