package permission

import (
	"bytes"
	"encoding/json"
	"github.com/tidwall/gjson"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/core"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/tvm"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const RpcAddr = "http://127.0.0.1:8101/"

const PublicRpcNamespace string = "Gzv_"
const (
	DataSourceRPC = iota + 1
	DataSourceLocalChain
)

var DataSourceType = DataSourceLocalChain

// Result is rpc Request successfully returns the variable parameter
type Result struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"Data"`
}

// ErrorResult is rpc Request error returned variable parameter
type ErrorResult struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RPCReqObj is complete rpc Request body
type RPCReqObj struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Jsonrpc string        `json:"jsonrpc"`
	ID      uint          `json:"id"`
}

// RPCResObj is complete rpc response body
type RPCResObj struct {
	Jsonrpc string       `json:"jsonrpc"`
	ID      uint         `json:"id"`
	Result  *Result      `json:"result,omitempty"`
	Error   *ErrorResult `json:"error,omitempty"`
}

func (r *Result) IsSuccess() bool {
	return r.Status == 0
}

type SendTxArgs struct {
	Data     string `json:"data"`
	Value    uint64 `json:"Value"`
	Gas      uint64 `json:"gas"`
	GasPrice uint64 `json:"gasprice"`
	TxType   int    `json:"tx_type"`
	Target   string `json:"target"`
	Source   string `json:"target"`
}

type TxRawData struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Value     uint64 `json:"value"`
	GasLimit  uint64 `json:"gas_limit"`
	GasPrice  uint64 `json:"gas_price"`
	TxType    int    `json:"type"`
	Nonce     uint64 `json:"nonce"`
	Data      []byte `json:"data"`
	Sign      string `json:"sign"`
	ExtraData []byte `json:"extra_data"`
}

func TxRawToTransaction(tx *TxRawData) *types.Transaction {
	var target *common.Address
	if tx.Target != "" {
		t := common.StringToAddress(tx.Target)
		target = &t
	}
	src := common.StringToAddress(tx.Source)
	var sign []byte
	if tx.Sign != "" {
		sign = common.HexToSign(tx.Sign).Bytes()
	}

	raw := &types.RawTransaction{
		Data:      tx.Data,
		Value:     types.NewBigInt(tx.Value),
		Nonce:     tx.Nonce,
		Target:    target,
		Type:      int8(tx.TxType),
		GasLimit:  types.NewBigInt(tx.GasLimit),
		GasPrice:  types.NewBigInt(tx.GasPrice),
		Sign:      sign,
		ExtraData: tx.ExtraData,
		Source:    &src,
	}
	return &types.Transaction{RawTransaction: raw, Hash: raw.GenHash()}
}

var DefaultTxArgs = SendTxArgs{
	Gas:      500000,
	GasPrice: 500,
	TxType:   types.TransactionTypeTransfer,
}

const DefaultGas uint64 = 500000

var DefaultCallTxArgs = SendTxArgs{
	Gas:      DefaultGas,
	GasPrice: 500,
	TxType:   types.TransactionTypeContractCall,
}
var DefaultDeployTxArgs = SendTxArgs{
	Target:   "",
	Gas:      DefaultGas,
	GasPrice: 500,
	TxType:   types.TransactionTypeContractCreate,
}

const TAS = 1000000000

type callArgs struct {
	target string
	TxArgs SendTxArgs
}

func CallTxArgsSetData(params ...interface{}) SendTxArgs {
	var tc = DefaultCallTxArgs
	tc.Data = params[0].(string)
	if len(params) > 1 {
		tc.Gas = params[1].(uint64)
	}
	if len(params) > 2 {
		tc.Value = params[2].(uint64)
	}
	return tc
}

func Query(f string, params ...interface{}) (string, error) {
	param := RPCReqObj{
		Method:  f,
		Params:  params[:],
		ID:      1,
		Jsonrpc: "2.0",
	}
	paramBytes, err := json.Marshal(param)
	if err != nil {
		panic("Marshal param error")
	}
	resp, err := http.Post(RpcAddr, "application/json", bytes.NewReader(paramBytes))
	if err != nil {
		return ("http error"), err
	}
	defer resp.Body.Close()
	responseBytes, _ := ioutil.ReadAll(resp.Body)
	response := string(responseBytes)
	if response[0:1] == " " || response[len(response)-1:] == " " {
		panic("json blank char at begin or end")
	}
	if !gjson.Valid(response) {
		panic("json format error")
	}
	return response, nil
}

func GetNonce(addr string) uint64 {

	if DataSourceType == DataSourceLocalChain {
		return core.BlockChainImpl.GetNonce(common.StringToAddress(addr)) + 1
	}

	js, err := Query(PublicRpcNamespace+"nonce", addr)
	if err != nil {
		return 0
	}
	return gjson.Get(js, "result").Uint()
}

func GetReceipt(hash string) string {
	if DataSourceType == DataSourceLocalChain {
		hash := common.HexToHash(hash)
		rc := core.BlockChainImpl.GetTransactionPool().GetReceipt(hash)
		b, err := json.Marshal(rc)
		if err != nil {
			return ""
		}
		return string(b)
	}
	js, err := Query(PublicRpcNamespace+"txReceipt", hash)
	if err != nil {
		return ""
	}
	return gjson.Get(js, "result.Receipt").Raw
}

func GetAccountData(addr string, key string) string {
	if DataSourceType == DataSourceLocalChain {
		addr = strings.TrimSpace(addr)
		// input check
		if !common.ValidateAddress(addr) {
			return ""
		}
		address := common.StringToAddress(addr)
		chain := core.BlockChainImpl
		state, err := chain.GetAccountDBByHash(chain.QueryTopBlock().Hash)
		if err != nil {
			return ""
		}
		var resultData interface{}
		iter := state.DataIterator(address, []byte(key))
		if iter != nil {
			tmp := make([]map[string]interface{}, 0)
			for iter.Next() {
				k := string(iter.Key[:])
				if !strings.HasPrefix(k, key) {
					continue
				}
				v := tvm.VmDataConvert(iter.Value[:])
				item := make(map[string]interface{}, 0)
				item["key"] = k
				item["value"] = v
				tmp = append(tmp, item)
				resultData = tmp
			}
		}

		b, err := json.Marshal(resultData)
		if err != nil {
			return ""
		}
		//fmt.Printf("get account data1:%v", string(b))
		return string(b)
	}
	count := 1
	if len(key) == 0 {
		count = 100
	}
	js, err := Query(PublicRpcNamespace+"queryAccountData", addr, key, count)
	if err != nil {
		return ""
	}
	//fmt.Printf("get account data2:%v", gjson.Get(js, "result").Raw)

	return gjson.Get(js, "result").Raw
}

func RandBytes() []byte {
	length := 10
	r := rand.New(rand.NewSource(time.Now().Unix()))
	bs := make([]byte, length)
	for i := 0; i < length; i++ {
		b := r.Intn(26) + 65
		bs[i] = byte(b)
	}
	return bs
}
