------------------------------ MODULE Parser ------------------------------

EXTENDS Naturals, Sequences, FiniteSets, TLC

\* Finite atoms used by the parser model.
NoFlag == "NoFlag"
NoError == [class |-> "NoError", text |-> ""]
NoCase == "NoCase"

AssignedNone == "AssignedNone"
AssignedDefault == "AssignedDefault"
AssignedOnce == "AssignedOnce"
AssignedRepeatable == "AssignedRepeatable"

NoOrigin == "none"
LongOrigin == "long_flag_candidate"
ShortOrigin == "short_flag_candidate"
DynamicUnknownOrigin == "dynamic_unknown_flag"

SplitterClasses == {
  "none",
  "equal_nonempty",
  "equal_empty",
  "colon_nonempty",
  "colon_empty"
}

PrefixShapes == {
  "empty_prefix",
  "non_dash_prefix",
  "triple_dash_prefix",
  "bare_double_dash",
  "long_flag_candidate",
  "bare_single_dash",
  "short_flag_candidate",
  "single_dash_multi"
}

DetectorResults == {
  "known_flag",
  "dynamic_unknown_flag",
  "detector_error",
  "not_applicable"
}

FlagNames == {"PortRange", "Count", "Name", "RegionId", "help", "dynamic", "u"}
KnownLongFlags == {"PortRange", "Count", "Name", "RegionId", "help"}
KnownLongAssignableFlags == {"PortRange", "Count", "Name", "RegionId"}

Mode == [f \in FlagNames |->
  CASE f = "help" -> AssignedNone
    [] f \in {"dynamic", "u"} -> AssignedDefault
    [] OTHER -> AssignedOnce
]

AllowRepeatedUnknown == [f \in FlagNames |-> FALSE]

EmptyAssigned == [f \in FlagNames |-> FALSE]
EmptyValue == [f \in FlagNames |-> ""]
EmptyValues == [f \in FlagNames |-> <<>>]

State(current, currentFlag, currentFlagOrigin, assigned, value, values, outArg, outFlag, outMore, err) ==
  [
    current |-> current,
    currentFlag |-> currentFlag,
    currentFlagOrigin |-> currentFlagOrigin,
    assigned |-> assigned,
    value |-> value,
    values |-> values,
    outArg |-> outArg,
    outFlag |-> outFlag,
    outMore |-> outMore,
    err |-> err
  ]

InitState ==
  State(0, NoFlag, NoOrigin, EmptyAssigned, EmptyValue, EmptyValues, "", NoFlag, FALSE, NoError)

Observable(s) ==
  [
    current |-> s.current,
    outArg |-> s.outArg,
    outFlag |-> s.outFlag,
    outMore |-> s.outMore,
    err |-> s.err,
    flags |->
      [f \in FlagNames |->
        [
          assigned |-> s.assigned[f],
          value |-> s.value[f],
          values |-> s.values[f]
        ]
      ],
    pendingFlag |-> s.currentFlag,
    pendingFrom |-> s.currentFlagOrigin
  ]

ObservableEq(a, b) ==
  Observable(a) = Observable(b)

MissingValueErr(f) ==
  [class |-> "MissingValue", text |-> "--" \o f \o " must be assigned with value"]

DuplicateFlagErr(f) ==
  [class |-> "DuplicateFlag", text |-> "--" \o f \o " duplicated"]

FlagSetDuplicateErr(f) ==
  [class |-> "FlagSetDuplicate", text |-> "flag duplicated --" \o f]

InvalidDoubleDashErr ==
  [class |-> "InvalidDoubleDash", text |-> "not support '--' in command line"]

InvalidFlagFormErr(prefix) ==
  [class |-> "InvalidFlagForm", text |-> "not support flag form " \o prefix]

InvalidFlagErr(name) ==
  [class |-> "InvalidFlag", text |-> "invalid flag " \o name]

UnknownShortFlagErr(ch) ==
  [class |-> "UnknownShortFlag", text |-> "unknown flag -" \o ch]

AssignToNoValueFlagErr(f) ==
  [class |-> "AssignToNoValueFlag", text |-> "flag --" \o f \o " can't be assiged"]

