---
name: dever-crm
description: Use when 修改 Dever CRM 组件，包括客户中心、客户、客户资产、阶段任务、资料模板、财务流水、组织资源、work 前台 API、CRM front 插件、页面权限和迁移行为。
version: 0.1.0
---

# CRM 组件

本组件 skill 必须和 `shemic-dever` 一起使用。先遵守 Dever 框架规则，再按这里的 CRM 组件边界修改。

## 事实来源

- 组件源码：`backend/package/crm`
- 组件声明：`backend/package/crm/dever.json`
- Model：`model`
- Service：`service`
- API：`api`
- 后台页面：`front/page`
- 前端插件源码：`front/src/plugin.ts`

## 硬规则

- 普通后台 CRUD 优先使用 `Model + package/front + page JSON`。
- 不为客户、模板、财务、组织等普通维护页新增 CRUD wrapper API 或 Service。
- 客户资产、阶段任务、任务执行、资料提交、财务流水、飞书登录、AI 填写等真实业务流程留在 `service`。
- API 必须保持薄，只做请求解析、鉴权上下文和响应；业务编排放到 Service。
- 不手改生成文件、编译产物或项目级菜单来补 CRM 菜单。
- 不在 `dever.json` 写 `apiRoots`；API 扫描由 Dever 按组件自动处理。
- 公开接口只允许登录相关入口：`/crm/work/login`、`/crm/work/feishu_login`、`/crm/work/feishu_config`。
- CRM 依赖 `front`、`source`、`user`，避免新增反向依赖或 import cycle。

## 菜单边界

CRM 后台菜单由 `dever.json` 声明：

- `crm-center`：客户中心。
- `crm-customer-manage`：客户管理。
- `crm-business-config`：业务配置。
- `crm-org-manage`：组织管理。

新增 CRM 后台页面时优先挂到以上分组。不要在项目 `front.json` 里重复维护这些组件菜单。

## Page 和 Front 规则

- 后台页面路径继续放在 `front/page/admin/...`。
- 标准 list/update/detail 页应复用 front 自动推导 model 和 action。
- 左分类右列表只在右侧列表需要刷新时刷新数据，不为分类切换强行跳转 URL。
- 自定义 React/plugin 只用于 work 前台、任务表单、客户表格、详情弹窗等 page JSON 无法表达的交互。
- 不修改 `front/dist`；插件源码在 `front/src/plugin.ts`，发布产物由构建命令生成。

## Service/API 规则

- `service/work*`：work 前台登录、任务执行、协作、资料提交和副作用。
- `service/customer_asset.go`：客户资产创建、详情和业务一致性。
- `service/data_record.go`：资料模板、字段和记录保存。
- `service/rule.go`：脚本规则校验和执行边界。
- `service/setting`：后台配置页需要的真实聚合、校验和跨表保存逻辑。

新增逻辑时先找同目录现有 Service 复用。只有涉及事务、跨表规则、外部系统、状态流转或复杂校验时才写 Service。

## 常见检查

- 菜单不显示：先检查 `dever.json`、`module/crm/main.go` 和生成后的组件注册。
- 权限报错：先检查 page action、public 路由和 work 登录上下文，不要临时放开通配权限。
- 保存后列表异常：先检查 model sort/status/cate 字段、page 数据刷新和 Service 是否覆盖了标准保存流程。
- 飞书或前台登录异常：先检查公开路由、密钥读取和 token/session 写入，不要在日志输出密钥。
