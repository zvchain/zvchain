package report

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/zvchain/zvchain/log"
	"net/http"
	"time"
)

const UploadMessagePeriod = time.Hour
const UploadUrl = "http://monitor.firepool.pro:8888/collect"

func StartReport(pubKey, version string, chainId int) {
	defer func() {
		err := recover()
		if err != nil {
			log.DefaultLogger.Errorln(err)
		}
	}()
	num, brand := CpuInfo()
	mem := MemInfo()
	arch, platform, osVersion := OSInfo()
	buffer := bytes.NewBuffer([]byte{})
	buffer.WriteString(pubKey)
	buffer.WriteByte(byte(num))
	buffer.WriteString(brand)
	buffer.WriteByte(byte(mem))
	buffer.WriteString(arch)
	buffer.WriteString(platform)
	buffer.WriteString(osVersion)
	bs := md5.Sum(buffer.Bytes())
	id := hex.EncodeToString(bs[:])
	nm := NodeMsg{
		ID:         id,
		Version:    version,
		ChainId:    chainId,
		OSVersion:  osVersion,
		OSArch:     arch,
		OSPlatform: platform,
		CpuNum:     num,
		CpuBrand:   brand,
		Mem:        mem,
	}
	data, err := json.Marshal(nm)
	if err != nil {
		log.DefaultLogger.Errorln(err)
		return
	}
	upload(data)
	ticker := time.NewTicker(UploadMessagePeriod)
	for range ticker.C {
		upload(data)
	}
}

func upload(data []byte) {
	reader := bytes.NewReader(data)
	resp, err := http.Post(UploadUrl, "application/json", reader)
	if err != nil {
		log.DefaultLogger.Errorln(err)
		return
	}
	resp.Body.Close()
}

type NodeMsg struct {
	ID         string  `json:"id"`
	ChainId    int     `json:"chain_id"`
	Version    string  `json:"version"`
	OSVersion  string  `json:"os_version"`
	OSArch     string  `json:"os_arch"`
	OSPlatform string  `json:"os_platform"`
	CpuNum     int     `json:"cpu_num"`
	CpuBrand   string  `json:"cpu_brand"`
	Mem        float64 `json:"mem"`
}
