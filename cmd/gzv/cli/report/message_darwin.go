package report

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func CpuInfo() (coreNum int, brand string) {
	command := "sysctl machdep.cpu.brand_string"
	output, _ := cmd(command)
	output = strings.ReplaceAll(output, "machdep.cpu.brand_string:", "")
	output = strings.TrimSpace(output)
	return runtime.NumCPU(), output
}

func MemInfo() float64 {
	command := "sysctl hw.memsize"
	output, _ := cmd(command)
	output = strings.ReplaceAll(output, "hw.memsize:", "")
	output = strings.TrimSpace(output)
	mem, _ := strconv.Atoi(output)
	return float64(mem) / (1024 * 1024 * 1024)
}

func OSInfo() (arch, os, version string) {
	command := "sw_vers"
	output, _ := cmd(command)
	return runtime.GOARCH, runtime.GOOS, output
}

func cmd(cmd string) (string, error) {
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func HasProcessID(processId uint) bool {
	return true
}
