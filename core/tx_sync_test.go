package core

import (
	"bytes"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/tasdb"
	"io/ioutil"
	"math/big"
	"testing"
	"time"
)

//func TestCheckReceivedHashesInHitRate(t *testing.T) {
//	peer := newPeerTxsKeys()
//	txs := makeOrderTransactions()
//	for i := 0; i < len(txs); i++ {
//		peer.addSendHash(txs[i].Hash)
//	}
//
//	isSuccess := peer.checkReceivedHashesInHitRate(txs)
//	if !isSuccess {
//		t.Fatalf("except success,but got failed!")
//	}
//
//	changeHalfTxsToInvaildHashs(txs)
//
//	isSuccess = peer.checkReceivedHashesInHitRate(txs)
//	if !isSuccess {
//		t.Fatalf("except success,but got failed!")
//	}
//	txlen := len(txs)
//
//	txs[txlen-1].Hash = common.BigToAddress(big.NewInt(int64(300000))).Hash()
//	isSuccess = peer.checkReceivedHashesInHitRate(txs)
//	if isSuccess {
//		t.Fatalf("except success,but got failed!")
//	}
//	changeHashsToSame(txs)
//
//	isSuccess = peer.checkReceivedHashesInHitRate(txs)
//	if isSuccess {
//		t.Fatalf("except success,but got failed!")
//	}
//}
//
//func changeHalfTxsToInvaildHashs(txs []*types.Transaction) {
//	for i := 0; i < len(txs)/2; i++ {
//		txs[i].Hash = common.BigToAddress(big.NewInt(int64(i + 30000))).Hash()
//	}
//}
//
//func changeHashsToSame(txs []*types.Transaction) {
//	for i := 0; i < len(txs); i++ {
//		txs[i].Hash = common.BigToAddress(big.NewInt(int64(1))).Hash()
//	}
//}
//
//func makeOrderTransactions() []*types.Transaction {
//	txs := []*types.Transaction{}
//	for i := 1; i <= 200; i++ {
//		tx := &types.Transaction{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash()}
//		txs = append(txs, tx)
//	}
//	return txs
//}

// DefaultMessage is a default implementation of the Message interface.
// It can meet most of demands abort chain event
//type DefaultMessage struct {
//	body            []byte
//	source          string
//	chainID         uint16
//	protocalVersion uint16
//}

//func (m *DefaultMessage) GetRaw() []byte {
//	panic("implement me")
//}
//
//func (m *DefaultMessage) GetData() interface{} {
//	return m.Body
//}
//
//func (m *DefaultMessage) Body() []byte {
//	return m.body
//}
//
//func (m *DefaultMessage) Source() string {
//	return m.source
//}

func initTxSyncerTest(chain *FullBlockChain, pool *txPool) {
	s := &txSyncer{
		rctNotifiy:    common.MustNewLRUCache(txPeerMaxLimit),
		pool:          pool,
		ticker:        ticker.NewGlobalTicker("tx_syncer"),
		candidateKeys: common.MustNewLRUCache(100),
		chain:         chain,
		//logger:        taslog.GetLoggerByIndex(taslog.TxSyncLogConfig, common.GlobalConf.GetString("instance", "index", "")),
	}
	TxSyncer = s
}

//type txPoolTest struct {
//	bonPool   *rewardPool
//	received  *simpleContainer
//	asyncAdds *lru.Cache // Asynchronously added, accelerates validated transaction
//	// when add block on chain, does not participate in the broadcast
//
//	receiptDb          *tasdb.PrefixedDatabase
//	batch              tasdb.Batch
//	chain              types.BlockChain
//	gasPriceLowerBound *types.BigInt
//	lock               sync.RWMutex
//}

// newTransactionPool returns a new transaction tool object
func newTransactionPoolTest(chain *FullBlockChain, receiptDb *tasdb.PrefixedDatabase) types.TransactionPool {
	pool := &txPool{
		receiptDb: receiptDb,
		//batch:              chain.batch,
		asyncAdds: common.MustNewLRUCache(txCountPerBlock * maxReqBlockCount),
		chain:     chain,
		//gasPriceLowerBound: types.NewBigInt(uint64(common.GlobalConf.GetInt("chain", "gasprice_lower_bound", 1))),
	}
	pool.received = newSimpleContainer(maxPendingSize, maxQueueSize, chain)
	pool.bonPool = newRewardPool(chain.rewardManager, rewardTxMaxSize)
	initTxSyncerTest(chain, pool)

	return pool
}

//
//func initContext4Test() error {
//	common.DefaultLogger = taslog.GetLoggerByName("default")
//	common.InitConf("../tas_config_all.ini")
//	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
//	err := middleware.InitMiddleware()
//	if err != nil {
//		return err
//	}
//	BlockChainImpl = nil
//
//	err = InitCore(NewConsensusHelper4Test(groupsig.ID{}), getAccount())
//	GroupManagerImpl.RegisterGroupCreateChecker(&GroupCreateChecker4Test{})
//	clearTicker()
//	return err
//}
//
//func getAccount() *Account4Test {
//	var ksr = new(KeyStoreRaw4Test)
//
//	ksr.Key = common.Hex2Bytes("A8cmFJACR7VqbbuKwDYu/zj/hn6hcox97ujw2TNvCYk=")
//	secKey := new(common.PrivateKey)
//	if !secKey.ImportKey(ksr.Key) {
//		fmt.Errorf("failed to import key")
//		return nil
//	}
//
//	account := &Account4Test{
//		Sk:       secKey.Hex(),
//		Pk:       secKey.GetPubKey().Hex(),
//		Address:  secKey.GetPubKey().GetAddress().Hex(),
//		Password: "Password",
//	}
//	return account
//}
//
//func NewConsensusHelper4Test(id groupsig.ID) types.ConsensusHelper {
//	return &ConsensusHelperImpl4Test{ID: id}
//}