NeedValue(value, f) ==
  CASE Mode[f] = AssignedNone -> FALSE
    [] Mode[f] = AssignedDefault -> value[f] = ""
    [] Mode[f] = AssignedOnce -> value[f] = ""
    [] Mode[f] = AssignedRepeatable -> TRUE

ValidateOK(value, f) ==
  ~(Mode[f] = AssignedOnce /\ value[f] = "")

AssignableKnownLongFlag(f) ==
  /\ f \in KnownLongFlags
  /\ Mode[f] \in {AssignedDefault, AssignedOnce, AssignedRepeatable}

Token(raw, splitterClass, prefixShape, detectorResult, flag, value, parseError) ==
  [
    raw |-> raw,
    splitterClass |-> splitterClass,
    prefixShape |-> prefixShape,
    detectorResult |-> detectorResult,
    flag |-> flag,
    value |-> value,
    parseError |-> parseError
  ]

DetectorPrefixShapes == {"long_flag_candidate", "short_flag_candidate"}

DetectorResultsForPrefix(prefixShape) ==
  IF prefixShape \in DetectorPrefixShapes
  THEN {"known_flag", "dynamic_unknown_flag", "detector_error"}
  ELSE {"not_applicable"}

ValidTokenClasses ==
  UNION {
    {
      [
        splitterClass |-> sc,
        prefixShape |-> ps,
        detectorResult |-> dr
      ] :
        dr \in DetectorResultsForPrefix(ps)
    } :
      sc \in SplitterClasses,
      ps \in PrefixShapes
  }

SplitterSuffix(splitterClass) ==
  CASE splitterClass = "none" -> ""
    [] splitterClass = "equal_nonempty" -> "=value"
    [] splitterClass = "equal_empty" -> "="
    [] splitterClass = "colon_nonempty" -> ":value"
    [] splitterClass = "colon_empty" -> ":"

SplitValue(splitterClass) ==
  CASE splitterClass \in {"equal_nonempty", "colon_nonempty"} -> "value"
    [] OTHER -> ""

ClassPrefix(c) ==
  CASE c.prefixShape = "empty_prefix" -> ""
    [] c.prefixShape = "non_dash_prefix" -> "value"
    [] c.prefixShape = "triple_dash_prefix" -> "---cert"
    [] c.prefixShape = "bare_double_dash" -> "--"
    [] c.prefixShape = "bare_single_dash" -> "-"
    [] c.prefixShape = "single_dash_multi" -> "-abc"
    [] c.prefixShape = "long_flag_candidate" ->
      CASE c.detectorResult = "known_flag" -> "--PortRange"
        [] c.detectorResult = "dynamic_unknown_flag" -> "--dynamic"
        [] OTHER -> "--unknown"
    [] c.prefixShape = "short_flag_candidate" ->
      CASE c.detectorResult = "known_flag" -> "-h"
        [] c.detectorResult = "dynamic_unknown_flag" -> "-u"
        [] OTHER -> "-1"

RepresentativeRaw(c) ==
  ClassPrefix(c) \o SplitterSuffix(c.splitterClass)

FlagForClass(c) ==
  CASE c.prefixShape = "long_flag_candidate" /\ c.detectorResult = "known_flag" -> "PortRange"
    [] c.prefixShape = "long_flag_candidate" /\ c.detectorResult = "dynamic_unknown_flag" -> "dynamic"
    [] c.prefixShape = "short_flag_candidate" /\ c.detectorResult = "known_flag" -> "help"
    [] c.prefixShape = "short_flag_candidate" /\ c.detectorResult = "dynamic_unknown_flag" -> "u"
    [] OTHER -> NoFlag

ParseErrorForClass(c) ==
  CASE c.prefixShape = "bare_double_dash" -> InvalidDoubleDashErr
    [] c.prefixShape = "bare_single_dash" -> InvalidFlagFormErr("-")
    [] c.prefixShape = "single_dash_multi" -> InvalidFlagFormErr("-abc")
    [] c.prefixShape = "long_flag_candidate" /\ c.detectorResult = "detector_error" -> InvalidFlagErr("unknown")
    [] c.prefixShape = "short_flag_candidate" /\ c.detectorResult = "detector_error" -> UnknownShortFlagErr("1")
    [] OTHER -> NoError

