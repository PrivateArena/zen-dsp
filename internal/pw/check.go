package pw

import "os/exec"

func Check() bool {
	_, err := exec.LookPath("pw-dump")
	if err != nil {
		return false
	}
	return true
}
