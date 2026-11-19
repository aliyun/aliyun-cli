package cli

// UpdateNotice is an additive diagnostic in the v1 error envelope. It does not
// replace the original error, recovery command, or exit status.
type UpdateNotice struct {
	Component string `json:"component"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Severity  string `json:"severity"`
	Command   string `json:"command"`
	Message   string `json:"message"`
}

// SetUpdateNoticeProvider binds a cache-only provider. The returned completion
// callback records reminders only after they were successfully rendered.
func (ctx *Context) SetUpdateNoticeProvider(fn func() ([]UpdateNotice, func(bool))) {
	ctx.updateNoticeProvider = fn
}

func (ctx *Context) SetUpdateOwner(kind, name string) {
	ctx.updateOwnerKind, ctx.updateOwnerName = kind, name
}
func (ctx *Context) UpdateOwner() (string, string) { return ctx.updateOwnerKind, ctx.updateOwnerName }

func (ctx *Context) takeUpdateNotices() (notices []UpdateNotice, complete func(bool)) {
	complete = func(bool) {}
	// Automatic diagnostics must never turn a business error into a panic.
	defer func() {
		if recover() != nil {
			notices = nil
			complete = func(bool) {}
		}
	}()
	notices = append(notices, ctx.compatibilityNotices...)
	ctx.compatibilityNotices = nil
	if ctx.updateNoticeProvider != nil {
		remote, done := ctx.updateNoticeProvider()
		return append(notices, remote...), done
	}
	return
}

func (ctx *Context) AddCompatibilityNotice(notice UpdateNotice) {
	if ctx == nil {
		return
	}
	for _, n := range ctx.compatibilityNotices {
		if n.Component == notice.Component && n.Latest == notice.Latest {
			return
		}
	}
	ctx.compatibilityNotices = append(ctx.compatibilityNotices, notice)
}