ValueForClass(c) ==
  CASE c.prefixShape \in {"empty_prefix", "non_dash_prefix", "triple_dash_prefix"} -> RepresentativeRaw(c)
    [] c.splitterClass \in {"equal_nonempty", "colon_nonempty"} -> "value"
    [] OTHER -> ""

TokenFromClass(c) ==
  Token(
    RepresentativeRaw(c),
    c.splitterClass,
    c.prefixShape,
    c.detectorResult,
    FlagForClass(c),
    ValueForClass(c),
    ParseErrorForClass(c)
  )

KnownLongAssignedOnceToken(flagName, splitterClass) ==
  Token(
    "--" \o flagName \o SplitterSuffix(splitterClass),
    splitterClass,
    "long_flag_candidate",
    "known_flag",
    flagName,
    SplitValue(splitterClass),
    NoError
  )

KnownLongNoValueToken(splitterClass) ==
  Token(
    "--help" \o SplitterSuffix(splitterClass),
    splitterClass,
    "long_flag_candidate",
    "known_flag",
    "help",
    SplitValue(splitterClass),
    NoError
  )

KnownShortAssignedOnceToken(splitterClass) ==
  Token(
    "-c" \o SplitterSuffix(splitterClass),
    splitterClass,
    "short_flag_candidate",
    "known_flag",
    "Count",
    SplitValue(splitterClass),
    NoError
  )

DashRangeValueToken ==
  Token(
    "-1/-1",
    "none",
    "single_dash_multi",
    "not_applicable",
    NoFlag,
    "",
    InvalidFlagFormErr("-1/-1")
  )

UnknownShortLetterToken ==
  Token(
    "-z",
    "none",
    "short_flag_candidate",
    "detector_error",
    NoFlag,
    "",
    UnknownShortFlagErr("z")
  )

ClassRepresentativeTokens == {TokenFromClass(c) : c \in ValidTokenClasses}

SemanticRepresentativeTokens ==
  {KnownLongAssignedOnceToken(f, splitterClass) :
    f \in KnownLongAssignableFlags,
    splitterClass \in SplitterClasses}
  \cup {KnownLongNoValueToken(splitterClass) : splitterClass \in SplitterClasses}
  \cup {KnownShortAssignedOnceToken(splitterClass) : splitterClass \in SplitterClasses}
  \cup {DashRangeValueToken, UnknownShortLetterToken}

Tokens == ClassRepresentativeTokens \cup SemanticRepresentativeTokens

TokenClass(t) ==
  [
    splitterClass |-> t.splitterClass,
    prefixShape |-> t.prefixShape,
    detectorResult |-> t.detectorResult
  ]

TokenClassesInTokens == {TokenClass(t) : t \in Tokens}

TokenClassCoverage ==
  TokenClassesInTokens = ValidTokenClasses

RecognizedFlagToken(t) ==
  \/ /\ t.prefixShape = "long_flag_candidate"
     /\ t.detectorResult \in {"known_flag", "dynamic_unknown_flag"}
  \/ /\ t.prefixShape = "short_flag_candidate"
     /\ t.detectorResult \in {"known_flag", "dynamic_unknown_flag"}

DashLeadingValueCandidate(t) ==
  /\ t.splitterClass = "none"
  /\ t.prefixShape \in {"single_dash_multi", "short_flag_candidate"}
  /\ ~RecognizedFlagToken(t)

AllowedDashValueEnhancement(st, t) ==
  /\ st.currentFlag # NoFlag
  /\ st.currentFlagOrigin = LongOrigin
  /\ AssignableKnownLongFlag(st.currentFlag)
  /\ NeedValue(st.value, st.currentFlag)
  /\ DashLeadingValueCandidate(t)

