package endpoint

import "testing"

func TestWithSuffix(t *testing.T) {
	t.Setenv(EnvSuffix, " aliyuncs.alibaba \n")
	for _, tc := range []struct{ in, want string }{
		{"ecs.cn-hangzhou.aliyuncs.com", "ecs.cn-hangzhou.aliyuncs.alibaba"},
		{"sts.aliyuncs.com:8443", "sts.aliyuncs.alibaba:8443"},
		{"https://bucket.oss-cn-hangzhou.aliyuncs.com:443/a%2Fb?next=ecs.aliyuncs.com#part", "https://bucket.oss-cn-hangzhou.aliyuncs.alibaba:443/a%2Fb?next=ecs.aliyuncs.com#part"},
		{"service.aliyuncs.com.aliyuncs.com", "service.aliyuncs.com.aliyuncs.alibaba"},
		{"ecs.ALIYUNCS.COM", "ecs.aliyuncs.alibaba"},
		{"ecs.cn-hangzhou.aliyuncs.alibaba", "ecs.cn-hangzhou.aliyuncs.alibaba"},
		{"https://example.com/a.aliyuncs.com", "https://example.com/a.aliyuncs.com"},
		{"ecs.aliyuncs.com.example.com", "ecs.aliyuncs.com.example.com"},
		{"notaliyuncs.com", "notaliyuncs.com"},
		{"aliyuncs.com", "aliyuncs.com"},
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"http://[::1]:8080/path", "http://[::1]:8080/path"},
		{"https://%bad", "https://%bad"},
		{"https://%zz", "https://%zz"},
		{"https://ecs.aliyuncs.com:invalid/path", "https://ecs.aliyuncs.com:invalid/path"},
		{"", ""},
	} {
		t.Run(tc.in, func(t *testing.T) {
			if got := WithSuffix(tc.in); got != tc.want {
				t.Fatalf("WithSuffix(%q) = %q; want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSuffixValidationAndDefault(t *testing.T) {
	for _, suffix := range []string{"", "aliyuncs.com", " aliyuncs.com ", "aliyuncs.alibaba", "evil.com", ".foo", "host:8080", "a/b", "ALIYUNCS.ALIBABA"} {
		t.Run(suffix, func(t *testing.T) {
			t.Setenv(EnvSuffix, suffix)
			valid := suffix == "" || suffix == "aliyuncs.com" || suffix == " aliyuncs.com " || suffix == "aliyuncs.alibaba"
			if err := Validate(); (err == nil) != valid {
				t.Fatalf("Validate() = %v", err)
			}
			if suffix != "aliyuncs.alibaba" && WithSuffix("ecs.aliyuncs.com") != "ecs.aliyuncs.com" {
				t.Fatal("default or invalid suffix changed destination")
			}
		})
	}
}
