package plugin

import "errors"

// LookupLocalPlugin resolves the installed plugin record for a trigger command token.
func LookupLocalPlugin(command string) (*LocalPlugin, error) {
	mgr, err := NewManager()
	if err != nil {
		return nil, err
	}
	_, lp, err := mgr.findLocalPlugin(command)
	if err != nil {
		var notFoundErr *ErrPluginNotFound
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, err
	}
	return lp, nil
}

// IsInnerPluginCommand reports whether command resolves to an installed inner plugin.
func IsInnerPluginCommand(command string) bool {
	lp, err := LookupLocalPlugin(command)
	return err == nil && lp != nil && lp.Inner
}
