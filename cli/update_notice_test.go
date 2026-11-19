package cli

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/require"
)

func TestUpdateNoticesMergeCompatibilityAndRemoteDiagnostics(t *testing.T) {
	ctx := NewCommandContext(io.Discard, io.Discard)
	ctx.SetUpdateOwner("meta", "example")
	kind, owner := ctx.UpdateOwner()
	require.Equal(t, "meta", kind)
	require.Equal(t, "example", owner)
	compatibility := UpdateNotice{Component: "example", Latest: "3.6.0", Severity: "compatibility"}
	ctx.AddCompatibilityNotice(compatibility)
	ctx.AddCompatibilityNotice(compatibility)
	var nilContext *Context
	nilContext.AddCompatibilityNotice(compatibility)
	remote := UpdateNotice{Component: "aliyun", Latest: "3.7.0", Severity: "update"}
	var completed bool
	ctx.SetUpdateNoticeProvider(func() ([]UpdateNotice, func(bool)) {
		return []UpdateNotice{remote}, func(rendered bool) { completed = rendered }
	})
	notices, complete := ctx.takeUpdateNotices()
	require.Equal(t, []UpdateNotice{compatibility, remote}, notices)
	complete(true)
	require.True(t, completed)
	ctx.SetUpdateNoticeProvider(nil)
	notices, complete = ctx.takeUpdateNotices()
	require.Empty(t, notices, "compatibility guidance is consumed once")
	require.NotPanics(t, func() { complete(true) })
}

func TestPanickingUpdateProviderPreservesOriginalFailure(t *testing.T) {
	DisableExitCode()
	t.Cleanup(EnableExitCode)
	var stderr bytes.Buffer
	ctx := NewCommandContext(io.Discard, &stderr)
	ctx.SetUpdateNoticeProvider(func() ([]UpdateNotice, func(bool)) { panic("unavailable cache") })
	require.NotPanics(t, func() { newAliyunCmd().processError(ctx, errors.New("original failure")) })
	require.Contains(t, stderr.String(), "original failure")
	require.NotContains(t, stderr.String(), "unavailable cache")
}

type updateNoticeFailWriter struct{}

func (updateNoticeFailWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestUpdateNoticeCompletionTracksRenderingSuccess(t *testing.T) {
	DisableExitCode()
	t.Cleanup(EnableExitCode)
	language := i18n.GetLanguage()
	i18n.SetLanguage("en")
	t.Cleanup(func() { i18n.SetLanguage(language) })
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "rendered", true: "closed_stderr"}[failed], func(t *testing.T) {
			var stderr bytes.Buffer
			var writer io.Writer = &stderr
			if failed {
				writer = updateNoticeFailWriter{}
			}
			ctx := NewCommandContext(io.Discard, writer)
			var calls int
			ctx.SetUpdateNoticeProvider(func() ([]UpdateNotice, func(bool)) {
				return []UpdateNotice{{Severity: "compatibility", Message: "upgrade the CLI first"}}, func(rendered bool) {
					calls++
					require.Equal(t, !failed, rendered)
				}
			})
			newAliyunCmd().processError(ctx, errors.New("original failure"))
			require.Equal(t, 1, calls)
			if !failed {
				require.Contains(t, stderr.String(), "original failure")
				require.Contains(t, stderr.String(), "[Compatibility warning] upgrade the CLI first")
			}
		})
	}
}