/******************************************************/

//func (ts *txSyncer) onTxNotifyTest(msg notify.Message) {
//	nm := msg.(*DefaultMessageTest)
//	if peerManagerImpl.getOrAddPeer(nm.Source()).isEvil() {
//		ts.logger.Warnf("tx sync this source is is in evil...source is is %v\n", nm.Source())
//		return
//	}
//	reader := bytes.NewReader(nm.Body())
//	var (
//		hashs = make([]common.Hash, 0)
//		buf   = make([]byte, len(common.Hash{}))
//		count = 0
//	)
//
//	for {
//		n, _ := reader.Read(buf)
//		if n != len(common.Hash{}) {
//			break
//		}
//		if count > txMaxNotifyPerTime {
//			ts.logger.Warnf("Rcv onTxNotify,but count exceeds limit")
//			return
//		}
//		count++
//		hashs = append(hashs, common.BytesToHash(buf))
//	}
//	//fmt.Println(">>>>")
//	//fmt.Println(count)
//	//for _, v := range hashs{
//	//	fmt.Println(v)
//	//}
//	candidateKeys := ts.getOrAddCandidateKeys(nm.Source())
//	fmt.Println("candidateKeys:", candidateKeys)
//	accepts := make([]common.Hash, 0)
//	for _, k := range hashs {
//		if exist, _ := ts.pool.IsTransactionExisted(k); !exist {
//			accepts = append(accepts, k)
//		}
//	}
//	fmt.Println("accepts:", accepts)
//	candidateKeys.addTxHashes(accepts)
//	ts.logger.Debugf("Rcv txs notify from %v, size %v, accept %v, totalOfSource %v", nm.Source(), len(hashs), len(accepts), candidateKeys.txHashes.Len())
//
//}

const (
	adminAddress    = "0x6d880ddbcfb24180901d1ea709bb027cd86f79936d5ed23ece70bd98f22f2d84"
	adminPrivateKey = "0x04d5af5f6059473de094fe1b80a09ce30afa0d94cb2199f85762606fe146df8ec14b97c53bcbe876f5c61d7aba278ab3296425dfd492c573813618852abf71656a43fcc67b959cd448bdffc64c5ab07fbb04ee6b63df7804aa1dacee7f05163bf5"
	adminBPK        = "0x1adf7e028e8da12d1e4405e3457c9f9dbf9deb493ba2f72eab97eb09d6cae18723fe9ad86d3e5be397d5072aaf244a147e5ae72da6f4f4603cf6787d8cf456182e7127513f28f05a2ce4a4fd876b180c6d9bba92a8083ab204e48bdaea79c62624038994382dc2fed1deb3a979ddefbbbdeebc4dc27130055fd2a3b77199265d"
	adminBSK        = "0x1441a7b4cbd541e9d605286a8c4438bb8629714c3ebc5c32070991449fd4288b"
	adminVRFPK      = "0x70cf165338af4db313bf8e701f726b6b729215efb6015a1fc0733339af0c0ca0"
	adminVRFSK      = "0xd5d2e180509bc290b7463f4492499a3026f9126e25a21e77169167945fd4288f70cf165338af4db313bf8e701f726b6b729215efb6015a1fc0733339af0c0ca0"
)

// Supported transaction types
const (
	TransactionTypeTransfer       = 0
	TransactionTypeContractCreate = 1
	TransactionTypeContractCall   = 2
	TransactionTypeReward         = 3

	// Miner operation related type
	TransactionTypeStakeAdd    = 4
	TransactionTypeMinerAbort  = 5
	TransactionTypeStakeReduce = 6
	TransactionTypeStakeRefund = 7

	// Group operation related type
	TransactionTypeGroupPiece       = 9  //group member upload his encrypted share piece
	TransactionTypeGroupMpk         = 10 //group member upload his mpk
	TransactionTypeGroupOriginPiece = 11 //group member upload origin share piece

	TransactionTypeToBeRemoved = -1
)

// BigUint used as big.Int. Inheritance is for the implementation of Marshaler/Unmarshaler interface in msgpack framework
type BigInt struct {
	big.Int
}

func NewBigInt(v uint64) *BigInt {
	return &BigInt{Int: *new(big.Int).SetUint64(v)}
}

type FakeTransaction struct {
	Data   []byte  `msgpack:"dt,omitempty"` // Data of the transaction, cost gas
	Value  *BigInt `msgpack:"v"`            // The value the sender suppose to transfer
	Nonce  uint64  `msgpack:"nc"`           // The nonce indicates the transaction sequence related to sender
	Target []byte  `msgpack:"tg,omitempty"` // The receiver address
	Type   int     `msgpack:"tp"`           // Transaction type

	GasLimit *BigInt     `msgpack:"gl"`
	GasPrice *BigInt     `msgpack:"gp"`
	Hash     common.Hash `msgpack:"h"`

	ExtraData []byte          `msgpack:"ed"`
	Sign      []byte          `msgpack:"si"`  // The Sign of the sender
	Source    *common.Address `msgpack:"src"` // Sender address, recovered from sign
}

// GetBytesWithSign returns a byte array of the number with the first byte representing its sign.
// It must be success
func (b *BigInt) GetBytesWithSign() []byte {
	if b == nil {
		return []byte{}
	}
	bs, err := b.GobEncode()
	if err != nil {
		return []byte{}
	}
	return bs
}

func (tx *FakeTransaction) GenHash() common.Hash {
	if nil == tx {
		return common.Hash{}
	}
	buffer := bytes.Buffer{}
	if tx.Data != nil {
		buffer.Write(tx.Data)
	}
	buffer.Write(tx.Value.GetBytesWithSign())
	buffer.Write(common.Uint64ToByte(tx.Nonce))
	if tx.Target != nil {
		buffer.Write(tx.Target)
	}
	buffer.WriteByte(byte(tx.Type))
	buffer.Write(tx.GasLimit.GetBytesWithSign())
	buffer.Write(tx.GasPrice.GetBytesWithSign())
	if tx.ExtraData != nil {
		buffer.Write(tx.ExtraData)
	}

	return common.BytesToHash(common.Sha256(buffer.Bytes()))
}

