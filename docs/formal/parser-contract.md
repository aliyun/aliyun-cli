# Parser Formal Contract Draft

本文档定义 `cli` parser 的形式化契约草案，当前目标是支持一类增强：

```text
当一个已注册的、允许带参数的正常 long flag 正在等待 value 时，
如果下一个 token 是以 "-" 开头、但不是已识别 flag 的普通 token，
则 parser 可以把这个 token 当作该 flag 的 value。
```

典型例子：

```text
--PortRange -1/-1
```

这个增强不是 `PortRange` 或 `-1/-1` 的特例，而是一类受约束的 token 行为增强。

## 1. Proof Boundary

本契约只覆盖 parser 层：

- `Parser.ReadNextArg`
- `Parser.ReadAll`
- `Parser.readNext`
- `Parser.parseCommandArg`
- `SplitStringWithPrefix`
- `Flag.setIsAssigned`
- `Flag.needValue`
- `Flag.validate`
- `Flag.assign`

本契约不覆盖：

- shell 展开
- command dispatch
- OpenAPI request 构造
- metadata 加载与解析
- 云 API 返回
- i18n/help 输出
- parser 外层错误包装
- `Parser.SetAllowUnknown(true)` 下的完整状态空间

## 2. Token Class

原始 argv token 是无限集合。契约先把它抽象成有限 token class。

### 2.1 Splitter Class

由 `SplitStringWithPrefix(s, "=:")` 决定，只看第一个 `=` 或 `:`。

```yaml
splitter_class:
  - none
  - equal_nonempty
  - equal_empty
  - colon_nonempty
  - colon_empty
```

含义：

| class | 示例 | 说明 |
|---|---|---|
| `none` | `--PortRange`, `-1/-1`, `abc` | token 中没有 `=` 或 `:` |
| `equal_nonempty` | `--RegionId=cn-hangzhou` | 第一个 `=` 右侧非空 |
| `equal_empty` | `--RegionId=` | 第一个 `=` 右侧为空 |
| `colon_nonempty` | `--RegionId:cn-hangzhou`, `:value` | 第一个 `:` 右侧非空 |
| `colon_empty` | `--RegionId:` | 第一个 `:` 右侧为空 |

`equal_empty` / `colon_empty` 必须单独建模，因为 parser 使用 `value == ""` 判断 flag 是否仍然需要后续 value。

### 2.2 Prefix Shape

`prefix` 是第一个 `=` 或 `:` 左边的部分。分类顺序必须匹配 `parseCommandArg` 的源码分支。

```yaml
prefix_shape:
  - empty_prefix
  - non_dash_prefix
  - triple_dash_prefix
  - bare_double_dash
  - long_flag_candidate
  - bare_single_dash
  - short_flag_candidate
  - single_dash_multi
```

含义：

| class | 示例 | 当前 parser 含义 |
|---|---|---|
| `empty_prefix` | `""`, `=v`, `:v`, `=`, `:` | prefix 为空，不按 flag 处理 |
| `non_dash_prefix` | `abc`, `abc=1` | 普通 value / arg |
| `triple_dash_prefix` | `---`, `---cert`, `---cert=1` | 整个原 token 当 value |
| `bare_double_dash` | `--`, `--=x`, `--:x` | 固定错误：`not support '--' in command line` |
| `long_flag_candidate` | `--RegionId`, `--RegionId=cn-hangzhou` | 调用 long flag detector |
| `bare_single_dash` | `-`, `-=x`, `-:x` | 不是合法 short flag；旧行为必须保持 |
| `short_flag_candidate` | `-h`, `-r=value` | 调用 shorthand detector |
| `single_dash_multi` | `-abc`, `-1/-1`, `-abc=value` | 单横线后超过一个字符，不调用 detector |

### 2.3 Detector Result

只适用于：

- `long_flag_candidate`
- `short_flag_candidate`

```yaml
detector_result:
  - known_flag
  - dynamic_unknown_flag
  - detector_error
  - not_applicable
```

`detector_result` 不是词法形状，而是 registry / detector 查询结果。

## 3. Parser State Model

形式化状态最少包含：

```yaml
ParserState:
  current: Int
  currentFlag: NoFlag | FlagName
  currentFlagOrigin: none | long_flag_candidate | short_flag_candidate | dynamic_unknown_flag
  assigned:
    FlagName: Bool
  value:
    FlagName: String
  values:
    FlagName: Seq[String]
  outArg: String
  outFlag: NoFlag | FlagName
  outMore: Bool
  err:
    class: String
    text: String
```

`currentFlagOrigin` 用来区分 pending flag 是从什么 token 进入的。本次增强只针对正常 `--long` flag，因此模型必须知道 pending flag 是否来自 `long_flag_candidate`。

## 4. Observable

