# Canonical Response Schema Contract

## 1. 职责

本分册只负责把 Canonical API JSON 中的开放元数据 `responses`、`components` 变成 Help 可消费的响应文档。它不解析 CLI Flag、不输出 Help、不生成 Recovery。

## 2. 输入契约

Canonical 生产侧在每个 API JSON 中直接携带开放元数据的：

```json
{
  "responses": {},
  "components": {
    "schemas": {}
  }
}
```

生产侧只做既定的多语言规范化：Schema 任意层级的 `title`、`description` 扩展为 `title_en`、`title_zh`、`description_en`、`description_zh`。CLI 不再次调用翻译或 Meta 服务。

`canonicalmeta.API` 必须以惰性原始 JSON 形式持有这两个字段，使用 `json.RawMessage` 或等价的延迟解码类型。普通 API 执行不得为了未使用的 Response Help 把完整 Schema 解码成 `map[string]any`。

## 3. 输出模型

本模块向 Help 层提供一个与渲染格式无关的模型：

```text
ResponseSchemaDocument
  statusCode   string
  contentType  string (optional)
  schema       JSON node
  components   reachable components.schemas only
  warnings     []string (optional)
```

`schema` 保留原始 `$ref`。`components` 只包含从选中响应 Schema 可达的 `components.schemas`，不得复制整个 API 的 Components。

## 4. 成功响应选择

按以下固定顺序选一个响应：

1. 状态码 `200`；
2. 数值最小的其他 `2xx`；
3. `default`；
4. 都不存在时返回“无可用响应 Schema”，不选择 `4xx/5xx`。

选中响应后只读取响应 Body Schema，不展示 Response Header。

Content Type 按以下顺序选择第一个包含 Schema 的媒体类型：

1. `application/json`；
2. `application/*+json`；
3. 其他媒体类型按字典序；
4. 兼容响应对象直接包含 `schema` 的旧形态。

响应存在但没有 Body Schema 时，视为“无可用响应 Schema”，不把 Header 当作输出 Schema。

## 5. Ref 处理

- 只解析本地 `#/components/schemas/<name>` 引用以计算依赖闭包。
- 输出中的 `$ref` 原样保留，不内联展开。
- 一个 Component 引用另一个 Component 时递归加入闭包。
- 使用 `visited` 集合防止自引用或循环引用导致无限递归。
- 找不到 Ref 时保留原 `$ref`，在 `warnings` 中记录；不访问网络，不让整个 Help 崩溃。
- 非本地 Ref 原样保留并记录 Warning。

## 6. 异常与降级

| 输入情况 | 结果 |
| --- | --- |
| 没有 `responses` | 返回无 Schema 状态 |
| 没有成功响应 | 返回无 Schema状态 |
| 成功响应没有 Body Schema | 返回无 Schema 状态 |
| 没有 `components` | Schema 正常输出 |
| Ref 缺失或循环 | 保留 Ref，输出可用部分和 Warning |
| `responses/components` JSON 结构损坏 | 返回类型化的本地 Meta 读取错误，不 Panic |

## 7. 性能要求

- 产品 Help 和 API 搜索不得加载全部 API Response Schema。
- 仅进入一个 API Help 时读取该 API 文件。
- 仅 Request Help 需要 Array Query Example 时，以及显式 Response Help 时，才解析该 API 的 `responses/components`。
- Packed Meta 仍只随机读取一个 API frame，不全量解压。

## 8. 单元测试

至少覆盖：

- `200`、多个 `2xx`、`default` 和无成功响应；
- JSON、`+json`、其他 Content Type 和直接 `schema`；
- Inline Schema；
- 单层 Ref、多层 Ref、自引用、双向循环和缺失 Ref；
- Components 裁剪只包含可达节点；
- Response 只有 Header；
- 惰性解析：普通产品索引读取不解析 Response。
