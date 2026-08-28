# Help Search 契约

## 1. 统一搜索流水线

所有宿主 Search 共用一个引擎：

```text
按 HelpTarget 收集全部候选
  -> query/name/alias 统一归一化与分词
  -> 标识符匹配优先，文档匹配次之
  -> 全量稳定排序
  -> 记录真实 total
  -> 渲染前 20 条完整结果
```

内部常量 `helpSearchResultLimit = 20` 作用于 Root/Product/Action/Parameter/Response、AI/非 AI、Text/JSON。它与默认 Help 截断开关正交，`--help-all` 不解除 Search 上限。

合法零命中 exit 0；空 query 是本地语法错误 exit 2。超过 20 条时提示使用更具体 query。

## 2. 分词

Query、主名称、alias 使用同一 tokenizer，处理 CamelCase、连续大写、kebab、snake、点路径、普通分隔符和数字边界；中文保留原文做子串匹配。

```text
QueryMonthlyBill     -> query / monthly / bill
query-monthly-bill   -> query / monthly / bill
InstanceID           -> instance / id
--instance-ids       -> instance / ids
```

一次普通 Search 使用完整 Token 序列，不自动改成“任意 Token 命中”的宽泛 OR。Recovery 可以按信息量逐个尝试子组合，但每个 query 必须真实验证。

## 3. 排名

优先级固定：

1. 当前风格主名称或 alias 精确匹配；
2. 标识符 Token 序列/前缀匹配；
3. 标识符归一化后包含；
4. Title、Description、Help、约束、示例等文档字段匹配。

标识符匹配始终高于文档匹配。同级按当前风格展示名称稳定排序，Text/JSON 共用同一结果对象。

## 4. 各层候选集

| Target | 候选 | 标识符 | 文档 | 命中命令 |
| --- | --- | --- | --- | --- |
| Root | Core、Utility、Extension、Product | 名称、alias、product code | 本地中英文短描述 | 对象后置 Help |
| Product | 选定版本 Action | API name、`cmd_name` | `title_*`、`description_*` | Action 后置 Help |
| Action/Request | 当前风格业务参数、公开全局参数 | Flag、raw name、alias、路径 | Help、约束、枚举、示例 | L3 参数 Help |
| Parameter | 当前参数及全部嵌套字段 | Flag、raw name、字段路径 | Help、约束、枚举、示例 | 保持参数 Target |
| Response | Response 字段与相关 Components | JSON Path、Schema 字段 | `title_*`、`description_*` | 保持前置 Response Section |

Root 不跨入产品 Action 或 Extension 内部子命令。Utility 只返回规范路径，例如 `aliyun utils mcp-proxy --help`，不重复返回旧兼容入口。

## 5. 结果模型

Search Document 至少包含 `kind`、`query`、类型化 `matches` 和：

```json
{
  "result": {
    "shown": 20,
    "total": 63,
    "truncated": true
  }
}
```

Text 与 JSON 的 matches、shown、total、排序必须一致。单个 match 的 `command` 使用 01 分册统一 command builder。

## 6. Recovery 验证接口

Recovery 不自行实现匹配。它向统一 Search 服务传入完整 `HelpTarget + query`，只有 `total > 0` 才能发布 Search 命令。

候选顺序：高可信 `did_you_mean` → 错误名称完整 Token 序列 → 有区分度的 Token 子组合 → 单 Token。全部无命中时退回当前层普通 `--help`。

验证必须复用实际 Provider、版本、风格和候选集。不得从错误字符串猜另一个产品/Action，也不得生成插件不支持的宿主 Search。

## 7. 主要验收

- 同一概念的 Camel/kebab/snake 查询命中相同对象，显示名跟随当前风格。
- 名称命中稳定排在仅文档命中之前。
- 先全量排名再取 20，total 是真实命中数。
- Root 搜到 Core/Utils/Extension/Product，且不联网。
- 每个 Recovery Search 命令实际执行后至少有一个结果。