AssignResult(st, f, x) ==
  IF Mode[f] = AssignedNone
  THEN [st EXCEPT !.err = AssignToNoValueFlagErr(f)]
  ELSE
    [st EXCEPT
      !.assigned = [st.assigned EXCEPT ![f] = TRUE],
      !.value = [st.value EXCEPT ![f] = x],
      !.values =
        IF Mode[f] = AssignedRepeatable
        THEN [st.values EXCEPT ![f] = Append(st.values[f], x)]
        ELSE st.values,
      !.err = NoError
    ]

SetAssignedResult(st, f) ==
  IF st.assigned[f] = FALSE
  THEN [st EXCEPT !.assigned = [st.assigned EXCEPT ![f] = TRUE], !.err = NoError]
  ELSE
    IF Mode[f] = AssignedRepeatable
    THEN [st EXCEPT !.err = NoError]
    ELSE
      IF AllowRepeatedUnknown[f]
      THEN
        [st EXCEPT
          !.value = [st.value EXCEPT ![f] = ""],
          !.values = [st.values EXCEPT ![f] = <<>>],
          !.err = NoError
        ]
      ELSE [st EXCEPT !.err = DuplicateFlagErr(f)]

OldValueStep(st, t) ==
  IF st.currentFlag # NoFlag
  THEN
    LET assignedSt == AssignResult(st, st.currentFlag, t.value) IN
      IF assignedSt.err # NoError
      THEN assignedSt
      ELSE
        IF NeedValue(assignedSt.value, st.currentFlag)
        THEN assignedSt
        ELSE [assignedSt EXCEPT !.currentFlag = NoFlag, !.currentFlagOrigin = NoOrigin]
  ELSE [st EXCEPT !.outArg = t.value, !.err = NoError]

FlagOrigin(t) ==
  IF t.detectorResult = "dynamic_unknown_flag"
  THEN DynamicUnknownOrigin
  ELSE
    IF t.prefixShape = "long_flag_candidate"
    THEN LongOrigin
    ELSE ShortOrigin

DuplicateDynamicUnknownToken(st, t) ==
  /\ st.currentFlag # NoFlag
  /\ st.currentFlagOrigin = DynamicUnknownOrigin
  /\ t.detectorResult = "dynamic_unknown_flag"
  /\ t.flag = st.currentFlag

OldFlagStep(st, t) ==
  LET marked == SetAssignedResult(st, t.flag) IN
    IF marked.err # NoError
    THEN marked
    ELSE
      IF st.currentFlag # NoFlag /\ ~ValidateOK(st.value, st.currentFlag)
      THEN [marked EXCEPT !.err = MissingValueErr(st.currentFlag)]
      ELSE
        LET cleared == [marked EXCEPT !.currentFlag = NoFlag, !.currentFlagOrigin = NoOrigin] IN
          IF t.value # ""
          THEN AssignResult(cleared, t.flag, t.value)
          ELSE
            IF NeedValue(cleared.value, t.flag)
            THEN [cleared EXCEPT !.currentFlag = t.flag, !.currentFlagOrigin = FlagOrigin(t), !.err = NoError]
            ELSE [cleared EXCEPT !.err = NoError]

Advance(st) ==
  [st EXCEPT !.current = st.current + 1, !.outArg = "", !.outFlag = NoFlag, !.outMore = TRUE, !.err = NoError]

OldStep(st, t) ==
  LET base == Advance(st) IN
    IF DuplicateDynamicUnknownToken(st, t)
    THEN [base EXCEPT !.err = FlagSetDuplicateErr(t.flag)]
    ELSE IF t.parseError # NoError
    THEN [base EXCEPT !.err = t.parseError]
    ELSE
      IF t.flag = NoFlag
      THEN OldValueStep(base, t)
      ELSE [OldFlagStep(base, t) EXCEPT !.outFlag = t.flag]

NewStep(st, t) ==
  IF AllowedDashValueEnhancement(st, t)
  THEN
    LET base == Advance(st) IN
    LET assignedSt == AssignResult(base, st.currentFlag, t.raw) IN
      [assignedSt EXCEPT !.currentFlag = NoFlag, !.currentFlagOrigin = NoOrigin]
  ELSE OldStep(st, t)