func prepareMsgsEvil(txs []*FakeTransaction) network.Message {
	if len(txs) > 0 {
		txHashs := make([]common.Hash, 0)

		for _, tx := range txs {
			txHashs = append(txHashs, tx.Hash)
		}

		bodyBuf := bytes.NewBuffer([]byte{})
		for _, k := range txHashs {
			bodyBuf.Write(k[:])
		}

		message := network.Message{Code: network.TxSyncNotify, Body: bodyBuf.Bytes()}
		return message
	}
	return network.Message{}
}

func generateEvilTX(price uint64, target string, nonce uint64, value uint64) *FakeTransaction {
	var targetbyte common.Address
	copy(targetbyte[:], []byte(target))
	//fmt.Printf("TSRGETSDDR:%s\n", targetbyte.Hex())
	//fmt.Printf("TSRGETSDDR:%s\n", targetbyte)

	tx := &FakeTransaction{
		GasPrice: NewBigInt(price),
		GasLimit: NewBigInt(5000),
		Target:   []byte(target),
		Nonce:    nonce,
		Value:    NewBigInt(value),
		Type:     -100,
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()
	tx.Sign = append(tx.Sign, 'a')

	source := sk.GetPubKey().GetAddress()
	//tx.Source = source[:][:20]
	//copy(tx.Source[:],source...)
	tx.Source = &source

	return tx
}

func transactionFakeToPb(t *FakeTransaction) *tas_middleware_pb.Transaction {
	if t == nil {
		return nil
	}
	var (
		target []byte
	)
	if t.Target != nil {
		target = t.Target
	}
	tp := int32(t.Type)
	transaction := tas_middleware_pb.Transaction{
		Data:      t.Data,
		Value:     t.Value.GetBytesWithSign(),
		Nonce:     &t.Nonce,
		Target:    target,
		GasLimit:  t.GasLimit.GetBytesWithSign(),
		GasPrice:  t.GasPrice.GetBytesWithSign(),
		Hash:      t.Hash.Bytes(),
		ExtraData: t.ExtraData,
		Type:      &tp,
		Sign:      t.Sign,
	}
	return &transaction
}

func TransactionsFakeToPb(txs []*FakeTransaction) []*tas_middleware_pb.Transaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*tas_middleware_pb.Transaction, 0)
	for _, t := range txs {
		transaction := transactionFakeToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

// MarshalTransactions serialize []*Transaction
func MarshalFakeTransactions(txs []*FakeTransaction) ([]byte, error) {
	transactions := TransactionsFakeToPb(txs)
	transactionSlice := tas_middleware_pb.TransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

func TestFakeTxs(t *testing.T) {
	txs := make([]*FakeTransaction, 0)

	for i := 0; i < 5; i++ {
		tx := generateEvilTX(uint64(1000+i), "111", uint64(1+i), 1)
		txs = append(txs, tx)
	}

	body, e := MarshalFakeTransactions(txs)
	if e != nil {
		t.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
	}

	msg := notify.NewDefaultMessage(body, kindAddr, 0, 0)
	err := TxSyncer.onTxResponse(msg)
	if err != nil {
		fmt.Println(">>>>")
		fmt.Println(err)
	}

	txs1 := getTxPoolTx()
	if len(txs1) != 0 {
		t.Error("fake txs can't be add in the tx pool")
	}

}

func marshallTxs(txs []*types.Transaction, t *testing.T) []byte {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		t.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
	}
	return body
}

func ReadFile(length int) ([]byte, error) {
	content, err := ioutil.ReadFile("./test/lorem.txt")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return content[:length], nil
}

func generateNoValueTx(target string, txType int, gasLimit uint64, gasprice uint64) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.HexToAddress(target)
	tx := &types.Transaction{
		Nonce:    Nonce,
		Target:   &targetbyte,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	return tx
}

func generateNoNonceTx(target string, value uint64, txType int, gasLimit uint64, gasprice uint64) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.HexToAddress(target)
	tx := &types.Transaction{
		//Nonce:     Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	return tx
}

func generateNoTypeTx(target string, value uint64, gasLimit uint64, gasprice uint64) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.HexToAddress(target)
	tx := &types.Transaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	return tx
}

func generateNoGasPriceTx(target string, value uint64, gasLimit uint64, txType int) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.HexToAddress(target)
	tx := &types.Transaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasLimit: types.NewBigInt(gasLimit),
		Type:     int8(txType),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	return tx
}

func generateTX(data []byte, value uint64, nonce uint64, target string, txType int, gasLimit uint64, gasprice uint64, extraData []byte) *types.Transaction {
	if Count == 0 {
		Nonce = 1
	}
	tx := &types.Transaction{}
	//targetbyte := common.Address{common.HexToAddress(target)}
	targetbyte := common.HexToAddress(target)
	if nonce != 0 {
		// a mark
		if nonce == 999999999 {
			tx = &types.Transaction{
				Data:      data,
				Value:     types.NewBigInt(value),
				Nonce:     0,
				Target:    &targetbyte,
				Type:      int8(txType),
				GasLimit:  types.NewBigInt(gasLimit),
				GasPrice:  types.NewBigInt(gasprice),
				ExtraData: extraData,
			}
		} else {
			tx = &types.Transaction{
				Data:      data,
				Value:     types.NewBigInt(value),
				Nonce:     nonce,
				Target:    &targetbyte,
				Type:      int8(txType),
				GasLimit:  types.NewBigInt(gasLimit),
				GasPrice:  types.NewBigInt(gasprice),
				ExtraData: extraData,
			}
		}
	} else {
		if target == "" {
			tx = &types.Transaction{
				Data:      data,
				Value:     types.NewBigInt(value),
				Nonce:     Nonce,
				Type:      int8(txType),
				GasLimit:  types.NewBigInt(gasLimit),
				GasPrice:  types.NewBigInt(gasprice),
				ExtraData: extraData,
			}
		} else {
			tx = &types.Transaction{
				Data:      data,
				Value:     types.NewBigInt(value),
				Nonce:     Nonce,
				Target:    &targetbyte,
				Type:      int8(txType),
				GasLimit:  types.NewBigInt(gasLimit),
				GasPrice:  types.NewBigInt(gasprice),
				ExtraData: extraData,
			}
		}

		Nonce++
		Count++
	}

	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source

	return tx
}