`Observable` 是契约保护的用户可观察行为。它不是工具内置概念，必须由本契约定义。

```tla
Observable(s) ==
  [
    current     |-> s.current,
    outArg      |-> s.outArg,
    outFlag     |-> s.outFlag,
    outMore     |-> s.outMore,
    err         |-> s.err,

    flags       |-> [
      f \in FlagNames |->
        [
          assigned |-> s.assigned[f],
          value    |-> s.value[f],
          values   |-> s.values[f]
        ]
    ],

    pendingFlag |-> s.currentFlag,
    pendingFrom |-> s.currentFlagOrigin
  ]

ObservableEq(a, b) ==
  Observable(a) = Observable(b)
```

本契约要求错误文案也属于 Observable，因此错误必须按类别和精确文本一起保护。

## 5. Error Contract

parser proof boundary 内产生的错误按精确文案保护。

```tla
NoError ==
  [class |-> "NoError", text |-> ""]

MissingValueErr(f) ==
  [class |-> "MissingValue",
   text  |-> "--" \o f \o " must be assigned with value"]

DuplicateFlagErr(f) ==
  [class |-> "DuplicateFlag",
   text  |-> "--" \o f \o " duplicated"]

FlagSetDuplicateErr(f) ==
  [class |-> "FlagSetDuplicate",
   text  |-> "flag duplicated --" \o f]

InvalidDoubleDashErr ==
  [class |-> "InvalidDoubleDash",
   text  |-> "not support '--' in command line"]

InvalidFlagFormErr(prefix) ==
  [class |-> "InvalidFlagForm",
   text  |-> "not support flag form " \o prefix]

InvalidFlagErr(name) ==
  [class |-> "InvalidFlag",
   text  |-> "invalid flag " \o name]

UnknownShortFlagErr(ch) ==
  [class |-> "UnknownShortFlag",
   text  |-> "unknown flag -" \o ch]

AssignToNoValueFlagErr(f) ==
  [class |-> "AssignToNoValueFlag",
   text  |-> "flag --" \o f \o " can't be assiged"]
```

注意：`can't be assiged` 保留源码中的拼写。若未来要修正这个拼写，应作为显式契约变更处理。

## 6. Enhancement Boundary

### 6.1 可带参数的正常 long flag

```tla
AssignableKnownLongFlag(f) ==
  /\ f \in KnownLongFlags
  /\ Mode[f] \in {
       AssignedDefault,
       AssignedOnce,
       AssignedRepeatable
     }
```

排除：

- `AssignedNone` flag，例如 `--help`
- dynamic unknown flag
- 未注册 long flag
- shorthand-only 场景

### 6.2 已识别 flag token

```tla
RecognizedFlagToken(t) ==
  \/ /\ t.prefixShape = "long_flag_candidate"
     /\ t.detectorResult = "known_flag"

  \/ /\ t.prefixShape = "long_flag_candidate"
     /\ t.detectorResult = "dynamic_unknown_flag"

  \/ /\ t.prefixShape = "short_flag_candidate"
     /\ t.detectorResult = "known_flag"

  \/ /\ t.prefixShape = "short_flag_candidate"
     /\ t.detectorResult = "dynamic_unknown_flag"
```

已识别 flag token 不能被前一个 pending flag 吞成 value。

### 6.3 dash-leading value 候选

```tla
DashLeadingValueCandidate(t) ==
  /\ t.splitterClass = "none"
  /\ t.prefixShape \in {
       "single_dash_multi",
       "short_flag_candidate"
     }
  /\ ~RecognizedFlagToken(t)
```

允许：

- `-1/-1`
- `-abc`
- `-1`，前提是它不是已注册 shorthand / dynamic unknown shorthand
- `-x`，前提是它不是已注册 shorthand / dynamic unknown shorthand

不允许：

- `--RegionId`
- `--unknown`
- `-h`，如果它是已注册 shorthand
- `-abc=value`
- `-abc:value`

### 6.4 本次允许的增强

```tla
AllowedDashValueEnhancement(st, t) ==
  /\ st.currentFlag # NoFlag
  /\ st.currentFlagOrigin = "long_flag_candidate"
  /\ AssignableKnownLongFlag(st.currentFlag)
  /\ NeedValue(st.value, st.currentFlag)
  /\ DashLeadingValueCandidate(t)
```

翻译：

```text
当前有 pending flag；
pending flag 来自正常 --long flag；
该 flag 已注册，且允许带参数；
该 flag 当前确实还需要 value；
下一个 token 以 "-" 开头；
但下一个 token 不是已识别 flag；
则新 parser 可以把下一个 token 当作当前 flag 的 value。
```

## 7. Formal Properties

### 7.1 Token Class Coverage

