# App Store Connect — App 内购买项目配置手册

> **用途**：在 App Store Connect 中创建/维护 Voyager iOS 自动续期订阅 SKU。  
> **维护**：产品 ID、价格或权益变更时，同步更新本文档及代码中的 seed 数据。  
> **最后更新**：2026-05-21

---

## 文档与代码对齐关系

| 来源 | 路径 | 说明 |
|------|------|------|
| 主站方案与价格 | `grapery/scripts/membership_iap_seed.sql` | `subscription_plans` 表，`iap_product_id` / 价格 / 权益 |
| VipPay 商品 catalog | `grapery/internal/repository/pay/iap_grapery_seed.go` | 启动时写入 `iap_products`（缺失行才插入） |
| iOS StoreKit 本地测试 | `voyager/voyager/Resources/grapery.storekit` | Xcode 本地 IAP 调试，Product ID 须与 ASC 一致 |
| iOS 方案展示兜底 | `voyager/voyager/Services/MembershipStoreKitFallback.swift` | planId ↔ productId 映射 |
| IAP 环境变量与 Webhook | `grapery/docs/iap_config_requirements.md` | Apple API Key、Server Notifications |
| VipPay 部署配置 | `grapery/cmd/vippay/CONFIG.md` | JWT、数据库、Apple 密钥 |

**变更流程（建议）**：

1. 改 App Store Connect 商品 → 更新本文档  
2. 改价格/配额 → 更新 `membership_iap_seed.sql` + `iap_grapery_seed.go`  
3. 改 Product ID → 上述文件 + `grapery.storekit` + `MembershipStoreKitFallback.swift` 一并改  
4. 重新跑 seed / 部署 VipPay / 提交 App 新版本  

---

## 0. 前置信息

| 项 | 值 |
|----|-----|
| App 名称 | Voyager / Grapery |
| App Bundle ID | `com.rankquantity.voyager` |
| Apple Team ID | `UZLNTVX73Y` |
| App 内部 ID（StoreKit 配置） | `6752515439` |
| 订阅类型 | **自动续期订阅**（Auto-Renewable Subscription） |
| 在售 SKU 数量 | **4**（月付/年付 × 基础/高级） |
| 主销售货币（中国区） | **CNY 人民币** |
| VipPay 验单 Bundle ID 环境变量 | `APPLE_BUNDLE_ID=com.rankquantity.voyager` |

> **Product ID 必须与下表完全一致**。不一致会导致 VipPay 验单后无法匹配 `iap_products` / `subscription_plans`，用户配额与 VIP 状态不会更新。

---

## 1. 进入配置入口

