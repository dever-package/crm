# CRM Signing Flow Realignment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 跑通线索转客户到合同签署的默认入口流程，并修复十一维、规则输出和产品确认的错误定义。

**Architecture:** 保留数据模板、资料模板和流程配置三层。运行时只补充“分组子字段展示”“规则结果写回”“候选客户产品同步”三项通用能力，业务定义通过一份幂等 PostgreSQL 配置迁移重整。

**Tech Stack:** Go、Dever ORM、PostgreSQL、React/TypeScript、Dever page JSON。

**Verification Constraint:** 项目明确禁止运行 build 和任何自动化测试。本计划只执行格式化、静态搜索、JSON 解析、SQL 事务检查、Git 差异审计，并启动 `dever run` 供用户手工验证。

---

### Task 1: 固化设计与建立执行基线

**Files:**
- Create: `docs/superpowers/specs/2026-07-13-signing-flow-realignment-design.md`
- Create: `docs/superpowers/plans/2026-07-13-signing-flow-realignment.md`

- [ ] 核对当前分支为用户指定的 `main` 且工作区无未知修改。
- [ ] 保存已确认设计和本实施计划。
- [ ] 使用 `git diff --check` 检查文档格式，不执行测试或构建。

### Task 2: 让资料模板按分组子字段配置必填

**Files:**
- Modify: `service/work.go`
- Modify: `front/src/nodes/show/work-auth.tsx`
- Modify: `front/src/nodes/show/work-core.ts`

- [ ] 在 `attachWorkFormFieldOptions` 中为普通数据字段附加父分组的 `group_id`、`group_label` 和 `group_key`。
- [ ] 抽取后端父分组元数据帮助函数，只查询已启用且同模板的分组字段。
- [ ] 扩展 `WorkFormField` 类型，声明父分组元数据。
- [ ] 在 `buildWorkTaskFormState` 中把直接选择的分组子字段按 `group_id` 合成标签页；继续兼容旧的整组 FormField。
- [ ] 保证未分组的客户、资产和业务字段仍进入原来的普通分区。

### Task 3: 支持规则写回数据字段和候选产品

**Files:**
- Modify: `service/rule.go`
- Modify: `service/work_todo_execute.go`
- Modify: `service/customer_product.go`

- [ ] 给 `TaskRuleResult` 增加 `OutputFields map[string]any` 和 `ProductCodes []string`。
- [ ] 在 `normalizeTaskRuleResult` 中读取脚本返回的 `fields` 与 `product_codes`，保持原返回格式兼容。
- [ ] 在规则 dry-run 响应中展示标准化后的字段输出和候选产品编码。
- [ ] 新增 `applyTaskRuleOutputs`，按字段编码定位数据模板分类，并复用 `saveWorkFormDataRecords` 写入当前流程归属。
- [ ] 新增 `SyncCandidateCustomerProducts`：只创建、恢复或淘汰候选状态，不覆盖已确认、处理中和已完成产品。
- [ ] 在规则待办通过时先记录操作、应用字段和候选产品，再完成待办；任一失败由现有事务整体回滚。

### Task 4: 重整当前租户的签约配置

**Files:**
- Create: `migrations/postgres/012_signing_flow_realignment.sql`

- [ ] 补齐“线索信息”分类和“线索补充信息”模板，停用“客户来源与基础建档”。
- [ ] 将模板重命名和归类为“接单与建群”“资产基础信息”“十一维资料”“诊断结果”“专业协作意见”“合同信息”。
- [ ] 停用签约方向和动态产品编码字段，保留候选 T 字段。
- [ ] 重建“接单建档”资料模板，使其同时包含客户、接单和资产字段。
- [ ] 将十一维资料模板改为直接引用 96 个子字段，十二个“探针选项”必填，其余字段可选。
- [ ] 更新 T 节点规则，使其通过 `fields` 写回候选 T 和置信度。
- [ ] 将默认入口流程重建为六个阶段，并把“确认适用产品”配置为 `product` 任务。
- [ ] 把律师、ALA、财务协作任务放到“签约协作”阶段，把合同表单放到最终“合同签署”阶段。
- [ ] 修复全部产品分类；仅资产运营类产品默认关联现有运营流程。
- [ ] 删除无流程阶段和旧运行数据，保留客户、资产、线索和基础配置。
- [ ] 使用 PostgreSQL 事务和存在性判断保证迁移重复执行不会创建重复配置。

### Task 5: 静态审计并启动环境

**Files:**
- Review: all modified files

- [ ] 对 Go 文件运行 `gofmt`，不运行 Go 测试或构建。
- [ ] 使用 JSON 解析命令检查改动涉及的 page JSON；若未修改 JSON，则记录为不适用。
- [ ] 使用 `git diff --check` 和定向 `rg` 检查重复逻辑、旧签约方向和错误任务类型。
- [ ] 在事务中执行 `012_signing_flow_realignment.sql`，查询流程、阶段、任务、表单必填和产品分类结果。
- [ ] 按项目指令启动 `dever run`；不把启动成功等同于完整业务验收。
- [ ] 向用户提供访问地址和手工验收顺序，明确未运行自动化测试和构建命令。