```tla
TokenClassCoverage ==
  {TokenClass(t) : t \in Tokens} = ValidTokenClasses
```

含义：

```text
Parser.tla 中参与 case 生成的 Tokens，
必须覆盖我们已经认可的 token class 全集。
如果以后新增 token class 维度但忘了加代表 token，
TLC 会直接失败。
```

`Tokens` 可以包含额外业务代表，例如 `-1/-1` 和 `-z`。这些额外代表不会扩大 token class 全集，只是让同一个 class 下的重要错误文案或业务值也进入 Go 契约测试。

### 7.2 Legacy Compatibility

```tla
LegacyCompatibility ==
  \A st \in States:
    \A t \in Tokens:
      ~AllowedDashValueEnhancement(st, t) =>
        ObservableEq(NewStep(st, t), OldStep(st, t))
```

含义：

```text
除显式增强场景外，
所有可达 parser 状态、所有已建模 token class，
新旧 parser 的 Observable 必须完全一致。
```

### 7.3 Enhancement Property

```tla
DashValueEnhancementProperty ==
  \A st \in States:
    \A t \in Tokens:
      AllowedDashValueEnhancement(st, t) =>
        LET ns == NewStep(st, t) IN
          /\ ns.err = NoError
          /\ ns.value[st.currentFlag] = t.raw
          /\ ns.currentFlag = NoFlag
          /\ ns.currentFlagOrigin = NoOrigin
          /\ ns.current = st.current + 1
```

含义：

```text
增强场景下不能报错；
value 必须等于原始 token；
pending flag 必须清空；
current 必须前进一个 token。
```

### 7.4 No Recognized Flag Swallowing

```tla
NoRecognizedFlagSwallowing ==
  \A st \in States:
    \A t \in Tokens:
      /\ st.currentFlag # NoFlag
      /\ RecognizedFlagToken(t)
      => ~AllowedDashValueEnhancement(st, t)
```

含义：

```text
只要下一个 token 是已识别 flag，
它就不能被前一个 pending flag 当作 value 吞掉。
```

### 7.5 No Splitter Dash Expansion

```tla
NoSplitterDashExpansion ==
  \A st \in States:
    \A t \in Tokens:
      /\ t.prefixShape \in {
           "single_dash_multi",
           "short_flag_candidate"
         }
      /\ t.splitterClass # "none"
      => ~AllowedDashValueEnhancement(st, t)
```

含义：

```text
-abc=value、-abc:value、-1=foo、-1:foo 暂不纳入增强；
这批 token 的旧行为必须保持。
```

## 8. Compatibility Matrix

| Scenario | Enhancement allowed | Required behavior |
|---|---:|---|
| `--PortRange -1/-1` | yes | `PortRange.value = "-1/-1"` |
| `--Count -1`, where `-1` is not recognized shorthand | yes | `Count.value = "-1"` |
| `--Name -abc`, where `-abc` is not recognized flag | yes | `Name.value = "-abc"` |
| `--PortRange --RegionId cn-hangzhou` | no | Keep legacy behavior; must not swallow `--RegionId` |
| `--Name -h`, where `-h` is registered help shorthand | no | Keep legacy behavior; must not swallow `-h` |
| `--help -1`, where `help` is `AssignedNone` | no | Keep legacy behavior; must not assign value to `--help` |
| `--Name -abc=value` | no | Keep legacy behavior |
| `--Name -abc:value` | no | Keep legacy behavior |
| `--Name --unknown` | no | Keep legacy behavior |
| non-pending `-1/-1` | no | Keep legacy behavior |

## 9. Contract Change Rules

Future changes must follow these rules:

1. If behavior changes outside `AllowedDashValueEnhancement`, `LegacyCompatibility` must fail.
2. If a future requirement needs new behavior, add a new `Allowed...Enhancement` definition first.
3. If a change intentionally modifies a protected Observable, treat it as an explicit contract change.
4. Error text changes inside parser proof boundary are contract changes.
5. Parser-external behavior must not be smuggled into this contract unless proof boundary is expanded.

## 10. Running the Formal Check

The runnable model lives in:

```text
formal/parser/Parser.tla
formal/parser/Parser.cfg
```

The repository does not vendor `tla2tools.jar`. Install or download it outside the repo, then run:

```bash
TLA2TOOLS_JAR=$HOME/.cache/tla2tools/tla2tools.jar make formal-parser
```

The Makefile target writes TLC metadata to `/tmp/aliyun-cli-tlc-parser-states` by default, not to the repository.

The formal check covers:

- token classes as finite sets;
- `TokenClassCoverage`;
- old and new parser state transitions;
- `Observable`;
- `LegacyCompatibility`;
- `DashValueEnhancementProperty`;
- `NoRecognizedFlagSwallowing`;
- `NoSplitterDashExpansion`.