1. 打开 [App Store Connect](https://appstoreconnect.apple.com/)
2. **我的 App** → 选择 Voyager / Grapery App
3. 左侧 **营利**（Monetization）→ **订阅**（Subscriptions）
4. 若尚无订阅组，先按 [§2](#2-创建订阅组只需一次) 创建订阅组

---

## 2. 创建订阅组（只需一次）

| 字段 | 填写内容 |
|------|----------|
| **参考名称**（内部，用户不可见） | `Grapery Membership` |
| **App Store 显示名称 — 简体中文** | `Grapery 会员订阅` |
| **App Store 显示名称 — 英文 (en-US)** | `Grapery Membership` |
| **描述 — 简体中文** | Grapery 基础会员与高级会员订阅，支持月付与年付。订阅自动续费，可在 App Store 账户设置中管理或取消。 |
| **描述 — 英文** | Grapery Basic and Premium memberships with monthly and yearly billing. Subscriptions auto-renew; manage or cancel in App Store account settings. |
| **订阅群组 ID**（ASC） | `22103260` |

---

## 3. 订阅等级排序（Subscription Levels）

在同一订阅组 **Grapery Membership** 内设置 **2 个等级**。

> **Apple 规则**：Level **1** = 组内**最高**档（权益最多）；数字越大档越低。ASC 编辑界面按「最高级别服务优先」自上而下排列。

| ASC Level | 包含产品 | 说明 |
|-----------|----------|------|
| **1** | 高级月付、高级年付 | 高档（Premium / Prime） |
| **2** | 基础月付、基础年付 | 低档（Basic / Pro） |

- Level 2 → Level 1：升级；Level 1 → Level 2：降级  
- 同档内月付 ↔ 年付：计费周期切换  

**当前 ASC 排列**（2026-05-21 已创建）：Premium Yearly → Premium Monthly → Basic Yearly → Basic Monthly（符合上述 Level 1/2 分组）。

---

## 3.1 ASC 本地化描述长度限制

App Store Connect **订阅描述字段上限为 55 个字符**（中英文均计）。§4 中的长文案用于 App 内展示；写入 ASC 时请使用下表 **ASC 短文案**（已填入 ASC）：

| Product ID | zh 显示名 | zh ASC 描述（≤55） | en 显示名 | en ASC 描述（≤55） |
|------------|-----------|-------------------|-----------|-------------------|
| `com.grapery.prime.yearly` | 高级会员 · 年付 | 高级会员年付，200点/月，约省20%，自动续费。 | Premium · Yearly | Premium yearly, 200 credits/mo, save ~20%. Auto-renews. |
| `com.grapery.prime.monthly` | 高级会员 · 月付 | 高级会员月付，200点/月，并行生成，自动续费。 | Premium · Monthly | Premium monthly, 200 credits/mo. Auto-renews. |
| `com.grapery.pro.yearly` | 基础会员 · 年付 | 基础会员年付，100点/月，约省20%，自动续费。 | Basic · Yearly | Basic yearly, 100 credits/mo, save ~20%. Auto-renews. |
| `com.grapery.pro.monthly` | 基础会员 · 月付 | 基础会员月付，100点/月，AI创作额度，自动续费。 | Basic · Monthly | Basic monthly, 100 AI credits/mo. Auto-renews. |

---

## 4. 在售订阅产品（4 个 SKU）

路径：**订阅组 Grapery Membership → 「+」创建订阅**

### 4.1 通用设置（每个产品相同）

| 设置项 | 值 |
|--------|-----|
| 类型 | 自动续期订阅 |
| 家庭共享 | **关闭**（代码：`FamilyShareable: false`） |
| 免费试用 | 暂不配置（可后续在 ASC 添加 Introductory Offer） |
| 推介促销 | 暂不配置 |
| 审核截图 | 会员/支付页截图 1 张（见 [§7](#7-提交审核前检查)） |

---

### 4.2 产品 1：基础会员 · 月付

| 字段 | 值 |
|------|-----|
| **参考名称** | `Basic Monthly` |
| **产品 ID** | `com.grapery.pro.monthly` |
| **后端 plan id** | `plan_pro_monthly` |
| **订阅时长** | 1 个月 |
| **订阅等级** | Level 2 |
| **价格（中国大陆）** | **¥29.90 / 月** |
| **Apple ID**（ASC） | `6771576767` |
| **每月 AI 点数（展示 credits）** | 100 |

**本地化 — 简体中文 (zh-Hans)**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | 基础会员 · 月付 |
| 描述 | 每月 100 点 AI 创作额度；AI 分镜与场景配图；故事与作品云端保存；导出配图到相册；可使用最新图像生成模型。订阅自动续费。 |

**本地化 — 英文 (en-US)**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | Basic · Monthly |
| 描述 | 100 AI credits per month for stories and storyboards; AI scene imagery; cloud storage; export to Photos; latest image models. Auto-renews. |

**权益 key（App 内展示，来自 `Localizable.strings`）**

- `membership_feature_basic_credits`
- `membership_feature_export_local`
- `membership_feature_share_unlimited`
- `membership_feature_keep_forever`
- `membership_feature_latest_models`
- `membership_feature_basic_serial_creation`

---

### 4.3 产品 2：基础会员 · 年付

| 字段 | 值 |
|------|-----|
| **参考名称** | `Basic Yearly` |
| **产品 ID** | `com.grapery.pro.yearly` |
| **后端 plan id** | `plan_pro_yearly` |
| **订阅时长** | 1 年 |
| **订阅等级** | Level 2 |
| **价格（中国大陆）** | **¥287.00 / 年**（≈ 月付 ×12 × 0.8，约省 20%） |
| **ASC 价格档位** | **¥288.00 / 年**（Apple 无 ¥287.00 档位，已选最近档位；若需账单与 seed 严格一致，请将 `membership_iap_seed.sql` 改为 `288.00`） |
| **Apple ID**（ASC） | `6771575886` |
| **每月 AI 点数** | 100（按账期重置，不结转） |

**本地化 — 简体中文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | 基础会员 · 年付 |
| 描述 | 按年订阅基础会员，每月 100 点 AI 创作额度及全部基础权益。相较月付约省 20%。订阅自动续费。 |

**本地化 — 英文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | Basic · Yearly |
| 描述 | Annual Basic membership with 100 AI credits per month and all Basic benefits. Save ~20% vs monthly. Auto-renews. |

权益 key：同 [§4.2](#42-产品-1基础会员--月付)

---

### 4.4 产品 3：高级会员 · 月付

| 字段 | 值 |
|------|-----|
| **参考名称** | `Premium Monthly` |
| **产品 ID** | `com.grapery.prime.monthly` |
| **后端 plan id** | `plan_prime_monthly` |
| **订阅时长** | 1 个月 |
| **订阅等级** | Level 1 |
| **价格（中国大陆）** | **¥49.90 / 月** |
| **Apple ID**（ASC） | `6771575566` |
| **每月 AI 点数** | 200 |

**本地化 — 简体中文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | 高级会员 · 月付 |
| 描述 | 每月 200 点 AI 创作额度；分镜/碎片支持并行 AI 生成；AI 队列更高优先级；含基础会员全部权益。订阅自动续费。 |

**本地化 — 英文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | Premium · Monthly |
| 描述 | 200 AI credits per month; parallel AI generation for storyboards; higher queue priority; all Basic benefits included. Auto-renews. |

**权益 key**

- `membership_feature_premium_credits`
- `membership_feature_export_local`
- `membership_feature_share_unlimited`
- `membership_feature_keep_forever`
- `membership_feature_latest_models`
- `membership_feature_premium_parallel_creation`
- `membership_feature_premium_higher_priority`

---

### 4.5 产品 4：高级会员 · 年付

| 字段 | 值 |
|------|-----|
| **参考名称** | `Premium Yearly` |
| **产品 ID** | `com.grapery.prime.yearly` |
| **后端 plan id** | `plan_prime_yearly` |
| **订阅时长** | 1 年 |
| **订阅等级** | Level 1 |
| **价格（中国大陆）** | **¥479.00 / 年**（≈ 月付 ×12 × 0.8） |
| **Apple ID**（ASC） | `6771574421` |
| **每月 AI 点数** | 200 |

**本地化 — 简体中文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | 高级会员 · 年付 |
| 描述 | 按年订阅高级会员，每月 200 点 AI 创作额度及全部高级权益。相较月付约省 20%。订阅自动续费。 |

**本地化 — 英文**

| 字段 | 文案 |
|------|------|
| 订阅显示名称 | Premium · Yearly |
| 描述 | Annual Premium membership with 200 AI credits per month and all Premium benefits. Save ~20% vs monthly. Auto-renews. |

权益 key：同 [§4.4](#44-产品-3高级会员--月付)

---

## 5. 价格与 SKU 速查表

| Product ID | 档位 | 周期 | CNY | ASC 档位 | credits/月 | Level | Apple ID | 后端 plan id | ASC 参考名称 |
|------------|------|------|-----|----------|------------|-------|----------|--------------|--------------|
| `com.grapery.pro.monthly` | 基础 | 月 | ¥29.90 | ¥29.90 | 100 | 2 | `6771576767` | `plan_pro_monthly` | Basic Monthly |
| `com.grapery.pro.yearly` | 基础 | 年 | ¥287.00 | **¥288.00** | 100 | 2 | `6771575886` | `plan_pro_yearly` | Basic Yearly |
| `com.grapery.prime.monthly` | 高级 | 月 | ¥49.90 | ¥49.90 | 200 | 1 | `6771575566` | `plan_prime_monthly` | Premium Monthly |
| `com.grapery.prime.yearly` | 高级 | 年 | ¥479.00 | ¥479.00 | 200 | 1 | `6771574421` | `plan_prime_yearly` | Premium Yearly |

**定价说明**（来自 `membership_iap_seed.sql` 注释）：

- 年付 ≈ 12 × 月价 × 0.8（约省 20%）
- 1 展示 credit ≈ 1 张 AI 配图量级（内部 token 换算见 `common.CreditToTokenRatio`）

若 App Store 价格档位无精确匹配，选 **最接近** 的 Tier；**中国区以本表 CNY 为准**。

---

## 6. 不在 App Store 创建的产品（已下架）

以下 SKU 存在于历史 seed 中但 `is_active = 0`，**当前 App 未接入，勿在 ASC 新建**：

| Product ID | 说明 |
|------------|------|
| `com.grapery.pro.quarterly` | 基础季付 |
| `com.grapery.prime.quarterly` | 高级季付 |
| `com.grapery.ultra.monthly` | Ultra 月付 |
| `com.grapery.ultra.quarterly` | Ultra 季付 |
| `com.grapery.ultra.yearly` | Ultra 年付 |

---

## 7. 提交审核前检查

### 7.1 每个订阅的「App 审核信息」

**审核备注（可复制）**：

```text
本 App 提供自动续期订阅会员，用于解锁 AI 故事创作额度与高级功能。
测试步骤：登录 → 设置/会员 → 选择套餐 → 使用 Sandbox 账号完成购买。
Sandbox 测试账号：[在此填写 Sandbox 测试邮箱]
```

- **截图**：App 内「会员 / 支付」页，需可见套餐名称、价格与权益列表

### 7.2 App 版本关联

- **App 信息 → App 内购买项目**：将上述 4 个订阅全部关联到待提交版本
- **App 隐私**：已说明购买/订阅相关数据用途
- **订阅披露**：App 内须说明自动续费、取消方式（设置页 / 会员页文案）

### 7.3 后端与 Webhook（IAP 创建后必做）

| 检查项 | 预期值 |
|--------|--------|
| Product ID | 与 [§5](#5-价格与-sku-速查表) 四行完全一致 |
| `APPLE_BUNDLE_ID` | `com.rankquantity.voyager` |
| Server Notifications **V2** URL（Production + Sandbox） | `https://rankquantity.xyz/api/vippay/iap/apple/notification` |
| App Store Connect API Key（`.p8`） | 已配置 `APPLE_ISSUER_ID` / `APPLE_KEY_ID` / `APPLE_PRIVATE_KEY` |
| 数据库 | 已执行 `grapery/scripts/membership_iap_seed.sql` |
| VipPay 商品表 | 启动后 `iap_products` 含 4 行（或手动对齐 seed） |
| JWT | VipPay `JWT_SECRET` 与主 API 一致 |

详见 [iap_config_requirements.md](./iap_config_requirements.md)、[CONFIG.md](../cmd/vippay/CONFIG.md)。

---

## 8. 配置完成自检清单

```text
订阅组
  [x] Grapery Membership 已创建（群组 ID 22103260，中英文名称 + 描述）

4 个 SKU
  [x] com.grapery.pro.monthly     — ¥29.90/月  — Level 2 — 中英文本地化 — Apple ID 6771576767
  [x] com.grapery.pro.yearly      — ¥288/年*   — Level 2 — 中英文本地化 — Apple ID 6771575886
  [x] com.grapery.prime.monthly   — ¥49.90/月  — Level 1 — 中英文本地化 — Apple ID 6771575566
  [x] com.grapery.prime.yearly    — ¥479/年    — Level 1 — 中英文本地化 — Apple ID 6771574421

  * 后端 seed 为 ¥287；ASC 无 ¥287 档位，已用 ¥288

每个 SKU
  [x] 家庭共享 = 关闭
  [x] 价格 + 全球供应范围已设置
  [ ] 审核截图 + Sandbox 说明已填（状态仍为「元数据丢失」直至截图上传）
  [ ] 状态「准备提交」或已随 App 版本提交

App 版本
  [ ] 4 个订阅已关联到待审版本

代码对齐
  [ ] grapery.storekit Product ID 一致
  [ ] membership_iap_seed.sql / iap_grapery_seed.go 价格一致

后端
  [ ] VipPay prod 已部署，/api/vippay/health 正常
  [ ] Server Notifications URL 已在 ASC 配置
  [ ] Sandbox 购买 → verify 成功 → vip/info is_vip=true
```

---

## 9. Sandbox 验证步骤

1. **App Store Connect → 用户和访问 → Sandbox → 测试账户**：创建 Sandbox 账号  
2. **iPhone：设置 → App Store → 沙盒账户**：登录 Sandbox 账号（勿与生产 Apple ID 混用）  
3. **Xcode**：Scheme 可选用 `grapery.storekit` 做本地 StoreKit 测试（无需 ASC 商品「已批准」）  
4. **真机 Sandbox 购买**：App → 会员页 → 购买 → 确认 VipPay 日志  
5. **验证接口**：
   - `POST /api/vippay/iap/apple/verify` 返回 success  
   - `GET /api/vippay/vip/info` → `is_vip: true`，`credit_limit` 与档位一致  

**本地联调环境变量（iOS DEBUG）**：

```bash
API_BASE_URL=http://127.0.0.1:8080
PAYMENT_API_BASE_URL=http://127.0.0.1:8060   # 或实际 VipPay 端口（Docker 默认 8081）
```

---

## 10. 变更记录

| 日期 | 变更 | 操作人 |
|------|------|--------|
| 2026-05-21 | 初版：4 个在售 SKU、价格、ASC 文案与代码对齐说明 | — |
| 2026-05-21 | ASC 自动化配置：群组 `22103260`；4 SKU 已创建并设价/供应范围；补充 55 字描述限制、Apple ID、Level 1=Premium 说明；Basic 年付 ASC 档位 ¥288 | — |

<!-- 后续变更请在此表追加一行，并同步更新正文 §4–§5 -->
