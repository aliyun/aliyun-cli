package mock

import "testing"

func TestMatchIsCaseSensitive(t *testing.T) {
	rule := "ecs DescribeRegions"

	if !MatchCommand(rule, []string{"ecs", "DescribeRegions"}) {
		t.Fatalf("MatchCommand(%q, ecs DescribeRegions) = false, want true", rule)
	}
	if MatchCommand(rule, []string{"ECS", "DescribeRegions"}) {
		t.Fatalf("MatchCommand(%q, ECS DescribeRegions) = true, want false", rule)
	}
	if MatchCommand(rule, []string{"ecs", "describeregions"}) {
		t.Fatalf("MatchCommand(%q, ecs describeregions) = true, want false", rule)
	}
}

func TestMatchStarAndQuestionWildcards(t *testing.T) {
	if !MatchCommand("ecs *", []string{"ecs", "describe-regions", "--region", "cn-hangzhou"}) {
		t.Fatal("standalone * should match remaining args including spaces")
	}
	if !MatchCommand("ecs describe-regions --region cn-?", []string{"ecs", "describe-regions", "--region", "cn-a"}) {
		t.Fatal("cn-? should match cn-a")
	}
	if !MatchCommand("ecs describe-regions --region cn-?", []string{"ecs", "describe-regions", "--region", "cn-hangzhou"}) {
		t.Fatal("cn-? should match cn-hangzhou")
	}
}

func TestMatchQuestionMatchesMultipleNonSpaceCharacters(t *testing.T) {
	rule := "ecs op --name a?b"

	if !MatchCommand(rule, []string{"ecs", "op", "--name", "axb"}) {
		t.Fatal("a?b should match axb")
	}
	if !MatchCommand(rule, []string{"ecs", "op", "--name", "axxb"}) {
		t.Fatal("a?b should match axxb")
	}
	if MatchCommand(rule, []string{"ecs", "op", "--name", "ab"}) {
		t.Fatal("a?b should not match ab")
	}
}

func TestMatchQuestionDoesNotCrossWhitespace(t *testing.T) {
	if MatchCommand("ecs op --name a?b", []string{"ecs", "op", "--name", "ax", "b"}) {
		t.Fatal("? should not match across argument boundaries")
	}
}

func TestMatchQuestionMatchesLanguageCode(t *testing.T) {
	if !MatchCommand("ecs describe-regions --accept-language zh-?", []string{"ecs", "describe-regions", "--accept-language", "zh-CN"}) {
		t.Fatal("zh-? should match zh-CN")
	}
}

func TestMatchParametersAreUnordered(t *testing.T) {
	rule := "ecs describe-regions --region cn-? --foo bar"
	args := []string{"ecs", "describe-regions", "--foo", "bar", "--region", "cn-a"}

	if !MatchCommand(rule, args) {
		t.Fatalf("MatchCommand(%q, %#v) = false, want true", rule, args)
	}
}

func TestMatchIgnoresLeadingGlobalFlags(t *testing.T) {
	rule := "ecs DescribeRegions --region cn-?"
	args := []string{"--profile", "default", "--insecure", "ecs", "DescribeRegions", "--region", "cn-a"}

	if !MatchCommand(rule, args) {
		t.Fatalf("MatchCommand(%q, %#v) = false, want true", rule, args)
	}
}

func TestFirstCommandTokenSkipsLeadingGlobalFlags(t *testing.T) {
	args := []string{"--profile", "default", "-p", "test", "--insecure", "mock", "path"}

	if got := FirstCommandToken(args); got != "mock" {
		t.Fatalf("FirstCommandToken(%#v) = %q, want mock", args, got)
	}
}

func TestFirstCommandTokenReturnsEmptyWhenOnlyGlobalFlags(t *testing.T) {
	if got := FirstCommandToken([]string{"--profile", "default"}); got != "" {
		t.Fatalf("FirstCommandToken only flags = %q, want empty", got)
	}
}

