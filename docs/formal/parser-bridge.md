# Parser Formal Bridge Rules

本文档定义第六步第二小步的桥接规则：把 `Parser.tla` 产出的抽象 case 转成真实 Go parser 可执行的 test case。

本文件不是新的契约源头。唯一形式化模型源头仍然是：

```text
formal/parser/Parser.tla
```

桥接只负责翻译：

```text
Parser.tla selectedCase
  -> concrete argv
  -> Go flag fixture
  -> expected Go observable
```

## 1. Inputs

TLC case 枚举命令：

```bash
make formal-parser-cases TLA2TOOLS_JAR=/private/tmp/tla2tools/tla2tools.jar
```

输入产物：

```text
formal/parser/generated/abstract_cases.dump
```

每个 dump state 都有一个 `selectedCase`：

```tla
[
  stateName |-> "...",
  tokenRaw |-> "...",
  tokenClass |-> [
    splitterClass |-> "...",
    prefixShape |-> "...",
    detectorResult |-> "..."
  ],
  tokenFlag |-> "...",
  tokenValue |-> "...",
  tokenParseError |-> [class |-> "...", text |-> "..."],
  allowedDashValueEnhancement |-> TRUE_or_FALSE,
  oldObservable |-> ...,
  newObservable |-> ...
]
```

`oldObservable` 和 `newObservable` 都由 `Parser.tla` 计算得出。Go test 的 expected 默认使用 `newObservable`。

## 2. State Bridge

`stateName` 表示读取当前 token 之前的 parser 抽象状态。Go test 通过 argv 前缀把真实 parser 带到同一个状态。

| `stateName` | Go argv prefix | 进入当前 token 前必须满足 |
|---|---|---|
| `init` | `[]` | `current=0`, `currentFlag=nil` |
| `pending_PortRange` | `["--PortRange"]` | `current=1`, `currentFlag=PortRange`, `PortRange.assigned=true`, `PortRange.value=""` |
| `pending_Count` | `["--Count"]` | `current=1`, `currentFlag=Count`, `Count.assigned=true`, `Count.value=""` |
| `pending_Name` | `["--Name"]` | `current=1`, `currentFlag=Name`, `Name.assigned=true`, `Name.value=""` |
| `pending_RegionId` | `["--RegionId"]` | `current=1`, `currentFlag=RegionId`, `RegionId.assigned=true`, `RegionId.value=""` |
| `pending_short_Count` | `["-c"]` | `current=1`, `currentFlag=Count`, `Count.assigned=true`, `Count.value=""`, `pendingFrom=short_flag_candidate` |
| `pending_dynamic_long_dynamic` | `["--dynamic"]` | `current=1`, `currentFlag=dynamic`, `dynamic.assigned=true`, `dynamic.value=""`, `pendingFrom=dynamic_unknown_flag` |
| `pending_dynamic_short_u` | `["-u"]` | `current=1`, `currentFlag=u`, `u.assigned=true`, `u.value=""`, `pendingFrom=dynamic_unknown_flag` |

桥接生成器必须失败即停：

```text
unknown stateName -> fail
prefix read returns error -> fail
prefix does not create expected pending flag -> fail
prefix current does not equal expected prefix length -> fail
```

## 3. Token Bridge

当前 token 直接使用 `selectedCase.tokenRaw`。

```text
argv = statePrefix(selectedCase.stateName) + [selectedCase.tokenRaw]
```

示例：

| abstract case | concrete argv |
|---|---|
| `stateName=init`, `tokenRaw="-1/-1"` | `["-1/-1"]` |
| `stateName=pending_PortRange`, `tokenRaw="-1/-1"` | `["--PortRange", "-1/-1"]` |
| `stateName=pending_PortRange`, `tokenRaw="--RegionId"` | `["--PortRange", "--RegionId"]` |
| `stateName=pending_Count`, `tokenRaw="-abc:value"` | `["--Count", "-abc:value"]` |

生成器不能重新解释 token 文本。`tokenRaw` 的分类和 expected 已经由 `Parser.tla` 决定。

## 4. Flag Fixture

Go test 使用固定 flag registry，必须和 `Parser.tla` 的 `FlagNames` / `Mode` 对齐。

| flag | Go `AssignedMode` | shorthand | TLA mode |
|---|---:|---|---|
| `PortRange` | `AssignedOnce` | none | `AssignedOnce` |
| `Count` | `AssignedOnce` | `c` | `AssignedOnce` |
| `Name` | `AssignedOnce` | none | `AssignedOnce` |
| `RegionId` | `AssignedOnce` | none | `AssignedOnce` |
| `help` | `AssignedNone` | `h` | `AssignedNone` |
| `dynamic` | dynamic unknown flag | none | `AssignedDefault` |
| `u` | dynamic unknown shorthand | `u` | `AssignedDefault` |

Go fixture rules:

```text
ctx := NewCommandContext(...)
ctx.Flags().Add(&Flag{Name: "PortRange", AssignedMode: AssignedOnce})
ctx.Flags().Add(&Flag{Name: "Count", Shorthand: 'c', AssignedMode: AssignedOnce})
ctx.Flags().Add(&Flag{Name: "Name", AssignedMode: AssignedOnce})
ctx.Flags().Add(&Flag{Name: "RegionId", AssignedMode: AssignedOnce})
ctx.Flags().Add(&Flag{Name: "help", Shorthand: 'h', AssignedMode: AssignedNone})
parser := NewParser(argv, ctx)
parser.SetAllowUnknown(false)
```

