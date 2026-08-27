# Help Command Surface and Section Rendering

## 1. 职责

本分册独占 Help Provider 路由、新 Flag 解析、Section 选择和文本/JSON 外层文档组装。搜索算法、Array Path 算法、AI Mode 判断和错误 Recovery 分别由其他分册提供。

## 2. 命令语法

```bash
aliyun help ecs DescribeInstances
aliyun help ecs DescribeInstances --cli-section request
aliyun help ecs DescribeInstances --cli-section response
aliyun help ecs DescribeInstances --cli-section response --format json
```

所有新增 Flag：

| Flag | 值 | 适用目标 |
| --- | --- | --- |
| `--cli-section` | `request` / `response` | 必须指定 API |
| `--cli-search` | 非空关键词 | Root、Product、API Request/Response |
| `--cli-all` | 无值 | Root、Product；API Section 不接受 |

非法值、Response 未指定 API、API Section 携带 `--cli-all` 时返回明确的本地选项错误，不静默忽略。

`--cli-search` 与 `--cli-all` 同时用于 Root/Product 时，搜索语义优先且返回全部搜索结果；`--cli-all` 不额外改变结果。

## 3. Provider 路由

文本 Help 必须先判断已安装插件，再决定是否进入新 Canonical Help：

1. 已安装 Go 插件：按现状转发给插件进程；
2. 已安装 Meta 插件：按现状由 Runtime 读取插件 Meta；
3. 未安装插件且内置 Canonical 可用：进入本设计的新 Help；
4. Canonical 不存在：保持现有未知产品或插件安装提示。

新 Canonical Help 不依赖远程插件索引是否可用。

Machine JSON Help 保持当前 Unify 路由：`--format json` 和 `--help=json` 始终进入宿主的稳定 Machine Help Service，读取内置 Canonical；即使产品插件已安装，也不委托插件生成 JSON。Section、Search、Listing 和 AI Mode Hint 都扩展在这条既有宿主路径上。

## 4. Request Section

- `request` 是默认值，原有 `aliyun help <product> <api>` 文本行为保持兼容。
- AI Mode 和非 AI Mode 都完整展示参数，不截断。
- PascalCase 命令使用兼容 V1 参数视图；kebab 命令使用 Canonical 参数视图。
- `--cli-search` 由搜索模块过滤当前参数视图及全局参数。
- Array Response Query Example 和 AI Mode Hint 通过各自模块提供的可选扩展追加。

## 5. Response Section

文本输出使用稳定、可复制的 JSON Schema 表达，不把 Schema 改写成人类描述表格：

```text
Response Schema (HTTP 200, application/json):
{
  "type": "object",
  "properties": {
    "Instances": {"$ref": "#/components/schemas/Instances"}
  }
}

Components:
{
  "schemas": {
    "Instances": {"type": "object"}
  }
}
```

- 不内联 Ref，不展示 Response Header。
- 无 Schema 时退出码为 0，并明确输出 `No response schema is available for this API.`。
- `--cli-search` 输出命中字段的完整路径和最小父级 Schema；依赖 Components 重新按过滤后 Schema 计算。
- Request/Response 都完整展示，和 `--cli-all` 无关。

## 6. JSON Help

`--format json` 是渲染格式，不与 Section 冲突。沿用现有 Machine Help 外层身份字段，并增加 `section`。

本文取代 [Agentic Help JSON v1 Design](2026-08-18-agentic-help-json-v1-design.md) 中“`outputSchema`、`pagination`、`risk`、`recovery` 固定输出 `null`”的约定。尚无实际值的可选字段改为省略；稳定外层身份、版本选择和 DTO 投影原则继续保留。

Request Section 保留现有请求字段；没有实际值的可选字段省略，不再输出 `null`、空字符串、空数组或空对象。

- 未搜索的 Request JSON 继续输出 `parameterSets.camel`、`parameterSets.kebab` 和 `activeParameterSet`，保持现有 v1 契约。
- 使用 `--cli-search` 时只返回 `activeParameterSet` 对应的命中参数，省略未激活参数集合，避免同一参数以两种风格重复出现。

Response Section 示例：

```json
{
  "schemaVersion": "v1",
  "kind": "api",
  "section": "response",
  "target": {
    "path": ["aliyun", "ecs", "DescribeInstances"],
    "requestedStyle": "camel"
  },
  "product": {
    "code": "ecs",
    "selectedVersion": "2014-05-26"
  },
  "api": {
    "name": "DescribeInstances",
    "cmdName": "describe-instances"
  },
  "outputSchema": {
    "statusCode": "200",
    "contentType": "application/json",
    "schema": {
      "type": "object",
      "properties": {
        "Instances": {
          "$ref": "#/components/schemas/Instances"
        }
      }
    },
    "components": {
      "schemas": {
        "Instances": {
          "type": "object"
        }
      }
    }
  }
}
```

规则：

- Request Section 不输出空的 `outputSchema`。
- Response Section 不输出 Request 参数集合。
- 没有可用 Schema 时省略 `outputSchema`，增加非空 `notice`。
- `warnings`、`components`、`responseQueryExample` 和 `aiModeHint` 仅有实际内容时输出。
- JSON 前后不得拼接文本。
- Machine Help 已有的结构化错误协议保持合法 JSON，不纳入本地 Agent Error Envelope。

## 7. 风格与版本

- API 输入全小写时按 kebab 风格解析；其他情况按 PascalCase 风格解析，沿用现有规则。
- 生成的 Help 命令、搜索结果和示例必须使用相同风格。
- 显式 `--api-version` 必须贯穿 Product/API Help、Search 和 Recovery。
- 未显式指定版本时，PascalCase 使用 Legacy 默认版本，kebab 使用插件默认版本，保持当前 Unify 行为。

## 8. 兼容入口

- `aliyun <product> <api> --help` 继续映射到 Request Section。
- `--help=json` 继续映射到 `aliyun help ... --format json` 的同一构建路径。
- 新 Flag 必须在兼容入口中被正确传递，不得被当成未知 API 参数或静默丢弃。

## 9. 单元测试

- 文本 Provider 路由覆盖无插件、Meta 插件和 Go 插件；Machine JSON 在三种情况下都使用宿主 Canonical；
- Section 默认值、非法值和缺少 API；
- Request/Response × Text/JSON × Camel/Kebab；
- 显式版本；
- 无 Response Schema；
- JSON 无空可选字段且可被标准 JSON Parser 解析；
- 旧 `--help` / `--help=json` 入口兼容。