func generateTXs(count int) []*types.Transaction {
	txs := []*types.Transaction{}

	for i := 0; i < count; i++ {
		//hash := sha256.Sum256([]byte(strconv.Itoa(i)))
		//tx := &types.Transaction{Hash: hash}
		//tx := &types.Transaction{
		//	Hash: common.BigToAddress(big.NewInt(int64(i))).Hash(),
		//	Nonce:uint64(i+1),
		//}

		tx := generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(1000+i), uint64(1000+i), nil)
		txs = append(txs, tx)
	}
	var sign = common.BytesToSign(txs[0].Sign)
	pk, err := sign.RecoverPubkey(txs[0].Hash.Bytes())
	if err != nil {
		fmt.Println("generate tx err")
		return nil
	}
	src := pk.GetAddress()
	fmt.Println("SRC:", src.Hex())
	return txs
}

func sendTxToPool(trans *types.Transaction) error {
	if trans.Sign == nil {
		return fmt.Errorf("transaction sign is empty")
	}

	if ok, err := BlockChainImpl.GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		common.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		return err
	}
	return nil
}

func getTxPoolTx() []*types.Transaction {
	txs := BlockChainImpl.GetTransactionPool().GetAllTxs()
	return txs
}

func getNonce(addr string) uint64 {
	realAddr := common.HexToAddress(addr)
	return BlockChainImpl.GetNonce(realAddr)
}

func getBalance(addr string) uint64 {
	realAddr := common.HexToAddress(addr)
	return BlockChainImpl.GetBalance(realAddr).Uint64()
}

func prepareMsgs(txs []*types.Transaction) network.Message {
	if len(txs) > 0 {
		txHashs := make([]common.Hash, 0)

		for _, tx := range txs {
			txHashs = append(txHashs, tx.Hash)
		}

		bodyBuf := bytes.NewBuffer([]byte{})
		for _, k := range txHashs {
			bodyBuf.Write(k[:])
		}

		message := network.Message{Code: network.TxSyncNotify, Body: bodyBuf.Bytes()}
		return message
	}
	return network.Message{}
}

func (bpm *peerManager) genPeer(id string, evil bool) *peerMeter {
	v, exit := bpm.peerMeters.Get(id)
	if !exit {
		var expireTime time.Time
		if evil {
			expireTime = time.Now().Add(time.Hour)
		} else {
			expireTime = time.Now()
		}
		v = &peerMeter{
			id:             id,
			reqBlockCount:  maxReqBlockCount,
			evilExpireTime: expireTime,
		}
		if exit, _ = bpm.peerMeters.ContainsOrAdd(id, v); exit {
			v, _ = bpm.peerMeters.Get(id)
		}
	}
	return v.(*peerMeter)
}

const (
	evilAddr = "0x0000000000000000000000000000000000000000000000000000000000000123"
	kindAddr = "0x0000000000000000000000000000000000000000000000000000000000000abc"
)

var (
	Txs10   []*types.Transaction
	Txs50   []*types.Transaction
	Txs3001 []*types.Transaction

	TxBigData                    *types.Transaction
	TxBigExtraData               *types.Transaction
	TxBigExtraDataAndExtraData   *types.Transaction
	TxDataAndRxtraDataWithSpaces *types.Transaction

	TxOverBalanceValue *types.Transaction
	TxNoValue          *types.Transaction

	TxOverStateNonce0    *types.Transaction
	TxOverStateNonce1000 *types.Transaction
	TxNoNonce            *types.Transaction

	TxTypeTransfer            *types.Transaction
	TxTypeContractCreate      *types.Transaction
	TxTypeContractCreateEvil1 *types.Transaction
	TxTypeContractCreateEvil2 *types.Transaction
	TxTypeContractCall        *types.Transaction
	TxTypeContractCallEvil1   *types.Transaction
	TxTypeContractCallEvil2   *types.Transaction
	TxTypeReward              *types.Transaction
	TxTypeStakeAdd            *types.Transaction
	TxTypeMinerAbort          *types.Transaction
	TxTypeStakeReduce         *types.Transaction
	TxTypeStakeRefund         *types.Transaction
	TxTypeToBeRemoved         *types.Transaction
	TxTypeEvil                *types.Transaction
	TxTypeGroupPiece          *types.Transaction
	//TxTypeGroup
	TxNoType *types.Transaction

	TxLowGasLimit             *types.Transaction
	TxHighGasLimit            *types.Transaction
	TxLowGasPrice             *types.Transaction
	TxHighGasPrice            *types.Transaction
	TxHighGasPriceOverBalance *types.Transaction
	TxNoGasPrice              *types.Transaction

	TxErrHash   *types.Transaction
	TxNilHash   *types.Transaction
	TxLongHash  *types.Transaction
	TxShortHash *types.Transaction

	TxErrSign   *types.Transaction
	TxNilSign   *types.Transaction
	TxLongSign  *types.Transaction
	TxShortSign *types.Transaction

	TxSourceNotExist *types.Transaction
	TxTargetNotExist *types.Transaction
)

