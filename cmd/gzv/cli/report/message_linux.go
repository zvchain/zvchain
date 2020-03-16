package report

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func CpuInfo() (coreNum int, brand string) {
	command := "cat /proc/cpuinfo | grep name | cut -f2 -d: | uniq"
	output, _ := cmd(command)
	output = strings.TrimSpace(output)
	return runtime.NumCPU(), output
}

func MemInfo() float64 {
	command := "cat /proc/meminfo |grep MemTotal"
	output, _ := cmd(command)
	output = strings.ReplaceAll(output, "MemTotal:", "")
	output = strings.ReplaceAll(output, "kB", "")
	output = strings.TrimSpace(output)
	mem, _ := strconv.Atoi(output)
	return float64(mem) / (1024 * 1024)
}

func OSInfo() (arch, os, version string) {
	command := "cat /proc/version"
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