Unknown flag fixture rules:

- 如果 `stateName` 是 `pending_dynamic_long_dynamic` 或 `pending_dynamic_short_u`，必须启用 `ctx.SetUnknownFlags(NewFlagSet())` 和 `ctx.command.EnableUnknownFlag=true`，否则 Go parser 无法进入该 pending state。
- 如果当前 `tokenClass.detectorResult == dynamic_unknown_flag`，也必须启用 unknown flag fixture，确保 token detector 结果和 `Parser.tla` 一致。
- 如果当前 `tokenClass.detectorResult == detector_error`，不能启用 unknown flag fixture；否则真实 Go detector 会把 `--unknown` / `-z` 动态加入 unknown flag set，和该 token class 不一致。
- `Parser.tla` 使用 `TokenApplicable` 过滤掉“动态 unknown pending state + detector_error token”这种真实 fixture 无法构造的组合。

## 5. Execution Bridge

Go test runner 应运行在 `package cli`，这样可以观察 parser/flag 的内部状态：

```text
Parser.current
Parser.currentFlag
Parser.currentFlagOrigin
Flag.assigned
Flag.value
Flag.values
```

执行步骤：

```text
1. 构造 argv = prefix + [tokenRaw]
2. 构造 flag fixture
3. 调用 readNext() 消费 prefix 中的每个 token
4. 校验 prefix 后 parser 进入 stateName 对应状态
5. 再调用 readNext() 读取 tokenRaw
6. 捕获 actual Go observable
7. actual Go observable == selectedCase.newObservable
```

不能用 Go 实现结果反推 expected。expected 只能来自 `selectedCase.newObservable`。

## 6. Observable Bridge

Go actual observable 和 TLA `Observable` 字段一一对应：

| TLA field | Go capture |
|---|---|
| `current` | `parser.GetCurrent()` |
| `outArg` | `arg` returned by `readNext()` |
| `outFlag` | returned `flag.Name`, or `NoFlag` when nil |
| `outMore` | `more` returned by `readNext()` |
| `err.class` | classify by exact `err.Error()` text |
| `err.text` | `err.Error()`, or `""` when nil |
| `flags[f].assigned` | `flag.IsAssigned()` |
| `flags[f].value` | internal `flag.value` |
| `flags[f].values` | internal `flag.values`, normalized nil -> empty array |
| `pendingFlag` | `parser.currentFlag.Name`, or `NoFlag` when nil |
| `pendingFrom` | `parser.currentFlagOrigin` |

错误分类必须基于精确文案：

| text shape | class |
|---|---|
| `""` | `NoError` |
| `--{flag} must be assigned with value` | `MissingValue` |
| `--{flag} duplicated` | `DuplicateFlag` |
| `flag duplicated --{flag}` | `FlagSetDuplicate` |
| `not support '--' in command line` | `InvalidDoubleDash` |
| `not support flag form {prefix}` | `InvalidFlagForm` |
| `invalid flag {name}` | `InvalidFlag` |
| `unknown flag -{ch}` | `UnknownShortFlag` |
| `flag --{flag} can't be assiged` | `AssignToNoValueFlag` |

错误文案是契约保护内容，拼写也必须精确匹配。

## 7. `pendingFrom` Bridge

`Parser.tla` 的 `pendingFrom` 是契约内部观察点，用来保证增强只作用于正常 `--long` flag 产生的 pending state。

Go `Parser` 已增加内部 `currentFlagOrigin` 字段，contract test runner 直接读取真实 parser 状态：

```text
actual.pendingFrom = parser.currentFlagOrigin
```

因此 `pendingFrom` 不是从 test case 反推的字段，而是真实 Go 实现的内部可观察状态。

## 8. Coverage Requirements

桥接生成器必须输出覆盖矩阵，并校验：

```text
abstract case count == TLC dump case count
stateName coverage == NamedStates
tokenRaw coverage == Tokens after TokenApplicable filtering
each TokenApplicable stateName x tokenRaw pair has exactly one Go case
allowedDashValueEnhancement=true cases are preserved
allowedDashValueEnhancement=false cases use newObservable equal to oldObservable
```

当前模型期望：

```text
8 state classes
87 concrete token representatives
674 TokenApplicable abstract cases
16 allowed dash-leading enhancement cases
```

若 `Parser.tla` 中 `NamedStates`、`Tokens` 或 `TokenApplicable` 变化，上述数字必须随 TLC dump 自动更新，不能手工硬编码成永久常量。

## 9. Generated Artifacts

生成器产出：

```text
cli/testdata/parser_contract_cases.json
docs/formal/parser-case-coverage.md
```

Go contract runner：

```text
cli/parser_contract_test.go
```

该 runner 使用 `parser_contract` build tag，普通 `go test ./cli` 不会运行它。实现阶段必须显式运行：

```bash
make parser-contract-test
```

JSON 中每个 case 包含：

```json
{
  "name": "pending_PortRange__dash_value_-1/-1",
  "stateName": "pending_PortRange",
  "argv": ["--PortRange", "-1/-1"],
  "tokenRaw": "-1/-1",
  "allowedDashValueEnhancement": true,
  "expected": {
    "source": "selectedCase.newObservable"
  }
}
```