var (
	Nonce uint64
	Count int
)

func initPeer() {
	evilPeer := peerManagerImpl.genPeer(evilAddr, true)
	kindPeer := peerManagerImpl.genPeer(kindAddr, false)

	fmt.Printf("evil:%+v\n", evilPeer.evilExpireTime.Hour())
	fmt.Printf("kind:%+v\n", kindPeer.evilExpireTime.Hour())
}

func initTxsAndOthers() {

	initPeer()

	Txs50 = generateTXs(52)
	Txs10 = generateTXs(10)
	//Txs3001 = generateTXs(3001)

	// bad Data and Extra Data
	data, err := ReadFile(64002)
	fmt.Println("DATALEN", len(data))
	if err != nil {
		fmt.Println("read file err:", err)
		return
	}

	// big data tx
	TxBigData = generateTX(data, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// big extradata tx
	TxBigExtraData = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), data)
	// big extradata and data tx
	TxBigExtraDataAndExtraData = generateTX(data[:len(data)/2+1], 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), data[len(data)/2-1:])
	// tx have spaces and other special chars in data and extra data
	TxDataAndRxtraDataWithSpaces = generateTX([]byte("this is a test transaction.!@#$%^&*()<>?:{}+_-=^%#&%@#$^9~#rtg(*)~~~ER>?>||~~!````24111111"), 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), []byte("this is a test transaction.!@#$%^&*()<>?:{}+_-=^%#&%@#$^9~#rtg(*)~~~ER>?>||~~!````24111111"))
	fmt.Println("TxDataAndRxtraDataWithSpaces:", TxDataAndRxtraDataWithSpaces.Nonce)

	// over balance value
	TxOverBalanceValue = generateTX(nil, getBalance(adminAddress)+1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	fmt.Println("TxOverBalanceValue:", TxOverBalanceValue.Nonce)
	// no value
	TxNoValue = generateNoValueTx("111", TransactionTypeTransfer, uint64(500000), uint64(1000))

	// nonce less equal than statenonce,999999999 is just a mark, means the tx's nonce is just 0
	TxOverStateNonce0 = generateTX(nil, 1, 999999999, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// nonce over statenonce + 1000
	TxOverStateNonce1000 = generateTX(nil, 1, getNonce(adminAddress)+1001, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// tx have no nonce
	TxNoNonce = generateNoNonceTx("111", 1, TransactionTypeTransfer, uint64(500000), uint64(1000))

	// different types of transactions
	// transfer
	TxTypeTransfer = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// contract deploy and call normal
	TxTypeContractCreate = generateTX([]byte("sdfa"), 1, 0, "", TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCall = generateTX([]byte("sdfa"), 1, 0, "111", TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	// contract deploy and call evil
	TxTypeContractCreateEvil1 = generateTX(nil, 1, 0, "", TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCreateEvil2 = generateTX([]byte("sdfa"), 1, 0, "111", TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCallEvil1 = generateTX(nil, 1, 0, "111", TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	TxTypeContractCallEvil2 = generateTX([]byte("sdfa"), 1, 0, "", TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	// stake add
	//TxTypeStakeAdd = generateTX([]byte("sdfa"), 1, 0, "111", TransactionTypeStakeAdd, uint64(500000), uint64(1000), nil)

	// evil type of tx
	TxTypeEvil = generateTX(nil, 1, 0, "111", 99, uint64(500000), uint64(1000), nil)
	// no type tx
	TxNoType = generateNoTypeTx("111", 1, uint64(500000), uint64(1000))

	//about gas limit and gas price
	TxLowGasLimit = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(399), uint64(1000), nil)
	TxHighGasLimit = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500001), uint64(1000), nil)
	TxLowGasPrice = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(399), nil)
	TxHighGasPrice = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(100000000000), nil)
	TxHighGasPriceOverBalance = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000000000000), nil)

	// bad hash tx
	TxErrHash = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxErrHash.Hash = common.HexToHash(string(reverse(TxErrHash.Hash[:])))
	TxNilHash = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxNilHash.Hash = common.HexToHash(string(""))
	TxLongHash = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxLongHash.Hash = common.HexToHash(string(append(TxLongHash.Hash[:], TxLongHash.Hash[:]...)))
	TxShortHash = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxShortHash.Hash = common.HexToHash(string(TxShortHash.Hash[:][:len(TxShortHash.Hash)]))

	// bad sign tx
	TxErrSign = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxErrSign.Sign = reverse(TxErrSign.Sign)
	TxNilSign = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxNilSign.Sign = nil
	TxLongSign = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxLongSign.Sign = append(TxLongSign.Sign, TxLongSign.Sign...)
	TxShortSign = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxShortSign.Sign = TxShortHash.Sign[:len(TxShortSign.Sign)]

	// a not exist source addr
	TxSourceNotExist = generateTX(nil, 1, 0, "111", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	source1 := common.HexToAddress("0x0000000000000000000000000000000000000000000000000000000000000123")
	TxSourceNotExist.Source = &source1

	TxTargetNotExist = generateTX(nil, 1, 0, "", TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	//TxTargetNotExist.Target = nil

}

func TestTxPool(t *testing.T) {

	//fmt.Println("IMPL:", BlockChainImpl)
	//_, err := BlockChainImpl.LatestStateDB()
	//if err != nil {
	//	fmt.Println(err)
	//	t.Fatalf("get status failed")
	//}

	Txs10 := generateTXs(10)
	for _, tx := range Txs10 {
		err := sendTxToPool(tx)
		if err != nil {
			fmt.Println(err.Error())
			t.Error("ADD TX FAIL")
		}
	}

	txs := getTxPoolTx()
	fmt.Println(len(txs))
	fmt.Println("finish")
}

func TestOnTxNotify(t *testing.T) {
	initTxsAndOthers()
	var Txs10Hashes []common.Hash
	for _, tx := range Txs10 {
		Txs10Hashes = append(Txs10Hashes, tx.Hash)
	}

	var Txs50Hashes []common.Hash
	for _, tx := range Txs50 {
		Txs50Hashes = append(Txs50Hashes, tx.Hash)
	}

	message10 := prepareMsgs(Txs10)
	fmt.Println("len10:", len(message10.Body))
	message50 := prepareMsgs(Txs50)
	fmt.Println("len50:", len(message50.Body))

	// 01. count < txMaxNotifyPerTime(50) && evil address
	defaultMsg01 := notify.NewDefaultMessage(message10.Body, evilAddr, 0, 0)

	// 02. count > txMaxNotifyPerTime(50) && evil address
	defaultMsg02 := notify.NewDefaultMessage(message50.Body, evilAddr, 0, 0)

	// 03. count < txMaxNotifyPerTime(50) && kind address
	defaultMsg03 := notify.NewDefaultMessage(message10.Body, kindAddr, 0, 0)

	// 04. count > txMaxNotifyPerTime(50) && kind address
	defaultMsg04 := notify.NewDefaultMessage(message50.Body, kindAddr, 0, 0)

	// 05. body is nil && kind address
	defaultMsg05 := notify.NewDefaultMessage(nil, kindAddr, 0, 0)

	// 06. body is a error msg && kind address
	fakeTxs := []*types.Transaction{
		{
			Hash: common.HexToHash("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"),
		},
		{
			Hash: common.HexToHash("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz"),
		},
		{
			Hash: common.HexToHash("0xabcd"),
		},
	}
	fakeHashes := prepareMsgs(fakeTxs)
	fakeBody := bytes.Join([][]byte{fakeHashes.Body[:64], fakeHashes.Body[94:]}, []byte(""))
	fmt.Printf("fake:%x\n", fakeBody)
	fmt.Printf("fake:%s\n", string(fakeBody))
	defaultMsg06 := notify.NewDefaultMessage(fakeBody, kindAddr, 0, 0)

	// 07  body is a bad hash msg && kind address
	hash := []byte("acf5abbba560952636719f0zzzzzwxyy")
	var destHash [32]byte
	copy(destHash[:], hash)
	defaultMsg07 := notify.NewDefaultMessage(hash, kindAddr, 0, 0)

	err := TxSyncer.onTxNotify(defaultMsg01)
	if err == nil {
		fmt.Println("MSG01:", err)
		t.Error("defaultMsg01 should be fail")
	}

	err = TxSyncer.onTxNotify(defaultMsg02)
	if err == nil {
		t.Errorf("defaultMsg02 should be fail")
	}

	err = TxSyncer.onTxNotify(defaultMsg03)
	if err != nil {
		fmt.Println("MSG03:", err)
		t.Errorf("defaultMsg03 should be success")
	}

	err = TxSyncer.onTxNotify(defaultMsg04)
	if err == nil {
		fmt.Println("MSG04:", err)
		t.Errorf("defaultMsg04 should be fail")
	}

	err = TxSyncer.onTxNotify(defaultMsg05)
	if err != nil {
		fmt.Println("MSG05:", err)
		t.Errorf("defaultMsg05 should be success")
	}

	err = TxSyncer.onTxNotify(defaultMsg06)
	if err != nil {
		fmt.Println("MSG06:", err)
		t.Errorf("defaultMsg06 should be success")
	}

	err = TxSyncer.onTxNotify(defaultMsg07)
	if err != nil {
		fmt.Println("MSG07:", err)
		t.Errorf("defaultMsg07 should be success")
	}

	isExist := TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"))
	if !isExist {
		t.Errorf("tx %s should exist in cache", "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103")
	}

	// 0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz -> 0x0000000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0
	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz"))
	if !isExist {
		t.Errorf("tx %s should exist in cache", "0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz")

	}

	// 0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz -> 0x0000000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0
	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("0x0000000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0"))
	if !isExist {
		t.Errorf("tx %s should exist in cache", "0x0000000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0")

	}

	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("0xabcd"))
	if isExist {
		t.Errorf("tx %s should not exist in cache", "0xabcd")
	}

	// Due to the way the byte is read, "0xabcd" is actually equal to "0xabcd000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0"
	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("0xabcd000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0"))
	if isExist {
		t.Errorf("tx %s should not exist in cache", "0xabcd000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0")
	}

	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(destHash)
	if !isExist {
		t.Errorf("bytes %s should in the cache, although it's not a standard hex", destHash)
	}
	isExist = TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source()).hasHash(common.HexToHash("61636635616262626135363039353236333637313966307a7a7a7a7a77787979"))
	if !isExist {
		t.Errorf("bytes %s should in the cache, %s is equal to %s", "61636635616262626135363039353236333637313966307a7a7a7a7a77787979", destHash, "61636635616262626135363039353236333637313966307a7a7a7a7a77787979")
	}

	//fmt.Println(">>>>>")
	//fmt.Println(common.HexToHash("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f05zzzzzwxyz").Hex())
	//fmt.Println(common.HexToHash("0x0000000000c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0").Hex())
	//fmt.Println(common.HexToHash("c2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f0").Hex())

	kindCandidateKeys := TxSyncer.getOrAddCandidateKeys(defaultMsg03.Source())
	for k, hash := range Txs50Hashes {
		if kindCandidateKeys.hasHash(hash) && k >= 10 {
			//t.Log("success")
			t.Errorf("kindCandidateKeys only have 10 txhashes")
		}
	}

	evilCandidateKeys := TxSyncer.getOrAddCandidateKeys(defaultMsg02.Source())
	for _, hash := range Txs10Hashes {
		if evilCandidateKeys.hasHash(hash) {
			t.Errorf("evilCandidateKeys is a evil address, should have no txhashes in it")
		}
	}

	peer02IsExist := peerManagerImpl.isPeerExists(defaultMsg02.Source())
	peer03IsExist := peerManagerImpl.isPeerExists(defaultMsg03.Source())
	peerFakeIsExist := peerManagerImpl.isPeerExists("0x12345")
	if !peer02IsExist || !peer03IsExist {
		t.Errorf("source %s and %s should in the cache", defaultMsg02.Source(), defaultMsg03.Source())
	}
	if peerFakeIsExist {
		t.Errorf("source %s shouldn't exist in the cache", "0x12345")
	}
}

