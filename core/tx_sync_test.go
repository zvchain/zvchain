package core

import (
	"bytes"
	"fmt"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/ticker"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/tasdb"
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
)

func generateTX(price uint64, target string, nonce uint64, value uint64) *types.Transaction {

	targetbyte := common.BytesToAddress(genHash(target))

	tx := &types.Transaction{
		GasPrice: types.NewBigInt(price),
		GasLimit: types.NewBigInt(5000),
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    types.NewBigInt(value),
	}
	tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminAddress)
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	source := sk.GetPubKey().GetAddress()
	tx.Source = &source

	return tx
}

func generateTXs(count int, t *testing.T) []*types.Transaction {
	txs := []*types.Transaction{}

	for i := 0; i < count; i++ {
		//hash := sha256.Sum256([]byte(strconv.Itoa(i)))
		//tx := &types.Transaction{Hash: hash}
		//tx := &types.Transaction{
		//	Hash: common.BigToAddress(big.NewInt(int64(i))).Hash(),
		//	Nonce:uint64(i+1),
		//}
		tx := generateTX(uint64(1000+i), "111", uint64(i+1), 1)
		txs = append(txs, tx)
	}
	var sign = common.BytesToSign(txs[0].Sign)
	pk, err := sign.RecoverPubkey(txs[0].Hash.Bytes())
	if err != nil {
		t.Fatalf("error")
	}
	src := pk.GetAddress()
	fmt.Println("SRC:", src.Hex())
	//accountDB, err := BlockChainImpl.LatestStateDB()
	//if err != nil {
	//	fmt.Println(err)
	//	t.Fatalf("get status failed")
	//}
	//accountDB.AddBalance(src, new(big.Int).SetUint64(111111111111111111))
	//balance := accountDB.GetBalance(src)
	//fmt.Println("BALANCE:",balance)
	return txs
}

func sendTxToPool(trans *types.Transaction) error {
	if trans.Sign == nil {
		return fmt.Errorf("transaction sign is empty")
	}

	if ok, err := BlockChainImpl.GetTransactionPool().AddTransaction(trans); err != nil || !ok {
		log.DefaultLogger.Errorf("AddTransaction not ok or error:%s", err.Error())
		return err
	}
	return nil
}

func getTxPoolTx() []*types.Transaction {
	txs := BlockChainImpl.GetTransactionPool().GetReceived()
	return txs
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

//func init() {
//	initPeerManager()
//	err := initContext4Test()
//	if err != nil {
//		fmt.Println(err)
//	}
//	//newTransactionPoolTest(nil, nil)
//	//initTxSyncerTest(nil, txPoolImpl.(*txPool))
//}
const (
	evilAddr = "0x0000000000000000000000000000000000000000000000000000000000000123"
	kindAddr = "0x0000000000000000000000000000000000000000000000000000000000000abc"
)

func TestTxPool(t *testing.T) {

	fmt.Println("IMPL:", BlockChainImpl)
	_, err := BlockChainImpl.LatestStateDB()
	if err != nil {
		fmt.Println(err)
		t.Fatalf("get status failed")
	}

	//Txs10 := generateTXs(10, t)
	//for _, tx := range Txs10 {
	//	err := sendTxToPool(tx)
	//	if err != nil {
	//		fmt.Println(err.Error())
	//		t.Error("ADD TX FAIL")
	//	}
	//}
	//
	//txs := getTxPoolTx()
	//for _, tx := range txs{
	//	fmt.Printf("%v",tx)
	//}
	//fmt.Println("finish")
}

func TestOnTxNotify(t *testing.T) {

	Txs10 := generateTXs(10, t)
	Txs50 := generateTXs(52, t)
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

	evilPeer := peerManagerImpl.genPeer(evilAddr, true)
	kindPeer := peerManagerImpl.genPeer(kindAddr, false)

	fmt.Printf("evil:%+v\n", evilPeer.evilExpireTime.Hour())
	fmt.Printf("kind:%+v\n", kindPeer.evilExpireTime.Hour())

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
	copy(destHash[:], []byte("acf5abbba560952636719f0zzzzzwxyy"))
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
	Txs50 := generateTXs(52, t)
	body, e := types.MarshalTransactions(Txs50)
	if e != nil {
		t.Errorf("Discard MarshalTransactions because of marshal error:%s!", e.Error())
	}
	msg := notify.NewDefaultMessage(body, kindAddr, 0, 0)
	err := TxSyncer.onTxResponse(msg)
	if err != nil {
		fmt.Println(err.Error())
	}
}
