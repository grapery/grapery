# 创作链路契约与验收（2026-10-16）

本轮覆盖 Voyager iOS、Grapery API 与 Grapery Agent 的故事碎片、故事板和故事角色。代码完成、离线测试、真实模型效果、上线生效是四种不同证据。本文记录实现与验证边界，不把单元测试通过称为生成质量优秀。

## 业务不变量

1. **用户设定是权威输入。** 工作流自定义提示词不能移除故事背景、视觉设定、已接受正文和编辑范围。结构化输出通过校验后才能进入下一阶段；只有一次定向修复机会，失败保留草稿，不用通用套话继续付费绘图。
2. **局部修改有稳定范围。** 碎片的单页视觉修改复用分格、对白、节拍，仅调整绘制指令；保留全局 VisualBible、StoryState 和其他页。故事板通过稳定 scene ID 选择目标，保留非目标 scene ID、媒体与正文基线；禁止缺图后通过 compactMap 错位选页。
3. **改稿先生成，成功后再提交。** 故事板读取当前内容的 SHA-256 revision，生成候选版本，事务锁定原稿、比对 revision 后提交。模型错误、写入失败、旧客户端版本冲突均不能先删除已接受场景。历史基线保存在 generation run 中；历史查询只对本人开放。
4. **画完和验收分开。** 新碎片记录 `unreviewed / passed / blocked`，包含每页与整稿状态。审核不可用、返回无效 JSON、未覆盖完整页面均不能伪装为通过。新生成作品发布由服务端存储的质量与结算状态决定。已接受背景图和客户端合字后的成品图必须保持可追溯关系。
5. **角色生成按阶段完成。** attributes、character、portrait、threeViews 分别记录状态、错误和尝试数。素材失败时保留已创建角色，返回 partial；重试只补未完成阶段。数据库 CAS 领取、续租、fence 写入与启动恢复防止两个 worker 覆盖同一任务。
6. **费用以账本为事实。** 同一完成结果的结算采用确定性交易 ID，在余额行锁内校验幂等意图并原子写入余额和 ledger。相同 ID 换金额/范围返回冲突。展示记录费用与模型 tokens 分开，历史无关联账本时显示无法核对。平台自动一致性修复和临时审核恢复不向用户重复收图费。

## 提示词结构

- 碎片：用户原文与锁定事实 → 页级叙事节拍 → 严格分格/对白/构图计划 → 人物身份参考优先 → 按格绘图 → 确定性拼版 → 一致性验收。多页计划采用随页数增长的输出预算，不能继续使用固定 1000-token 多页截断方案。
- 故事板：故事世界、父节点结尾、当前已接受正文、十字段角色快照、用户本轮修改与目标 scene ID → 分场候选 → 结构/范围检查 → 原子提交。明确的格数/布局优先于模板默认值。
- 角色：故事世界、已有角色与用户已填十字段 → 只补空字段 → 严格 JSON 长度/必填/类型校验与一次修复 → 创建/保存关联 → 身份参考继承的形象图与三视图。前端模型返回后也只补当前空项，避免覆盖等待期间的用户输入。

## 客户端与 API 契约

| 用途 | 契约 |
| --- | --- |
| 故事板详情 | `contentRevision` |
| 故事板结构修改 | `baseRevision`, `targetSceneIds`，冲突时重新读取并让用户选择基于哪版修改 |
| 故事板历史 | `GET /api/v1/storyboards/:id/revisions`，仅本人 |
| 碎片状态 | `result.quality`, `result.comicDocument`, `cost.amount` 为小数，`cost.tokens/state/text` |
| 检查暂不可用 | `POST /api/v1/fragments/generate/:taskId/recheck`，复用原图，不重新绘制 |
| 文本账单待结算 | `POST /api/v1/fragments/generate/:taskId/settle`，仅重结同一任务 |
| 合字后的发布图 | `POST /api/v1/fragments/generate/:taskId/prepare-publication`，`imageUrls`，复核成品图后再发布 |
| 角色部分完成 | task `status=partial`, `stages`，已有角色可查看，重试补缺项 |
| 同步角色补全 | `storyId`, `knownAttributes` |

