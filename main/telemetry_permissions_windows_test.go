//go:build windows

package main

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"golang.org/x/sys/windows"
)

func denyTelemetryWrites(t *testing.T, path string) {
	t.Helper()
	original, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal(err)
	}
	if original == nil {
		t.Skip("filesystem does not provide a security descriptor for the permission fixture")
	}
	originalDACL, _, err := original.DACL()
	if errors.Is(err, windows.ERROR_OBJECT_NOT_FOUND) || err == nil && originalDACL == nil {
		// A null DACL allows all access; adding only a deny ACE would also remove
		// the implicit read allowance and would not model a read-only fixture.
		t.Skip("permission fixture requires an existing non-null DACL")
	}
	if err != nil {
		t.Fatal(err)
	}
	control, _, err := original.Control()
	if err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	var pin runtime.Pinner
	pin.Pin(user.User.Sid)
	defer pin.Unpin()
	dacl, err := windows.ACLFromEntries([]windows.EXPLICIT_ACCESS{{
		AccessPermissions: windows.FILE_WRITE_DATA | windows.FILE_APPEND_DATA |
			windows.FILE_WRITE_EA | windows.FILE_WRITE_ATTRIBUTES,
		AccessMode:  windows.DENY_ACCESS,
		Inheritance: windows.NO_INHERITANCE,
		Trustee: windows.TRUSTEE{
			TrusteeForm:  windows.TRUSTEE_IS_SID,
			TrusteeType:  windows.TRUSTEE_IS_USER,
			TrusteeValue: windows.TrusteeValueFromSID(user.User.Sid),
		},
	}}, originalDACL)
	if err != nil {
		t.Fatal(err)
	}
	restoreInfo := windows.SECURITY_INFORMATION(windows.DACL_SECURITY_INFORMATION)
	if control&windows.SE_DACL_PROTECTED != 0 {
		restoreInfo |= windows.PROTECTED_DACL_SECURITY_INFORMATION
	} else {
		restoreInfo |= windows.UNPROTECTED_DACL_SECURITY_INFORMATION
	}
	t.Cleanup(func() {
		err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, restoreInfo, nil, nil, originalDACL, nil)
		runtime.KeepAlive(original)
		if err != nil && !os.IsNotExist(err) {
			t.Errorf("restore telemetry fixture DACL: %v", err)
		}
	})
	// Keep the current inherited grants while adding a deny on this object only.
	// In particular, READ_CONTROL and SYNCHRONIZE remain usable, and existing
	// config/lock files do not inherit the directory's write denial.
	if err := windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		restoreInfo, nil, nil, dacl, nil); err != nil {
		t.Fatal(err)
	}
}
