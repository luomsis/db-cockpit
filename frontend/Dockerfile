# ============ DB Cockpit · 前端镜像（固定 linux/amd64） ============
# 两阶段构建；两个 FROM 均显式固定 amd64 —— 无论在哪台机器（含 arm64 Mac）
# 构建产物都是 amd64，可直接 docker save 迁移到内网 x86_64 Linux。

# ---------- 构建阶段 ----------
FROM --platform=linux/amd64 node:20-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build

# ---------- 运行阶段 ----------
FROM --platform=linux/amd64 nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
