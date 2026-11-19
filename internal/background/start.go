package background

import "os/exec"

// Start disconnects all standard handles and reaps the child while the parent lives.
func Start(executable string, args []string, env []string) error {
	cmd := exec.Command(executable, args...)
	cmd.Env = env
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
