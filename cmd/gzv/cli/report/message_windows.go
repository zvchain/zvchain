package report

import (
	"github.com/axgle/mahonia"
	"math/big"
	"os/exec"
	"runtime"
	"strings"
)

func CpuInfo() (coreNum int, brand string) {
	output, _ := exec.Command("wmic", "cpu", "get", "name").Output()
	data := string(output)
	data = strings.ReplaceAll(data, "Name", "")
	data = strings.TrimSpace(data)
	return runtime.NumCPU(), data
}

func MemInfo() float64 {
	output, _ := exec.Command("wmic", "MEMORYCHIP", "get", "Capacity").Output()
	data := string(output)
	data = strings.ReplaceAll(data, "Capacity", "")
	data = strings.TrimSpace(data)
	mem := strings.Split(data, "\n")
	lens := len(mem)
	var total uint64
	if lens > 0 {
		for i := 0; i < lens; i++ {
			mx, _ := new(big.Int).SetString(strings.TrimSpace(mem[i]), 10)
			total += mx.Uint64()
		}
	}
	return float64(total) / (1024 * 1024 * 1024)
}

func OSInfo() (arch, os, version string) {
	output, _ := exec.Command("wmic", "os", "get", "Caption").Output()
	srcCoder := mahonia.NewDecoder("gbk")
	data := srcCoder.ConvertString(string(output))
	data = strings.ReplaceAll(data, "Caption", "")
	data = strings.TrimSpace(data)
	return runtime.GOARCH, runtime.GOOS, data
}

func HasProcessID(processId uint) bool {
	buf := bytes.Buffer{}
	cmd := exec.Command("wmic", "process", "get", "processid")
	cmd.Stdout = &buf
	cmd.Run()
	cmd2 := exec.Command("findstr", fmt.Sprintf("%d", processId))
	cmd2.Stdin = &buf
	data, _ := cmd2.CombinedOutput()
	if len(data) == 0 {
		return false
	}
	return true
}
