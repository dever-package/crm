# 简化流程实施计划

> 实施方式：在 `feature/simple-workflow` 分支由主代理按批次连续执行，不使用子代理。

**目标：** 将现有 CRM 的流程能力收敛为“流程 -> 阶段 -> 任务”，支持阶段自动/手动分配、普通与协作任务统一分配、显式完成阶段、终止流失和流程调度权限，同时保持客户、资产、十一维资料表单和规则核验能力不变。

**架构：** 继续使用现有 Dever Model + Page JSON 管理配置。运行时拆为阶段进入、人员分配、阶段流转三个聚焦服务；`WorkService` 只负责鉴权和请求编排；工作台在现有客户详情中增加一个紧凑的流程操作区。流程关系仅支持有序阶段和一个可选后续流程，不引入 ReactFlow、条件分支、版本表或新的角色体系。

**技术栈：** Go、Dever ORM/Page JSON、React/TypeScript、现有 front 组件库。

**验证约束：** 按项目要求不运行 `npm run build`、Go/前端测试或任何 test 命令。仅执行 `gofmt`、JSON 语法解析、Dever 静态审计、Git diff 自查，并重启现有 `dever run` 环境供人工验证。

---

## 任务 1：收敛配置模型与后台表单

**文件：**
- 修改：`model/workflow.go`
- 修改：`model/stage.go`
- 修改：`model/task.go`
- 修改：`model/customer_stage.go`
- 修改：`model/staff.go`
- 修改：`model/options.go`
- 修改：`service/setting/workflow.go`
- 修改：`front/page/admin/workflow/update.json`
- 修改：`front/page/admin/stage/update.json`
- 修改：`front/page/admin/task/update.json`
- 修改：`front/page/admin/staff/update.json`

- [x] 流程增加唯一默认入口和可选后续流程；校验自引用、循环引用、删除引用和默认入口唯一性。
- [x] 阶段增加 `auto/manual` 分配方式，负责部门和分配方式均为必填。
- [x] 任务负责方式统一为 `stage/auto/manual`；移除后台固定人员配置，自动/手动任务只配置目标部门。
- [x] 人员增加“流程调度”权限；进度增加完成/终止时间和终止原因。
- [x] 后台表单只暴露业务人员真正需要的字段，复用现有 Model 选项和 CRUD Hook。
- [x] 对修改的 Go 文件执行 `gofmt`，对四个 JSON 文件做语法解析。
- [x] 提交：`feat: simplify workflow configuration`

## 任务 2：实现统一的自动与手动分配

**文件：**
- 新增：`service/workflow_assignment.go`
- 修改：`service/workflow_runtime.go`
- 修改：`service/work_todo_execute.go`
- 修改：`service/work_query.go`

- [x] 实现按部门查询启用人员，并按“当前活跃资产最少、最近分配最早、ID”选择阶段负责人。
- [x] 实现任务自动分配，按“待处理任务最少、最近分配最早、ID”选择负责人；同批任务逐条创建，后续选择可看到前一条负载。
- [x] 阶段跟随任务直接分配给当前阶段负责人；手动任务保留目标部门但负责人为空。
- [x] 手动未分配任务不再允许部门成员领取或执行，只有负责人明确后才能办理。
- [x] 首次启动必须使用默认入口流程；自动首阶段直接分配，手动首阶段要求调用方提供负责人。
- [x] 对修改的 Go 文件执行 `gofmt` 和 diff 自查。
- [x] 提交：`feat: add workflow assignment and transitions`

## 任务 3：把阶段流转改为显式操作

**文件：**
- 新增：`service/workflow_transition.go`
- 修改：`service/workflow_runtime.go`
- 修改：`service/work_todo_execute.go`
- 修改：`service/work_audit.go`

- [x] 删除“任务完成后自动跳阶段”；自动规则仍可自动执行，但只改变任务状态。
- [x] 实现“完成阶段”：校验当前操作者、所有必做任务、目标阶段分配方式，然后取消未完成可选任务并进入下一阶段。
- [x] 当前流程最后阶段完成后，仅按配置的 `next_workflow_id` 进入后续流程；无后续流程则完成整个进度。
- [x] 实现“终止/流失”：仅当前负责人可操作，原因必填，取消待办且不进入后续流程。
- [x] 每次进入、完成、终止、分配和改派都写入现有操作日志，包含操作者与关键前后值。
- [x] 对修改的 Go 文件执行 `gofmt` 和 diff 自查。
- [x] 已与统一分配一起提交，保持运行时逻辑原子完整。

## 任务 4：增加工作台流程接口和调度权限

**文件：**
- 修改：`service/work.go`
- 新增：`service/work_flow_actions.go`
- 修改：`service/work_query.go`
- 修改：`api/work.go`

- [x] 登录会话和 `/crm/work/me` 返回 `can_dispatch`。
- [x] 客户详情返回当前流程、阶段、负责人、阶段任务、是否可完成/终止/分配等能力字段。
- [x] 增加候选人员、分配任务、改派任务、更换阶段负责人、完成阶段和终止流程接口；API 层保持薄封装。
- [x] 当前阶段负责人可分配本阶段的手动任务；调度员可对全部活跃业务做分配、改派、更换负责人和完成阶段。
- [x] 调度员不能代办任务、跳过必做项、修改任务结果或代替负责人终止流程。
- [x] 客户列表增加 `mine/all` 查询范围；非调度员强制为 `mine`。
- [x] 对修改的 Go 文件执行 `gofmt` 和 diff 自查。
- [x] 提交：`feat: add simple workflow controls to workbench`

## 任务 5：简化工作台流程操作

**文件：**
- 修改：`front/src/nodes/show/work-core.ts`
- 新增：`front/src/nodes/show/work-flow-actions.tsx`
- 修改：`front/src/nodes/show/work-auth.tsx`

- [x] 扩展现有类型，承接当前进度、阶段任务、候选人员和权限字段。
- [x] 在客户资产详情的“流程”页签顶部增加单一流程操作区：进度、负责人、任务、分配/改派、完成阶段、终止。
- [x] 当前负责人完成阶段时，自动阶段直接确认；手动阶段在同一操作中选择下一负责人。
- [x] 调度员在现有工作台看到“我的/全部”切换和管理动作，普通员工界面不增加多余控件。
- [x] 复用现有 `workApi`、按钮和提示组件，不把业务逻辑继续堆进 `work-auth.tsx`。
- [x] 仅做 TypeScript/JSX 静态阅读和 diff 自查，不运行 build/test。
- [x] 与工作台接口一起提交，保证前后端契约同步。

## 任务 6：补齐默认流程并做静态交付检查

**文件：**
- 视现有项目约定修改：CRM 模型 seed 或幂等初始化逻辑
- 修改：`docs/superpowers/specs/2026-07-12-simple-workflow-design.md`（仅当实现细节需要同步）

- [ ] 为无流程的新租户提供一套可编辑默认配置：签署流程包含接单建档、资料收集、诊断核验、产品确认、合同签署、签约确认；运营流程由签署流程显式衔接。
- [ ] 不按 MKT/NPL/PM 名称硬编码部门，只绑定实际部门 ID；十一维表单和自动规则继续作为任务配置。
- [ ] 检查全部 JSON 可解析、Go 文件已格式化、前端引用和导出一致、无旧自动流转调用残留。
- [ ] 运行 Dever 静态审计（不运行 build/test），重启 `dever run` 并确认 8082 监听。
- [ ] 汇总人工验证路径和已知边界。
- [ ] 提交：`chore: finalize simple workflow defaults`
