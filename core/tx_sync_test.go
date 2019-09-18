package core

import (
	"bytes"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/vmihailenco/msgpack"
	"github.com/zvchain/zvchain/common"
	"github.com/zvchain/zvchain/consensus/base"
	"github.com/zvchain/zvchain/consensus/groupsig"
	"github.com/zvchain/zvchain/core/group"
	"github.com/zvchain/zvchain/log"
	"github.com/zvchain/zvchain/middleware/notify"
	"github.com/zvchain/zvchain/middleware/pb"
	"github.com/zvchain/zvchain/middleware/types"
	"github.com/zvchain/zvchain/network"
	"github.com/zvchain/zvchain/storage/account"
	"io/ioutil"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func TestCheckReceivedHashesInHitRate(t *testing.T) {
	defer clearSelf(t)
	init4TxSync(t)

	peer := newPeerTxsKeys()
	txs := makeOrderTransactions()
	for i := 0; i < len(txs); i++ {
		peer.addSendHash(txs[i].Hash)
	}

	isSuccess := peer.checkReceivedHashesInHitRate(txs)
	if !isSuccess {
		t.Fatalf("except success,but got failed!")
	}

	changeHalfTxsToInvaildHashs(txs)

	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if !isSuccess {
		t.Fatalf("except success,but got failed!")
	}
	txlen := len(txs)

	txs[txlen-1].Hash = common.BigToAddress(big.NewInt(int64(300000))).Hash()
	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if isSuccess {
		t.Fatalf("except success,but got failed!")
	}
	changeHashsToSame(txs)

	isSuccess = peer.checkReceivedHashesInHitRate(txs)
	if isSuccess {
		t.Fatalf("except success,but got failed!")
	}
}

func changeHalfTxsToInvaildHashs(txs []*types.Transaction) {
	for i := 0; i < len(txs)/2; i++ {
		txs[i].Hash = common.BigToAddress(big.NewInt(int64(i + 30000))).Hash()
	}
}

func changeHashsToSame(txs []*types.Transaction) {
	for i := 0; i < len(txs); i++ {
		txs[i].Hash = common.BigToAddress(big.NewInt(int64(1))).Hash()
	}
}

func makeOrderTransactions() []*types.Transaction {
	txs := []*types.Transaction{}
	for i := 1; i <= 200; i++ {
		tx := &types.Transaction{Hash: common.BigToAddress(big.NewInt(int64(i))).Hash()}
		txs = append(txs, tx)
	}
	return txs
}

//DefaultMessage is a default implementation of the Message interface.
//It can meet most of demands abort chain event
type DefaultMessage struct {
	body            []byte
	source          string
	chainID         uint16
	protocalVersion uint16
}

func (m *DefaultMessage) GetRaw() []byte {
	panic("implement me")
}

func (m *DefaultMessage) GetData() interface{} {
	return m.Body
}

func (m *DefaultMessage) Body() []byte {
	return m.body
}

func (m *DefaultMessage) Source() string {
	return m.source
}

/******************************************************/
const (
	adminAddress    = "zvbd10ac1f9a60c81fa2bc60d20985a31d53c49903271ddd5cccfdbaac5bc1e69c"
	adminPublicKey  = "0x0445d9b84bc3bd71f58ad2b1d907c5d12cc29bf59a460f3a60636150d043580072b5ab87fa5a9e5a0d071839abe34e8333f6cd49741f792714e6360b54a1c6fa80"
	adminPrivateKey = "0x0445d9b84bc3bd71f58ad2b1d907c5d12cc29bf59a460f3a60636150d043580072b5ab87fa5a9e5a0d071839abe34e8333f6cd49741f792714e6360b54a1c6fa808fbbc6713ff2fd8714539d29f5f4a04111a6b657bcdfb73cfd067913d6d84e29"
	adminBPK        = "0x1526e13ecab389c5e69f9c84505a9034b228495d3da0ceebd12da1defc99ade51ecac0103760355b9b1c14aa61a6f6cc2ba0e3dcdf1fe0b40655ed1b8e98b4a2182b46ff5a0c9e1eb3b988e38537016b21df9690e186509e4f348de36f3431bd0d43a51c6a213a02cfc072488c223a381404f5f3fe49b99b568db0c44ccc8d50"
	adminBSK        = "0xf2b358b7a12ba600d723df3e48ccd033e8c43b3815e05ac43a88788076eea5"
	adminVRFPK      = "0xdd89484c8a79d8beccea4f0f816034ca7d152bb0a51688ef0e86a4cd7dd8dc50"
	adminVRFSK      = "0x00f2b358b7a12ba600d723df3e48ccd033e8c43b3815e05ac43a88788076eea5dd89484c8a79d8beccea4f0f816034ca7d152bb0a51688ef0e86a4cd7dd8dc50"
)

const (
	evilAddr       = "zv0000000000000000000000000000000000000000000000000000000000000123"
	kindAddr       = "zv0000000000000000000000000000000000000000000000000000000000000abc"
	kindToEvilAddr = "zv0000000000000000000000000000000000000000000000000000000000000fff"
)

var (
	NetImpl      *NetTest
	OnReqTxsTest []*types.Transaction
)

var (
	Txs10   []*types.Transaction
	Txs50   []*types.Transaction
	Txs2000 []*types.Transaction
	Txs3000 []*types.Transaction

	TxNilBody                    *types.Transaction
	TxBigData                    *types.Transaction
	TxBigExtraData               *types.Transaction
	TxBigExtraDataAndExtraData   *types.Transaction
	TxDataAndRxtraDataWithSpaces *types.Transaction

	TxOverBalanceValue *types.Transaction
	TxNoValue          *types.Transaction

	TxOverStateNonce0    *types.Transaction
	TxOverStateNonce1000 *types.Transaction
	TxNoNonce            *types.Transaction

	TxTypeTransfer             *types.Transaction
	TxTypeContractCreate       *types.Transaction
	TxTypeContractCreateEvil1  *types.Transaction
	TxTypeContractCreateEvil2  *types.Transaction
	TxTypeContractCall         *types.Transaction
	TxTypeContractCallEvil1    *types.Transaction
	TxTypeContractCallEvil2    *types.Transaction
	TxTypeRewardBadData        *types.Transaction
	TxTypeRewardBadExtra       *types.Transaction
	TxTypeStakeAddProposal     *types.Transaction
	TxTypeStakeAddVerify       *types.Transaction
	TxTypeStakeAddFakes        []*types.Transaction
	TxTypeStakeAddFake1        *types.Transaction
	TxTypeStakeAddFake2        *types.Transaction
	TxTypeStakeAddFake3        *types.Transaction
	TxTypeStakeAddFake4        *types.Transaction
	TxTypeStakeAddFake5        *types.Transaction
	TxTypeStakeReduce          *types.Transaction
	TxTypeStakeReduceFake1     *types.Transaction
	TxTypeMinerAbort           *types.Transaction
	TxTypeStakeRefund          *types.Transaction
	TxTypeEvil                 *types.Transaction
	TxTypeGroupPiece           *types.Transaction
	TxTypeGroupPieceBadData    *types.Transaction
	TxTypeGroupPieceNilData    *types.Transaction
	TxTypeGroupPieceWithTarget *types.Transaction
	//TxTypeGroup
	TxNoType *types.Transaction

	TxLowGasLimit             *types.Transaction
	TxHighGasLimit            *types.Transaction
	TxLowGasPrice             *types.Transaction
	TxHighGasPrice            *types.Transaction
	TxHighGasPriceOverBalance *types.Transaction
	TxNoGasPrice              *types.Transaction
	TxNoGasLimit              *types.Transaction

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

func initNetwork() {
	net := new(NetTest)
	NetImpl = net
}

func init4TxSync(t *testing.T) {
	initContext4Test(t)
	initPeerManager()
	types.InitMiddleware()
	initNetwork()
	initTxSyncer(BlockChainImpl, BlockChainImpl.GetTransactionPool().(*txPool), NetImpl)
	//network.Init(nil,nil,nil)

	types.DefaultPVFunc = PvFuncTest
	//
	evilPeer := peerManagerImpl.genPeer(evilAddr, true)
	kindPeer := peerManagerImpl.genPeer(kindAddr, false)

	fmt.Printf("evil:%+v\n", evilPeer.evilExpireTime.Hour())
	fmt.Printf("kind:%+v\n", kindPeer.evilExpireTime.Hour())
	AddBalance()
}

func initTxsAndOthers(t *testing.T) {

	init4TxSync(t)

	Txs50 = generateTXs(52, false)
	Txs10 = generateTXs(10, false)

	// bad Data and Extra Data
	data, err := ReadFile(64002)
	fmt.Println("DATALEN", len(data))
	if err != nil {
		fmt.Println("read file err:", err)
		return
	}

	// big data tx
	TxBigData = generateTX(data, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// big extradata tx
	TxBigExtraData = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), data)
	// big extradata and data tx
	TxBigExtraDataAndExtraData = generateTX(data[:len(data)/2+1], 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), data[len(data)/2-1:])
	// tx have spaces and other special chars in data and extra data
	TxDataAndRxtraDataWithSpaces = generateTX([]byte("this is a test transaction.!@#$%^&*()<>?:{}+_-=^%#&%@#$^9~#rtg(*)~~~ER>?>||~~!````24111111"), 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), []byte("this is a test transaction.!@#$%^&*()<>?:{}+_-=^%#&%@#$^9~#rtg(*)~~~ER>?>||~~!````24111111"))
	fmt.Println("TxDataAndRxtraDataWithSpaces:", TxDataAndRxtraDataWithSpaces.Nonce)

	// over balance value
	TxOverBalanceValue = generateTX(nil, getBalance(adminAddress)+1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	fmt.Println("TxOverBalanceValue:", TxOverBalanceValue.Nonce)
	// no value
	TxNoValue = generateNoValueTx("111", types.TransactionTypeTransfer, uint64(500000), uint64(1000))

	// nonce less equal than statenonce,999999999 is just a mark, means the tx's nonce is just 0
	TxOverStateNonce0 = generateTX(nil, 1, 999999999, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// nonce over statenonce + 1000
	TxOverStateNonce1000 = generateTX(nil, 1, getNonce(adminAddress)+1001, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// tx have no nonce
	TxNoNonce = generateNoNonceTx("111", 1, types.TransactionTypeTransfer, uint64(500000), uint64(1000))

	// different types of transactions
	// transfer
	TxTypeTransfer = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// contract deploy and call normal
	TxTypeContractCreate = generateTX([]byte("sdfa"), 1, 0, "", types.TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCall = generateTX([]byte("sdfa"), 1, 0, "111", types.TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	// contract deploy and call evil
	TxTypeContractCreateEvil1 = generateTX(nil, 1, 0, "", types.TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCreateEvil2 = generateTX([]byte("sdfa"), 1, 0, "111", types.TransactionTypeContractCreate, uint64(500000), uint64(1000), nil)
	TxTypeContractCallEvil1 = generateTX(nil, 1, 0, "111", types.TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	TxTypeContractCallEvil2 = generateTX([]byte("sdfa"), 1, 0, "", types.TransactionTypeContractCall, uint64(500000), uint64(1000), nil)
	// stake add
	TxTypeStakeAddProposal = generateStakeAddTx(1, adminAddress, types.TransactionTypeStakeAdd, uint64(500000), uint64(1000), types.MinerTypeProposal)
	TxTypeStakeAddVerify = generateStakeAddTx(1, adminAddress, types.TransactionTypeStakeAdd, uint64(500000), uint64(1000), types.MinerTypeVerify)
	// stake add fake
	TxTypeStakeAddFakes = generateFakeStakeAddTxs(1, adminAddress, types.TransactionTypeStakeAdd, uint64(500000), uint64(1000), types.MinerTypeVerify)
	TxTypeStakeAddFake1 = TxTypeStakeAddFakes[0]
	TxTypeStakeAddFake2 = TxTypeStakeAddFakes[1]
	TxTypeStakeAddFake3 = TxTypeStakeAddFakes[2]
	TxTypeStakeAddFake4 = TxTypeStakeAddFakes[3]
	TxTypeStakeAddFake5 = TxTypeStakeAddFakes[4]
	// stake reduce
	TxTypeStakeReduce = genetateMinerOpTx(1, adminAddress, types.TransactionTypeStakeReduce, uint64(500000), uint64(1000), types.MinerTypeVerify, false)
	TxTypeStakeReduceFake1 = genetateMinerOpTx(1, adminAddress, types.TransactionTypeStakeReduce, uint64(500000), uint64(1000), types.MinerTypeVerify, true)

	TxTypeMinerAbort = genetateMinerOpTx(1, adminAddress, types.TransactionTypeMinerAbort, uint64(500000), uint64(1000), types.MinerTypeVerify, false)
	TxTypeStakeRefund = genetateMinerOpTx(1, adminAddress, types.TransactionTypeStakeRefund, uint64(500000), uint64(1000), types.MinerTypeVerify, false)
	// group piece
	TxTypeGroupPiece = generateGroupTx(1, types.TransactionTypeGroupPiece, uint64(500000), uint64(1000), types.MinerTypeVerify, false, false, false)
	TxTypeGroupPieceBadData = generateGroupTx(1, types.TransactionTypeGroupPiece, uint64(500000), uint64(1000), types.MinerTypeVerify, true, false, false)
	TxTypeGroupPieceNilData = generateGroupTx(1, types.TransactionTypeGroupPiece, uint64(500000), uint64(1000), types.MinerTypeVerify, true, true, false)
	TxTypeGroupPieceWithTarget = generateGroupTx(1, types.TransactionTypeGroupPiece, uint64(500000), uint64(1000), types.MinerTypeVerify, false, false, true)
	TxTypeRewardBadData = generateTX([]byte(string("12345")), 1, 0, "111", types.TransactionTypeReward, uint64(500000), uint64(1000), []byte(string("12345")))
	TxTypeRewardBadExtra = generateTX(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000123"), 1, 0, "111", types.TransactionTypeReward, uint64(500000), uint64(1000), nil)

	// evil type of tx
	TxTypeEvil = generateTX(nil, 1, 0, "111", 99, uint64(500000), uint64(1000), nil)
	// no type tx
	TxNoType = generateNoTypeTx("111", 1, uint64(500000), uint64(1000))

	//about gas limit and gas price
	TxLowGasLimit = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(399), uint64(1000), nil)
	TxHighGasLimit = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500001), uint64(1000), nil)
	TxLowGasPrice = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(399), nil)
	//TxHighGasPrice = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(100000000), nil)
	TxHighGasPriceOverBalance = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000000000000), nil)
	TxNoGasPrice = generateNoGasPriceTx("111", 1, uint64(500000), types.TransactionTypeTransfer)
	TxNoGasLimit = generateNoGasLimitTx("111", 1, uint64(1000), types.TransactionTypeTransfer)

	// bad sign tx
	TxErrSign = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxErrSign.Sign = reverse(TxErrSign.Sign)
	TxNilSign = generateNoSignTx("111", 1, uint64(500000), uint64(1000), types.TransactionTypeTransfer)
	TxLongSign = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxLongSign.Sign = append(TxLongSign.Sign, TxLongSign.Sign...)
	TxShortSign = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	TxShortSign.Sign = TxShortSign.Sign[:len(TxShortSign.Sign)-1]

	// a not exist source addr
	TxSourceNotExist = generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	source1 := common.StringToAddress("zv0000000000000000000000000000000000000000000000000000000000000123")
	TxSourceNotExist.Source = &source1

	TxTargetNotExist = generateTX(nil, 1, 0, "", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	//TxTargetNotExist.Target = nil

}

type NetTest struct {
}

func (s *NetTest) BuildProposerGroupNet(proposers []*network.Proposer) {
	panic("implement me")
}

func (s *NetTest) AddProposers(proposers []*network.Proposer) {
	panic("implement me")
}

func (s *NetTest) Send(id string, msg network.Message) error {
	txbody := msg.Body
	txs, err := types.UnMarshalTransactions(txbody)
	if err != nil {
		return err
	}
	ts := []*types.Transaction{}
	for _, t := range txs {
		tn := &types.Transaction{
			RawTransaction: t,
			Hash:           t.GenHash(),
		}
		ts = append(ts, tn)
	}

	OnReqTxsTest = ts
	return nil
}

func (s *NetTest) SendWithGroupRelay(id string, groupID string, msg network.Message) error { return nil }

func (s *NetTest) SpreadAmongGroup(groupID string, msg network.Message) error { return nil }

func (s *NetTest) SpreadToGroup(groupID string, groupMembers []string, msg network.Message, digest network.MsgDigest) error {
	return nil
}

func (s *NetTest) NotifyTransactions(msg network.Message) error { return nil }

func (s *NetTest) TransmitToNeighbor(msg network.Message) error { return nil }

func (s *NetTest) Broadcast(msg network.Message) error { return nil }

func (s *NetTest) ConnInfo() []network.Conn { return nil }

func (s *NetTest) BuildGroupNet(groupID string, members []string) {}

func (s *NetTest) DissolveGroupNet(groupID string) {}

// BigUint used as big.Int. Inheritance is for the implementation of Marshaler/Unmarshaler interface in msgpack framework
type BigInt struct {
	big.Int
}

func NewBigInt(v uint64) *BigInt {
	return &BigInt{Int: *new(big.Int).SetUint64(v)}
}

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

func AddBalance() {
	blocks := GenBlocks()
	stateDB, _ := account.NewAccountDB(common.Hash{}, BlockChainImpl.stateCache)

	stateDB.AddBalance(common.StringToAddress(adminAddress), new(big.Int).SetUint64(99999999999999999))

	exc := &executePostState{state: stateDB}
	root := stateDB.IntermediateRoot(true)
	blocks[0].Header.StateTree = common.BytesToHash(root.Bytes())
	BlockChainImpl.commitBlock(blocks[0], exc)
	lastBlockHash = blocks[0].Header.Hash
}

func TestMinerAdd(t *testing.T) {
	defer clearSelf(t)
	initTxsAndOthers(t)
	tx := generateStakeAddTx(1, "111", types.TransactionTypeStakeAdd, uint64(500000), uint64(1000), types.MinerTypeProposal)
	marshallTx := marshallTxs([]*types.Transaction{tx}, t)
	msg := notify.NewDefaultMessage(marshallTx, kindAddr, 0, 0)

	err := TxSyncer.onTxResponse(msg)
	if err != nil {
		fmt.Println(err)
	}

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
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	//tx.Source = source[:][:20]
	//copy(tx.Source[:],source...)
	tx.Source = &source
	tx.Hash = tx.GenHash()
	sign, _ := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()
	tx.Sign = append(tx.Sign, 'a')

	return tx
}

func transactionFakeToPb(t *FakeTransaction) *tas_middleware_pb.RawTransaction {
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
	transaction := tas_middleware_pb.RawTransaction{
		Data:      t.Data,
		Value:     t.Value.GetBytesWithSign(),
		Nonce:     &t.Nonce,
		Target:    target,
		GasLimit:  t.GasLimit.GetBytesWithSign(),
		GasPrice:  t.GasPrice.GetBytesWithSign(),
		ExtraData: t.ExtraData,
		Type:      &tp,
		Sign:      t.Sign,
	}
	return &transaction
}

func TransactionsFakeToPb(txs []*FakeTransaction) []*tas_middleware_pb.RawTransaction {
	if txs == nil {
		return nil
	}
	transactions := make([]*tas_middleware_pb.RawTransaction, 0)
	for _, t := range txs {
		transaction := transactionFakeToPb(t)
		transactions = append(transactions, transaction)
	}
	return transactions
}

// MarshalTransactions serialize []*Transaction
func MarshalFakeTransactions(txs []*FakeTransaction) ([]byte, error) {
	transactions := TransactionsFakeToPb(txs)
	transactionSlice := tas_middleware_pb.RawTransactionSlice{Transactions: transactions}
	return proto.Marshal(&transactionSlice)
}

func generateGroupTx(value uint64, txType int, gasLimit uint64, gasprice uint64, mType types.MinerType, isEvil bool, isNilData bool, withTarget bool) *types.Transaction {

	//var data []byte
	data1 := make([]byte, 1000)
	if !isEvil {
		seed := common.HexToHash("ab454fdea57373b25b150497e016fcfdc06b55a66518e3756305e46f3dda7ff4")
		sender := common.StringToAddress(adminAddress).Bytes()
		//groupSender := group.NewPacketSender(BlockChainImpl.(*FullBlockChain))

		//Round 1
		data := &group.EncryptedSharePiecePacketImpl{}
		data.SeedD = seed
		data.SenderD = sender
		data.Pubkey0D = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f75555555555555")
		data.PiecesD = common.Hex2Bytes("65e85ec7613cdb6bc6e40d3b09c1c2efd9556b82a1e4b3db5f00000000000000")
		byteData, err := msgpack.Marshal(data)
		fmt.Println("LEN:", len(byteData))
		if err != nil {
			fmt.Printf("msg pack err: %v", err)
		}
		data1 = byteData
	} else {
		if isNilData {
			data1 = []byte{}
		} else {
			data1 = []byte{'a'}
		}
	}

	tx := &types.RawTransaction{
		Data:     data1,
		Value:    types.NewBigInt(value),
		Nonce:    Nonce,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}

	if withTarget {
		targetbyte := common.StringToAddress("111")
		tx.Target = &targetbyte
	}

	//tx.Hash = tx.GenHash()
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()

	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func genetateMinerOpTx(value uint64, target string, txType int, gasLimit uint64, gasprice uint64, mType types.MinerType, isEvil bool) *types.Transaction {
	targetbyte := common.StringToAddress(target)
	var tx *types.RawTransaction
	if isEvil {
		tx = &types.RawTransaction{
			Data:     []byte{byte('.')},
			Value:    types.NewBigInt(value),
			Nonce:    Nonce,
			Target:   &targetbyte,
			Type:     int8(txType),
			GasLimit: types.NewBigInt(gasLimit),
			GasPrice: types.NewBigInt(gasprice),
		}
	} else {
		tx = &types.RawTransaction{
			Data:     []byte{byte(mType)},
			Value:    types.NewBigInt(value),
			Nonce:    Nonce,
			Target:   &targetbyte,
			Type:     int8(txType),
			GasLimit: types.NewBigInt(gasLimit),
			GasPrice: types.NewBigInt(gasprice),
		}
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()

	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateStakeAddTx(value uint64, target string, txType int, gasLimit uint64, gasprice uint64, mType types.MinerType) *types.Transaction {

	pks := &types.MinerPks{
		MType: types.MinerType(mType),
	}
	var bpk groupsig.Pubkey
	bpk.SetHexString(adminBPK)
	pks.Pk = bpk.Serialize()
	pks.VrfPk = base.Hex2VRFPublicKey(adminVRFPK)
	data, err := types.EncodePayload(pks)
	if err != nil {
		fmt.Println(err)
	}

	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Data:     data,
		Value:    types.NewBigInt(value),
		Nonce:    Nonce,
		Target:   &targetbyte,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()

	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateFakeStakeAddTxs(value uint64, target string, txType int, gasLimit uint64, gasprice uint64, mType types.MinerType) []*types.Transaction {

	pks := &types.MinerPks{
		MType: types.MinerType(mType),
	}
	var bpk groupsig.Pubkey
	bpk.SetHexString(adminBPK)
	pks.Pk = bpk.Serialize()
	pks.VrfPk = base.Hex2VRFPublicKey(adminVRFPK)
	data, err := types.EncodePayload(pks)
	if err != nil {
		fmt.Println(err)
	}

	data1 := make([]byte, len(data))
	data2 := make([]byte, len(data))
	data3 := make([]byte, len(data))
	data4 := make([]byte, len(data))
	data5 := make([]byte, len(data)+1)
	copy(data2, data)
	copy(data3, data)
	copy(data4, data)
	copy(data5, data)

	data1 = data1[:len(data1)/2]
	data2[0] = 2
	data3[1] = 3
	data4[len(data4)-1] = data4[len(data4)-1] + 1
	data5 = append(data5, 0)
	datas := [][]byte{data1, data2, data3, data4, data5}

	var txs []*types.Transaction
	targetbyte := common.StringToAddress(target)
	for i := 0; i < 5; i++ {
		tx := &types.RawTransaction{
			Data:     datas[i],
			Value:    types.NewBigInt(value),
			Nonce:    Nonce,
			Target:   &targetbyte,
			Type:     int8(txType),
			GasLimit: types.NewBigInt(gasLimit),
			GasPrice: types.NewBigInt(gasprice),
		}
		sk1 := common.HexToSecKey(adminPrivateKey)
		source1 := sk1.GetPubKey().GetAddress()
		tx.Source = &source1

		sign, _ := sk1.Sign(tx.GenHash().Bytes())
		tx.Sign = sign.Bytes()

		tn := &types.Transaction{
			RawTransaction: tx,
			Hash:           tx.GenHash(),
		}
		txs = append(txs, tn)
		Nonce++

	}
	return txs
}

func marshallTxs(txs []*types.Transaction, t *testing.T) []byte {
	ts := []*types.RawTransaction{}
	for _, tx := range txs {
		ts = append(ts, tx.RawTransaction)
	}
	body, e := types.MarshalTransactions(ts)
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
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Target:   &targetbyte,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoNonceTx(target string, value uint64, txType int, gasLimit uint64, gasprice uint64) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		//Nonce:     Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		Type:     int8(txType),
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoTypeTx(target string, value uint64, gasLimit uint64, gasprice uint64) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasLimit: types.NewBigInt(gasLimit),
		GasPrice: types.NewBigInt(gasprice),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoGasPriceTx(target string, value uint64, gasLimit uint64, txType int) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasLimit: types.NewBigInt(gasLimit),
		Type:     int8(txType),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoHashTx(target string, value uint64, gasPrice uint64, gasLimit uint64, txType int) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasPrice: types.NewBigInt(gasPrice),
		GasLimit: types.NewBigInt(gasLimit),
		Type:     int8(txType),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoSignTx(target string, value uint64, gasPrice uint64, gasLimit uint64, txType int) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasPrice: types.NewBigInt(gasPrice),
		GasLimit: types.NewBigInt(gasLimit),
		Type:     int8(txType),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateNoGasLimitTx(target string, value uint64, gasPrice uint64, txType int) *types.Transaction {
	//tx := &types.Transaction{}
	targetbyte := common.StringToAddress(target)
	tx := &types.RawTransaction{
		Nonce:    Nonce,
		Value:    types.NewBigInt(value),
		Target:   &targetbyte,
		GasPrice: types.NewBigInt(gasPrice),
		Type:     int8(txType),
	}
	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()
	Nonce++

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateTX(data []byte, value uint64, nonce uint64, target string, txType int, gasLimit uint64, gasprice uint64, extraData []byte) *types.Transaction {
	if Count == 0 {
		Nonce = 1
	}
	tx := &types.RawTransaction{}
	targetbyte := common.StringToAddress(target)
	if nonce != 0 {
		// a mark
		if nonce == 999999999 {
			tx = &types.RawTransaction{
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
			tx = &types.RawTransaction{
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
			tx = &types.RawTransaction{
				Data:      data,
				Value:     types.NewBigInt(value),
				Nonce:     Nonce,
				Type:      int8(txType),
				GasLimit:  types.NewBigInt(gasLimit),
				GasPrice:  types.NewBigInt(gasprice),
				ExtraData: extraData,
			}
		} else {
			tx = &types.RawTransaction{
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

	sk := common.HexToSecKey(adminPrivateKey)
	source := sk.GetPubKey().GetAddress()
	tx.Source = &source
	sign, _ := sk.Sign(tx.GenHash().Bytes())
	tx.Sign = sign.Bytes()

	return &types.Transaction{
		RawTransaction: tx,
		Hash:           tx.GenHash(),
	}
}

func generateTXs(count int, random bool) []*types.Transaction {
	txs := []*types.Transaction{}

	if random {
		for i := 0; i < count; i++ {
			tx := generateTX([]byte(strconv.Itoa(i)), 1, 0, "111", types.TransactionTypeTransfer, uint64(1000+i), uint64(1000+i), nil)
			txs = append(txs, tx)
		}
	} else {
		for i := 0; i < count; i++ {
			tx := generateTX(nil, 1, 0, "111", types.TransactionTypeTransfer, uint64(1000+i), uint64(1000+i), nil)
			txs = append(txs, tx)
		}
	}

	var sign = common.BytesToSign(txs[0].Sign)
	pk, err := sign.RecoverPubkey(txs[0].Hash.Bytes())
	if err != nil {
		fmt.Println("generate tx err")
		return nil
	}
	src := pk.GetAddress()
	fmt.Println("SRC:", src.AddrPrefixString())
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
	txs := BlockChainImpl.GetTransactionPool().GetAllTxs()
	return txs
}

func getNonce(addr string) uint64 {
	realAddr := common.StringToAddress(addr)
	return BlockChainImpl.GetNonce(realAddr)
}

func getBalance(addr string) uint64 {
	realAddr := common.StringToAddress(addr)
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

func TestFakeTxs(t *testing.T) {
	defer clearSelf(t)
	initTxsAndOthers(t)

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
		t.Error("fake txs can't be add in the tx pool")
	}

	txs1 := getTxPoolTx()
	if len(txs1) != 0 {
		t.Error("fake txs can't be add in the tx pool")
	}

}

func TestOnTxResponse(t *testing.T) {
	defer clearSelf(t)

	initTxsAndOthers(t)

	bodyTxNilBody := marshallTxs([]*types.Transaction{}, t)
	body10 := marshallTxs(Txs10, t)
	bodyBigData := marshallTxs([]*types.Transaction{TxBigData}, t)
	bodyBigExtraData := marshallTxs([]*types.Transaction{TxBigExtraData}, t)
	bodyBigExtraDataAndExtraData := marshallTxs([]*types.Transaction{TxBigExtraDataAndExtraData}, t)
	bodyDataAndRxtraDataWithSpaces := marshallTxs([]*types.Transaction{TxDataAndRxtraDataWithSpaces}, t)

	//bodyOverBalanceValue := marshallTxs([]*types.Transaction{TxOverBalanceValue}, t)
	//bodyTxNoValue := marshallTxs([]*types.Transaction{TxNoValue}, t)

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
	bodyTxTypeStakeAddProposal := marshallTxs([]*types.Transaction{TxTypeStakeAddProposal}, t)
	bodyTxTypeStakeAddVerify := marshallTxs([]*types.Transaction{TxTypeStakeAddVerify}, t)
	bodyTxTypeStakeAddFake1 := marshallTxs([]*types.Transaction{TxTypeStakeAddFake1}, t)
	bodyTxTypeStakeAddFake2 := marshallTxs([]*types.Transaction{TxTypeStakeAddFake2}, t)
	bodyTxTypeStakeAddFake3 := marshallTxs([]*types.Transaction{TxTypeStakeAddFake3}, t)
	//bodyTxTypeStakeAddFake4 := marshallTxs([]*types.Transaction{TxTypeStakeAddFake4}, t)
	bodyTxTypeStakeAddFake5 := marshallTxs([]*types.Transaction{TxTypeStakeAddFake5}, t)
	bodyTxTypeStakeReduce := marshallTxs([]*types.Transaction{TxTypeStakeReduce}, t)
	bodyTxTypeStakeReduceFake1 := marshallTxs([]*types.Transaction{TxTypeStakeReduceFake1}, t)
	bodyTxTypeMinerAbort := marshallTxs([]*types.Transaction{TxTypeMinerAbort}, t)
	bodyTxTypeStakeRefund := marshallTxs([]*types.Transaction{TxTypeStakeRefund}, t)
	bodyTxTypeGroupPiece := marshallTxs([]*types.Transaction{TxTypeGroupPiece}, t)
	//bodyTxTypeGroupPieceBadData := marshallTxs([]*types.Transaction{TxTypeGroupPieceBadData}, t)
	bodyTxTypeGroupPieceNilData := marshallTxs([]*types.Transaction{TxTypeGroupPieceNilData}, t)
	bodyTxTypeGroupPieceWithTarget := marshallTxs([]*types.Transaction{TxTypeGroupPieceWithTarget}, t)
	bodyTxTypeRewardBadData := marshallTxs([]*types.Transaction{TxTypeRewardBadData}, t)
	bodyTxTypeRewardBadExtra := marshallTxs([]*types.Transaction{TxTypeRewardBadExtra}, t)

	bodyTxTypeEvil := marshallTxs([]*types.Transaction{TxTypeEvil}, t)
	bodyTxNoType := marshallTxs([]*types.Transaction{TxNoType}, t)

	bodyTxLowGasLimit := marshallTxs([]*types.Transaction{TxLowGasLimit}, t)
	bodyTxHighGasLimit := marshallTxs([]*types.Transaction{TxHighGasLimit}, t)
	bodyTxLowGasPrice := marshallTxs([]*types.Transaction{TxLowGasPrice}, t)
	//bodyTxHighGasPrice := marshallTxs([]*types.Transaction{TxHighGasPrice}, t)
	bodyTxHighGasPriceOverBalance := marshallTxs([]*types.Transaction{TxHighGasPriceOverBalance}, t)
	bodyTxNoGasPrice := marshallTxs([]*types.Transaction{TxNoGasPrice}, t)
	bodyTxNoGasLimit := marshallTxs([]*types.Transaction{TxNoGasLimit}, t)

	bodyTxErrSign := marshallTxs([]*types.Transaction{TxErrSign}, t)
	bodyTxNilSign := marshallTxs([]*types.Transaction{TxNilSign}, t)
	bodyTxLongSign := marshallTxs([]*types.Transaction{TxLongSign}, t)
	bodyTxShortSign := marshallTxs([]*types.Transaction{TxShortSign}, t)

	//bodyTxSourceNotExist := marshallTxs([]*types.Transaction{TxSourceNotExist}, t)

	bodyTxTargetNotExist := marshallTxs([]*types.Transaction{TxTargetNotExist}, t)

	goodMsg := []*notify.DefaultMessage{

		//nil tx
		notify.NewDefaultMessage(bodyTxNilBody, kindAddr, 0, 0),

		// 10 txs and kind addr
		notify.NewDefaultMessage(body10, kindAddr, 0, 0),

		// tx with sepcial chars and spaces in data and extra data
		notify.NewDefaultMessage(bodyDataAndRxtraDataWithSpaces, kindAddr, 0, 0),

		// todo? check when exexute
		// tx with over balance value
		//notify.NewDefaultMessage(bodyOverBalanceValue, kindAddr, 0, 0),

		// transfer tx
		notify.NewDefaultMessage(bodyTxTypeTransfer, kindAddr, 0, 0),

		// contract create tx
		notify.NewDefaultMessage(bodyTxTypeContractCreate, kindAddr, 0, 0),

		// contract call tx
		notify.NewDefaultMessage(bodyTxTypeContractCall, kindAddr, 0, 0),

		notify.NewDefaultMessage(bodyTxNoType, kindAddr, 0, 0),

		//stakeadd
		notify.NewDefaultMessage(bodyTxTypeStakeAddProposal, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeStakeAddVerify, kindAddr, 0, 0),

		// stake reduce
		notify.NewDefaultMessage(bodyTxTypeStakeReduce, kindAddr, 0, 0),

		// miner abort
		notify.NewDefaultMessage(bodyTxTypeMinerAbort, kindAddr, 0, 0),

		// stake refund
		notify.NewDefaultMessage(bodyTxTypeStakeRefund, kindAddr, 0, 0),

		// group piece
		notify.NewDefaultMessage(bodyTxTypeGroupPiece, kindAddr, 0, 0),

		// stake add
		//notify.NewDefaultMessage(bodyTypeypeStakeAdd, kindAddr, 0, 0),

		// high gas pirce
		//notify.NewDefaultMessage(bodyTxHighGasPrice, kindAddr, 0, 0),

		// todo even source is a fake source there's no relations
		//notify.NewDefaultMessage(bodyTxSourceNotExist, kindAddr, 0, 0),
		// make a duplicate transaction, must take it at the end of the list
		//notify.NewDefaultMessage(bodyTxSourceNotExist, kindAddr, 0, 0),
	}

	bodyCountsMap := map[*notify.DefaultMessage]int{
		goodMsg[0]: len([]*types.Transaction{TxNilBody}),
		goodMsg[1]: len(Txs10),
		goodMsg[2]: len([]*types.Transaction{TxDataAndRxtraDataWithSpaces}),
		//goodMsg[3]:  len([]*types.Transaction{TxOverBalanceValue}),
		goodMsg[3]:  len([]*types.Transaction{TxTypeTransfer}),
		goodMsg[4]:  len([]*types.Transaction{TxTypeContractCreate}),
		goodMsg[5]:  len([]*types.Transaction{TxTypeContractCall}),
		goodMsg[6]:  len([]*types.Transaction{TxNoType}),
		goodMsg[7]:  len([]*types.Transaction{TxTypeStakeAddProposal}),
		goodMsg[8]:  len([]*types.Transaction{TxTypeStakeAddVerify}),
		goodMsg[9]:  len([]*types.Transaction{TxTypeStakeReduce}),
		goodMsg[10]: len([]*types.Transaction{TxTypeMinerAbort}),
		goodMsg[11]: len([]*types.Transaction{TxTypeStakeRefund}),
		goodMsg[12]: len([]*types.Transaction{TxTypeGroupPiece}),

		//goodMsg[13]: len([]*types.Transaction{TxHighGasPrice}),
		//goodMsg[15]: len([]*types.Transaction{TxSourceNotExist}),
		// must take it at the end of the list
		//goodMsg[16]: len([]*types.Transaction{TxSourceNotExist}),
	}

	// good txs
	t.Run("good txs", func(t *testing.T) {
		var poolLenBefore int
		var poolLenAfter int
		for k, msg := range goodMsg {

			poolLenBefore = len(getTxPoolTx())
			err := TxSyncer.onTxResponse(msg)
			poolLenAfter = len(getTxPoolTx())
			if k == 0 {
				continue
				//} else if k == len(goodMsg)-1 {
				//	fmt.Printf("%d:", k)
				//	fmt.Println("BeFORE:", poolLenBefore, "bodyCountsMap:", bodyCountsMap[msg], "poolLenAfter:", poolLenAfter)
				//	if poolLenBefore != poolLenAfter {
				//		t.Errorf("No.%d good exexute result is not what is expected，please check manually!", k)
				//	}
				//	if err != nil {
				//		fmt.Println(">err:", err)
				//		t.Errorf("No.%d good exexute result should success and retrun a nil error，please check manually!", k)
				//	}
			} else {
				fmt.Printf("%d:", k)
				fmt.Println("BeFORE:", poolLenBefore, "bodyCountsMap:", bodyCountsMap[msg], "poolLenAfter:", poolLenAfter)
				if poolLenBefore+bodyCountsMap[msg] != poolLenAfter {
					t.Errorf("No.%d good exexute result is not what is expected，please check manually! before pool length:%d,after pool length:%d,ideal length is %d\n", k, poolLenBefore, poolLenAfter, poolLenBefore+bodyCountsMap[msg])
				}
				if err != nil {
					fmt.Println(">err:", err)
					t.Errorf("No.%d good exexute result should success and retrun a nil error，please check manually!", k)
				}
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
		//notify.NewDefaultMessage(bodyTxNoValue, kindAddr, 0, 0),

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
		notify.NewDefaultMessage(bodyTxNoGasPrice, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxNoGasLimit, kindAddr, 0, 0),

		// err sign
		notify.NewDefaultMessage(bodyTxErrSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxNilSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxLongSign, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxShortSign, kindAddr, 0, 0),

		// todo
		notify.NewDefaultMessage(bodyTxTypeEvil, kindAddr, 0, 0),

		notify.NewDefaultMessage(bodyTxTargetNotExist, kindAddr, 0, 0),

		// fake stake adds
		notify.NewDefaultMessage(bodyTxTypeStakeAddFake1, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeStakeAddFake2, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeStakeAddFake3, kindAddr, 0, 0),
		// todo 修改data区域最后的字符不会报错
		//notify.NewDefaultMessage(bodyTxTypeStakeAddFake4, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeStakeAddFake5, kindAddr, 0, 0),

		// fake stake reduce
		notify.NewDefaultMessage(bodyTxTypeStakeReduceFake1, kindAddr, 0, 0),

		// todo group bad data 是否严格判断
		//notify.NewDefaultMessage(bodyTxTypeGroupPieceBadData, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeGroupPieceNilData, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeGroupPieceWithTarget, kindAddr, 0, 0),

		// reward tx with bad data and extradata
		notify.NewDefaultMessage(bodyTxTypeRewardBadData, kindAddr, 0, 0),
		notify.NewDefaultMessage(bodyTxTypeRewardBadExtra, kindAddr, 0, 0),
	}

	//不返回错误，不加入交易池
	t.Run("make bad msg bodies-badMsg02", func(t *testing.T) {
		poolLenBefore := len(getTxPoolTx())
		fmt.Println("poolLenBefore:", poolLenBefore)
		for k, msg := range badMsg02 {
			poolLenBefore := len(getTxPoolTx())
			err := TxSyncer.onTxResponse(msg)
			fmt.Printf("%d:make bad msg bodies err:%v\n", k, err)
			poolLenAfter := len(getTxPoolTx())
			if poolLenBefore != poolLenAfter {
				fmt.Println("K:", k)
				poolLenBefore = poolLenAfter
			}
		}
		poolLenAfter := len(getTxPoolTx())
		fmt.Println("poolLenAfter:", poolLenAfter)
		if poolLenBefore != poolLenAfter {
			t.Error("bad tx shouldn't add into txpool")
		}
	})

}

func TestOnTxNotify(t *testing.T) {
	defer clearSelf(t)

	initTxsAndOthers(t)

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

	// test if txpool has this tx or not
	NewTx01 := generateTX([]byte{'1'}, 1, 0, "12", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	message01 := prepareMsgs([]*types.Transaction{NewTx01})
	notifyMsg01 := notify.NewDefaultMessage(message01.Body, kindAddr, 0, 0)

	// put NewTx02 in to txpool
	NewTx02 := generateTX([]byte{'2'}, 1, 0, "121", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	message02 := prepareMsgs([]*types.Transaction{NewTx02})
	// notify msg
	notifyMsg02 := notify.NewDefaultMessage(message02.Body, kindAddr, 0, 0)
	// resp msg, put tx02 into txpool
	bodyNewTx02 := marshallTxs([]*types.Transaction{NewTx02}, t)
	respMsg02 := notify.NewDefaultMessage(bodyNewTx02, kindAddr, 0, 0)
	TxSyncer.onTxResponse(respMsg02)

	countBefore := 0
	countAfter1 := 0
	countAfter2 := 0
	TxSyncer.getOrAddCandidateKeys(notifyMsg01.Source()).forEach(func(k common.Hash) bool {
		countBefore++
		return true
	})
	fmt.Println("countBefore:", countBefore)
	TxSyncer.onTxNotify(notifyMsg01)
	TxSyncer.getOrAddCandidateKeys(notifyMsg01.Source()).forEach(func(k common.Hash) bool {
		countAfter1++
		return true
	})
	fmt.Println("countAfter1:", countAfter1)
	if countBefore != countAfter1-1 {
		t.Error("tx pool have no this tx, should take it")
	}
	TxSyncer.onTxNotify(notifyMsg02)
	TxSyncer.getOrAddCandidateKeys(notifyMsg01.Source()).forEach(func(k common.Hash) bool {
		countAfter2++
		return true
	})
	fmt.Println("countAfter2:", countAfter2)

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

func TestOnTxResquest(t *testing.T) {
	defer clearSelf(t)

	initTxsAndOthers(t)

	// put NewTx02 in to txpool
	NewTx := generateTX([]byte(strconv.Itoa(int(time.Now().UnixNano()))), 1, 0, "121", types.TransactionTypeTransfer, uint64(500000), uint64(1000), nil)
	// resp msg, put tx02 into txpool
	bodyNewTx := marshallTxs([]*types.Transaction{NewTx}, t)
	respMsg := notify.NewDefaultMessage(bodyNewTx, kindAddr, 0, 0)
	TxSyncer.onTxResponse(respMsg)

	// generate 3000+ txs(over limit)
	Txs3000 = generateTXs(3002, true)
	var TxHashesOver3000 []common.Hash
	for _, tx := range Txs3000 {
		TxHashesOver3000 = append(TxHashesOver3000, tx.Hash)
	}

	// generate 2000+ txs
	Txs2000 = generateTXs(2000, true)
	var TxHashes2000 []common.Hash
	for _, tx := range Txs2000 {
		TxHashes2000 = append(TxHashes2000, tx.Hash)
	}

	msg3000 := prepareMsgs(Txs3000)
	msg2000 := prepareMsgs(Txs2000)
	msgHaveInPool := prepareMsgs([]*types.Transaction{NewTx})
	msgNilBody := network.Message{0, 0, 0, nil}

	badMsg := []*notify.DefaultMessage{
		notify.NewDefaultMessage(msg3000.Body, kindAddr, 0, 0),
	}

	for k, v := range badMsg {
		err := TxSyncer.onTxReq(v)
		if err == nil {
			t.Errorf("No.%d badMsg exexute result is not what is expected，please check manually!", k)
		}

	}
	goodMsg := []*notify.DefaultMessage{
		// txs doesn't in txpool
		notify.NewDefaultMessage(msg2000.Body, kindAddr, 0, 0),
		// this tx have in txpool
		notify.NewDefaultMessage(msgHaveInPool.Body, kindAddr, 0, 0),
		// nil body
		notify.NewDefaultMessage(msgNilBody.Body, kindAddr, 0, 0),
	}
	for k, v := range goodMsg {
		err := TxSyncer.onTxReq(v)
		if err != nil {
			t.Errorf("No.%d badMsg exexute result is not what is expected，please check manually!", k)
		}
		if k == 1 {
			if len(OnReqTxsTest) != 1 {
				fmt.Println("len(OnReqTxsTest):", len(OnReqTxsTest))
				t.Errorf("%s tx have added in txpool, should find it in pool", "NewTx")
			}
		}
	}
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