所有写入入口均检查本人、草稿可变状态与资源归属，不能仅依赖前端按钮隐藏或新版路径的权限检查。兼容无新质量标记的历史作品，但不能借助清空 metadata 绕过新生成稿的验收。

## 费用与故障恢复的边界

- 保持现有 token 定价政策；余额预检按剩余实际格数估算，**不是整任务预授权或绝对最高费用承诺**。参考资产、供应商实际 token 与并发消费可能使最终账单不同。
- 有 `OperationID` 的图片调用会先查已存输出，若仅结算失败则结算同一结果并返回原 URL。确定失败且无输出的调用通过数据库 CAS 才可重试。
- 处理中且供应商结果未知的调用不能盲目重放。供应商幂等键/查询接口尚未全部统一，进程恰好在供应商成功、结果落库前退出时仍需要对账；本轮不宣称端到端 exactly-once。
- 老数据无任务关联账本时不回填虚构消费。当前费用文字是已记录结算数，不是供应商总成本或可保证的最终上限。

## 验证与发布

离线回归覆盖：非法调用不入队、场景跨故事板拒绝、陈旧 revision 冲突、场景插入失败回滚、非目标媒体保留、JSON 修复耗尽停止、修改不重写旧对白、审核不可用/空 JSON 不放行、账本并发去重/金额冲突/失败回滚、角色租约过期 fence 与定向重试。

Agent 的 `RunEvalWithSeeds` 和 `CreationContractSeeds(ownedStoryID)` 增加三类创作的真实产物门槛与人工叙事/视觉 rubric。`artifact_ready_rate` 与工具成功率分开统计；启动失败、超时不从分母中消失。**执行评测会调用真实生成并可能扣费，单元测试只验证 grader，不运行该套真实生成。**

发布前仍须在隔离测试世界运行真实模型样本，人工检查：用户设定保留、人物身份跨页一致、因果推进、三场/格数约束、局部编辑范围、对白可读性、失败恢复后账单。至少保留原提示词、版本、产物、评分理由和费用，比较改动前后同一组样本。没有样本证据不能报告通过率提升。

本轮未运行数据库迁移、重启现有服务或发布线上 prompt/workflow。角色任务新增持久化列需按既有 migration/AutoMigrate 流程在部署时创建；故事板需执行 [创建意图 hash 新列迁移](../scripts/storyboard_creation_request_hash_20261016.sql)。上线前先确认测试环境 schema 与实际生效的 workflow release/prompt snapshot，避免只改源码但仍运行旧模板。

## 本轮验证记录

- Grapery domain/service/repository/mysql/http 完整目标测试包通过，全部 Go 包编译检查通过。
- Agent eval/generation/client/prompt/workflow/http 完整目标测试包通过，Forge 默认 schema 契约测试通过。
- iOS 通用 arm64 模拟器目标 `build-for-testing` 成功，包含新增角色、碎片与故事板契约测试的编译。
- 指定模拟器测试运行受本机 CoreSimulator 版本不匹配阻挡：当前 1171.7.0，Xcode 要求 1174.9.2；未报告运行测试或真机交互验收通过。
- 没有调用真实生成模型、OSS 成品审核、收费、数据库迁移或部署。

当前选页使用独立 publicationManifest，完整 canonical 分镜保留，publicationAssets 记录已接受合字图与原页的对应关系。删除、撤销和保存后的重排只验收当前选页，未选坏页不会被标为通过。

选页删除/撤销在同一生成版本内恢复媒体选择；新的生成开始后清空客户端旧 URL 撤销栈，防止旧图配上新分镜。跨生成版本恢复需要服务端版本操作，不能只替换图片 URL。故事板历史目前提供读取基线，未新增自动回滚按钮。

## 参考来源

- [AWS：使用幂等 API 安全重试](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)：稳定意图标识、重复请求语义与原子副作用。
- [Anthropic：Agent 评测](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)：依据最终产物和多项 grader，而非只检查工具调用。
- [GORM：事务](https://gorm.io/docs/transactions.html)：同一事务内完成提交、错误时回滚。