func TestOnTxResponse(t *testing.T) {
	initTxsAndOthers()

	body10 := marshallTxs(Txs10, t)
	//body50 := marshallTxs(Txs50, t)
	//body3001 := marshallTxs(Txs3001, t)

	bodyBigData := marshallTxs([]*types.Transaction{TxBigData}, t)
	bodyBigExtraData := marshallTxs([]*types.Transaction{TxBigExtraData}, t)
	bodyBigExtraDataAndExtraData := marshallTxs([]*types.Transaction{TxBigExtraDataAndExtraData}, t)
	bodyDataAndRxtraDataWithSpaces := marshallTxs([]*types.Transaction{TxDataAndRxtraDataWithSpaces}, t)

	bodyOverBalanceValue := marshallTxs([]*types.Transaction{TxOverBalanceValue}, t)
	bodyTxNoValue := marshallTxs([]*types.Transaction{TxNoValue}, t)

	bodyOverStateNonce1000 := marshallTxs([]*types.Transaction{TxOverStateNonce1000}, t)
	bodyOverStateNonce0 := marshallTxs([]*types.Transaction{TxOverStateNonce0}, t)
	bodyTxNoNonce := marshallTxs([]*types.Transaction{TxNoNonce}, t)

	bodyTxTypeTransfer := marshallTxs([]*types.Transaction{TxTypeTransfer}, t)
	bodyTxTypeContractCreate := marshallTxs([]*types.Transaction{TxTypeContractCreate}, t)
	bodyTxTypeContractCreateEvil1 := marshallTxs([]*types.Transaction{TxTypeContractCreateEvil1}, t)
	bodyTxTypeContractCreateEvil2 := marshallTxs([]*types.Transaction{TxTypeContractCreateEvil2}, t)
	bodyTxTypeContractCall := marshallTxs([]*types.Transaction{TxTypeContractCall}, t)
	bodyTypeContractCallEvil1 := marshallTxs([]*types.Transaction{TxTypeContractCallEvil1}, t)
	bodyTypeContractCallEvil2 := marshallTxs([]*types.Transaction{TxTypeContractCallEvil2}, t)

	bodyTxTypeEvil := marshallTxs([]*types.Transaction{TxTypeEvil}, t)
	bodyTxNoType := marshallTxs([]*types.Transaction{TxNoType}, t)

	bodyTxLowGasLimit := marshallTxs([]*types.Transaction{TxLowGasLimit}, t)
	bodyTxHighGasLimit := marshallTxs([]*types.Transaction{TxHighGasLimit}, t)
	bodyTxLowGasPrice := marshallTxs([]*types.Transaction{TxLowGasPrice}, t)
	bodyTxHighGasPrice := marshallTxs([]*types.Transaction{TxHighGasPrice}, t)
	bodyTxHighGasPriceOverBalance := marshallTxs([]*types.Transaction{TxHighGasPriceOverBalance}, t)

	bodyTxErrHash := marshallTxs([]*types.Transaction{TxErrHash}, t)
	bodyTxErrHashNil := marshallTxs([]*types.Transaction{TxNilHash}, t)
	bodyTxLongHash := marshallTxs([]*types.Transaction{TxLongHash}, t)
	bodyTxShortHash := marshallTxs([]*types.Transaction{TxShortHash}, t)

	bodyTxErrSign := marshallTxs([]*types.Transaction{TxErrSign}, t)
	bodyTxNilSign := marshallTxs([]*types.Transaction{TxNilSign}, t)
	bodyTxLongSign := marshallTxs([]*types.Transaction{TxLongSign}, t)
	bodyTxShortSign := marshallTxs([]*types.Transaction{TxShortSign}, t)

	bodyTxSourceNotExist := marshallTxs([]*types.Transaction{TxSourceNotExist}, t)

	bodyTxTargetNotExist := marshallTxs([]*types.Transaction{TxTargetNotExist}, t)

	//bodyTypeypeStakeAdd := marshallTxs([]*types.Transaction{TxTypeStakeAdd}, t)

	goodMsg := []*notify.DefaultMessage{
		// 10 txs and kind addr
		notify.NewDefaultMessage(body10, kindAddr, 0, 0),

		// tx with sepcial chars and spaces in data and extra data
		notify.NewDefaultMessage(bodyDataAndRxtraDataWithSpaces, kindAddr, 0, 0),

		// todo?
		// tx with over balance value
		notify.NewDefaultMessage(bodyOverBalanceValue, kindAddr, 0, 0),

		// transfer tx
		notify.NewDefaultMessage(bodyTxTypeTransfer, kindAddr, 0, 0),

		// contract create tx
		notify.NewDefaultMessage(bodyTxTypeContractCreate, kindAddr, 0, 0),

		// contract call tx
		notify.NewDefaultMessage(bodyTxTypeContractCall, kindAddr, 0, 0),

		notify.NewDefaultMessage(bodyTxNoType, kindAddr, 0, 0),

		// stake add
		//notify.NewDefaultMessage(bodyTypeypeStakeAdd, kindAddr, 0, 0),

		// high gas pirce
		notify.NewDefaultMessage(bodyTxHighGasPrice, kindAddr, 0, 0),

		// todo even source is a fake source there's no relations
		notify.NewDefaultMessage(bodyTxSourceNotExist, kindAddr, 0, 0),
	}

	bodyCountsMap := map[*notify.DefaultMessage]int{
		goodMsg[0]: len(Txs10),
		goodMsg[1]: len([]*types.Transaction{TxDataAndRxtraDataWithSpaces}),
		goodMsg[2]: len([]*types.Transaction{TxOverBalanceValue}),
		goodMsg[3]: len([]*types.Transaction{TxTypeTransfer}),
		goodMsg[4]: len([]*types.Transaction{TxTypeContractCreate}),
		goodMsg[5]: len([]*types.Transaction{TxTypeContractCall}),
		goodMsg[6]: len([]*types.Transaction{TxNoType}),
		goodMsg[7]: len([]*types.Transaction{TxHighGasPrice}),
		goodMsg[8]: len([]*types.Transaction{TxSourceNotExist}),
	}

	// good txs
	t.Run("good txs", func(t *testing.T) {
		var poolLenBefore int
		var poolLenAfter int
		for k, msg := range goodMsg {
			poolLenBefore = len(getTxPoolTx())
			err := TxSyncer.onTxResponse(msg)
			poolLenAfter = len(getTxPoolTx())
			fmt.Printf("%d:", k)
			fmt.Println("BeFORE:", poolLenBefore, "bodyCountsMap:", bodyCountsMap[msg], "poolLenAfter:", poolLenAfter)
			if poolLenBefore+bodyCountsMap[msg] != poolLenAfter {
				t.Errorf("No.%d good exexute result is not what is expected，please check manually!", k)
			}
			if err != nil {
				fmt.Println(">err:", err)
				t.Errorf("No.%d good exexute result should success and retrun a nil error，please check manually!", k)
			}
		}
		fmt.Println("FIANL:", len(getTxPoolTx()))
	})

	badMsg := []*notify.DefaultMessage{
		// 反转body
		notify.NewDefaultMessage(reverse(body10), kindAddr, 0, 0),

		// 给body添加特殊字符
		notify.NewDefaultMessage(addBytes(body10), kindAddr, 0, 0),

		// 给body减少字符
		notify.NewDefaultMessage(cutBytes(body10), kindAddr, 0, 0),

		// tx counts out of limit
		//notify.NewDefaultMessage(body3001, kindAddr, 0, 0),

		// evil addr
		notify.NewDefaultMessage(body10, evilAddr, 0, 0),
	}

	//返回错误，不加入交易池
	t.Run("make bad msg bodies", func(t *testing.T) {
		for k, msg := range badMsg {
			err := TxSyncer.onTxResponse(msg)
			fmt.Println(err)
			if err == nil {
				t.Errorf("No.%d badMsg exexute result is not what is expected，please check manually!", k)
			}
		}
	})

	// bad tx, and can't return err,can check counts of tx in the tx pool
	badMsg02 := []*notify.DefaultMessage{
		// big data tx
		notify.NewDefaultMessage(bodyBigData, kindAddr, 0, 0),

		// big extra data tx
		notify.NewDefaultMessage(bodyBigExtraData, kindAddr, 0, 0),

		// big extra data tx and data
		notify.NewDefaultMessage(bodyBigExtraDataAndExtraData, kindAddr, 0, 0),

		// tx have no value
		notify.NewDefaultMessage(bodyTxNoValue, kindAddr, 0, 0),

		// tx with less equal than state nonce
		notify.NewDefaultMessage(bodyOverStateNonce0, kindAddr, 0, 0),

		// tx with over 1000 compare nonce of states
		notify.NewDefaultMessage(bodyOverStateNonce1000, kindAddr, 0, 0),

		// body have no tx nonce
		notify.NewDefaultMessage(bodyTxNoNonce, kindAddr, 0, 0),

		// tx contract deploy evil
		notify.NewDefaultMessage(bodyTxTypeContractCreateEvil1, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeContractCreateEvil2, kindAddr, 0, 0),

		// tx contract call evil
		notify.NewDefaultMessage(bodyTypeContractCallEvil1, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTypeContractCallEvil2, kindAddr, 0, 0),

		// tx about gas limit and gas price
		notify.NewDefaultMessage(bodyTxLowGasLimit, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxHighGasLimit, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxLowGasPrice, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxHighGasPriceOverBalance, kindAddr, 0, 0),

		// err hash
		notify.NewDefaultMessage(bodyTxErrHash, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxErrHashNil, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxLongHash, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxShortHash, kindAddr, 0, 0),

		// err sign
		notify.NewDefaultMessage(bodyTxErrSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxNilSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxLongSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxShortSign, kindAddr, 0, 0),

		// todo
		notify.NewDefaultMessage(bodyTxTypeEvil, kindAddr, 0, 0),

		notify.NewDefaultMessage(bodyTxTargetNotExist, kindAddr, 0, 0),
	}

	//不返回错误，不加入交易池
	t.Run("make bad msg bodies-badMsg02", func(t *testing.T) {
		poolLenBefore := len(getTxPoolTx())
		fmt.Println("poolLenBefore:", poolLenBefore)
		for k, msg := range badMsg02 {
			err := TxSyncer.onTxResponse(msg)
			fmt.Printf("%d:make bad msg bodies err:%v\n", k, err)
		}
		poolLenAfter := len(getTxPoolTx())
		fmt.Println("poolLenAfter:", poolLenAfter)
		if poolLenBefore != poolLenAfter {
			t.Error("bad tx shouldn't add into txpool")
		}
	})

}

func reverse(s []byte) []byte {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s
}

func addBytes(s []byte) []byte {
	badBody := append(s, []byte("123abc how are you?#@$!@$+_)(*&^%$#@@@=-09876522?><,./';[]}{{}~``")...)
	return badBody
}

func cutBytes(s []byte) []byte {
	badBody := s[:len(s)-1]
	return badBody
}