PendingLongState(f) ==
  [InitState EXCEPT
    !.current = 1,
    !.currentFlag = f,
    !.currentFlagOrigin = LongOrigin,
    !.assigned = [InitState.assigned EXCEPT ![f] = TRUE],
    !.outFlag = f,
    !.outMore = TRUE
  ]

PendingShortCountState ==
  [InitState EXCEPT
    !.current = 1,
    !.currentFlag = "Count",
    !.currentFlagOrigin = ShortOrigin,
    !.assigned = [InitState.assigned EXCEPT !["Count"] = TRUE],
    !.outFlag = "Count",
    !.outMore = TRUE
  ]

PendingDynamicLongState ==
  [InitState EXCEPT
    !.current = 1,
    !.currentFlag = "dynamic",
    !.currentFlagOrigin = DynamicUnknownOrigin,
    !.assigned = [InitState.assigned EXCEPT !["dynamic"] = TRUE],
    !.outFlag = "dynamic",
    !.outMore = TRUE
  ]

PendingDynamicShortState ==
  [InitState EXCEPT
    !.current = 1,
    !.currentFlag = "u",
    !.currentFlagOrigin = DynamicUnknownOrigin,
    !.assigned = [InitState.assigned EXCEPT !["u"] = TRUE],
    !.outFlag = "u",
    !.outMore = TRUE
  ]

States ==
  {InitState}
    \cup {PendingLongState(f) : f \in KnownLongAssignableFlags}
    \cup {PendingShortCountState, PendingDynamicLongState, PendingDynamicShortState}

NamedState(name, state) ==
  [name |-> name, state |-> state]

NamedStates ==
  {NamedState("init", InitState)}
    \cup {NamedState("pending_" \o f, PendingLongState(f)) :
      f \in KnownLongAssignableFlags}
    \cup {NamedState("pending_short_Count", PendingShortCountState)}
    \cup {NamedState("pending_dynamic_long_dynamic", PendingDynamicLongState)}
    \cup {NamedState("pending_dynamic_short_u", PendingDynamicShortState)}

StateRequiresUnknownDetector(ns) ==
  ns.name \in {"pending_dynamic_long_dynamic", "pending_dynamic_short_u"}

TokenApplicable(ns, t) ==
  ~(StateRequiresUnknownDetector(ns) /\ t.detectorResult = "detector_error")

Case(ns, t) ==
  [
    stateName |-> ns.name,
    tokenRaw |-> t.raw,
    tokenClass |-> TokenClass(t),
    tokenFlag |-> t.flag,
    tokenValue |-> t.value,
    tokenParseError |-> t.parseError,
    allowedDashValueEnhancement |-> AllowedDashValueEnhancement(ns.state, t),
    oldObservable |-> Observable(OldStep(ns.state, t)),
    newObservable |-> Observable(NewStep(ns.state, t))
  ]

Cases ==
  UNION {
    {Case(ns, t) : t \in {tt \in Tokens : TokenApplicable(ns, tt)}} :
      ns \in NamedStates
  }

LegacyCompatibility ==
  \A st \in States:
    \A t \in Tokens:
      ~AllowedDashValueEnhancement(st, t) =>
        ObservableEq(NewStep(st, t), OldStep(st, t))

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

NoRecognizedFlagSwallowing ==
  \A st \in States:
    \A t \in Tokens:
      /\ st.currentFlag # NoFlag
      /\ RecognizedFlagToken(t)
      => ~AllowedDashValueEnhancement(st, t)

NoSplitterDashExpansion ==
  \A st \in States:
    \A t \in Tokens:
      /\ t.prefixShape \in {"single_dash_multi", "short_flag_candidate"}
      /\ t.splitterClass # "none"
      => ~AllowedDashValueEnhancement(st, t)

VARIABLES dummy, selectedCase

Init ==
  /\ dummy = 0
  /\ selectedCase = NoCase

Next == UNCHANGED <<dummy, selectedCase>>

Spec == Init /\ [][Next]_<<dummy, selectedCase>>

CaseInit ==
  /\ dummy = 0
  /\ selectedCase \in Cases

CaseNext == UNCHANGED <<dummy, selectedCase>>

CaseSpec == CaseInit /\ [][CaseNext]_<<dummy, selectedCase>>

=============================================================================
