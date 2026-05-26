# 在线客户端

`online-client` 保存在线 Web 前端，用于访问在线服务端提供的用户、账户和交易接口。

## 目录结构

```text
online-client/
└── frontend/   # Vue 2 + Element UI 前端应用
```

## 功能范围

- 用户登录、注册和登录态检查。
- 管理员用户管理。
- 账户查询、创建、导入和管理。
- 交易准备、签名提交、交易列表和统计信息查看。
- 个人资料和密码修改。

## 本地运行

```bash
cd frontend
npm install
npm run serve
```

默认开发服务地址：

```text
http://localhost:8090
```

## 构建

```bash
cd frontend
npm run build
```

构建结果输出到 `frontend/dist/`。

## 相关文档

- `frontend/README.md`：前端应用详细说明。
- `frontend/API_DOCUMENTATION.md`：前端对接接口说明。
- `frontend/PRODUCTION_DEPLOYMENT_GUIDE.md`：生产环境部署说明。