func TestStripLeadingGlobalFlagsForms(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "inline long value",
			args: []string{"--profile=default", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "inline long colon value",
			args: []string{"--profile:default", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "short value",
			args: []string{"-p", "default", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "short no value",
			args: []string{"-q", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "dryrun no value",
			args: []string{"--dryrun", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "cli ai mode no value",
			args: []string{"--cli-ai-mode", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "no cli ai mode no value",
			args: []string{"--no-cli-ai-mode", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "double dash",
			args: []string{"--", "ecs", "DescribeRegions"},
			want: []string{"ecs", "DescribeRegions"},
		},
		{
			name: "triple dash is not a flag",
			args: []string{"---profile", "ecs", "DescribeRegions"},
			want: []string{"---profile", "ecs", "DescribeRegions"},
		},
		{
			name: "short group is not stripped",
			args: []string{"-abc", "ecs", "DescribeRegions"},
			want: []string{"-abc", "ecs", "DescribeRegions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLeadingGlobalFlags(tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("StripLeadingGlobalFlags(%#v) = %#v, want %#v", tt.args, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("StripLeadingGlobalFlags(%#v) = %#v, want %#v", tt.args, got, tt.want)
				}
			}
		})
	}
}

func TestMatchParametersBacktracksWildcardAmbiguity(t *testing.T) {
	rule := "ecs op --filter a* --filter ax --name abc"
	args := []string{"ecs", "op", "--filter", "ax", "--name", "abc", "--filter", "abc"}

	if !MatchCommand(rule, args) {
		t.Fatalf("MatchCommand(%q, %#v) = false, want true", rule, args)
	}
}

func TestMatchParameterGroupsKeepDashPrefixedValues(t *testing.T) {
	rule := "ecs op --first -a --second -b"

	matchingArgs := []string{"ecs", "op", "--second", "-b", "--first", "-a"}
	if !MatchCommand(rule, matchingArgs) {
		t.Fatalf("MatchCommand(%q, %#v) = false, want true", rule, matchingArgs)
	}

	swappedArgs := []string{"ecs", "op", "--second", "-a", "--first", "-b"}
	if MatchCommand(rule, swappedArgs) {
		t.Fatalf("MatchCommand(%q, %#v) = true, want false", rule, swappedArgs)
	}
}

func TestMatchParameterGroupsKeepFlagValueAssociation(t *testing.T) {
	rule := "ecs op --region cn-? --foo bar"
	args := []string{"ecs", "op", "--region", "bar", "--foo", "cn-a"}

	if MatchCommand(rule, args) {
		t.Fatalf("MatchCommand(%q, %#v) = true, want false", rule, args)
	}
}

func TestMatchAllowsExtraArgsWithoutStar(t *testing.T) {
	args := []string{"ecs", "describe-regions", "--region", "cn-a", "--extra", "value"}

	if !MatchCommand("ecs describe-regions --region cn-?", args) {
		t.Fatal("extra args should match because rule parameters are a subset")
	}
	if !MatchCommand("ecs describe-regions --region cn-? *", args) {
		t.Fatal("extra args should match when rule has standalone *")
	}
}

func TestMatchAllowsExtraFlagBeforeMatchedFlag(t *testing.T) {
	rule := "ecs describe-regions --accept-language zh-?"
	args := []string{"ecs", "describe-regions", "--region", "cn-shanghai", "--accept-language", "zh-aa"}

	if !MatchCommand(rule, args) {
		t.Fatalf("MatchCommand(%q, %#v) = false, want true", rule, args)
	}
}

func TestFindMatchUsesFirstMatchingRecord(t *testing.T) {
	records := []Record{
		{Name: "broad", Cmd: "ecs *"},
		{Name: "specific", Cmd: "ecs DescribeRegions"},
	}

	index, ok := FindMatch(records, []string{"ecs", "DescribeRegions"})

	if !ok {
		t.Fatal("FindMatch ok = false, want true")
	}
	if index != 0 {
		t.Fatalf("FindMatch index = %d, want 0", index)
	}
}

func TestConsumeMatchDecrementsAndDeletesExhaustedRecord(t *testing.T) {
	records := []Record{{Name: "once", Cmd: "ecs *", Times: 1}}

	got := Consume(records, 0)

	if len(got) != 0 {
		t.Fatalf("Consume returned %d records, want 0", len(got))
	}
}

func TestConsumeKeepsUnlimitedRecord(t *testing.T) {
	records := []Record{{Name: "unlimited", Cmd: "ecs *", Times: 0}}

	got := Consume(records, 0)

	if len(got) != 1 {
		t.Fatalf("Consume returned %d records, want 1", len(got))
	}
	if got[0].Times != 0 {
		t.Fatalf("Consume Times = %v, want 0", got[0].Times)
	}
}

func TestConsumeDecrementsWithoutDeletingRecord(t *testing.T) {
	records := []Record{{Name: "twice", Cmd: "ecs *", Times: 2}}

	got := Consume(records, 0)

	if len(got) != 1 {
		t.Fatalf("Consume returned %d records, want 1", len(got))
	}
	if got[0].Times != 1 {
		t.Fatalf("Consume Times = %d, want 1", got[0].Times)
	}
}

func TestConsumeInvalidIndexReturnsRecordsUnchanged(t *testing.T) {
	records := []Record{{Name: "twice", Cmd: "ecs *", Times: 2}}

	got := Consume(records, 1)

	if len(got) != len(records) {
		t.Fatalf("Consume returned %d records, want %d", len(got), len(records))
	}
	if &got[0] != &records[0] {
		t.Fatal("Consume should return the unchanged records slice for invalid index")
	}
	if got[0].Times != 2 {
		t.Fatalf("Consume Times = %d, want 2", got[0].Times)
	}
}
