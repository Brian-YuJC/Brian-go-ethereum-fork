package parallel

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Debug print
func print(item ...interface{}) { //利用 interface{} 来传递任意参数, 用...表示不限参数的个数
	//fmt.Print("[Debug]")
	fmt.Printf("%c[31;40;5m%s%c[0m", 0x1B, "[Debug Print]", 0x1B) //打印高亮文本
	for i := range item {
		fmt.Print(" ", item[i])
	}
	fmt.Print("\n")
}

// Transaction information 结构体里面记录了我们想要记录的数据
type TransactionInfo struct {
	TxHash    common.Hash    //Transaction 的 Hash
	From      common.Address // Transaction sender 的 Address
	To        common.Address //Transaction 发往的Address
	Value     *big.Int       //Transaction 中交易金额
	Fee       *big.Int
	GasPrice  *big.Int
	Data      []byte        //Transaction 携带的 Data
	CallStack []common.Hash //该 Transaction 调用smart contract的调用栈
}

type BlockInfo struct {
	BlockHash common.Hash //block 的 Hash
	GasLimit  uint64
	Tx        []TransactionInfo //该区块中的 Transaction information 列表
}

// 实例化一个全局 Block Information 对象
var blockInfo BlockInfo = BlockInfo{}

// blockInfo getter
func GetBlockInfo() *BlockInfo {
	return &blockInfo
}

// 实例化一个全局 Transaction Information 对象
var txInfo TransactionInfo = TransactionInfo{}

// Block information 的钩子函数，用于在当前Block执行过程中抛出信息,并修改全局变量 blockInfo
// name 抛出参数的名称
// data 抛出参数的内容
func BlockInfoHook(name string, data interface{}) {
	//var err error = nil
	switch name {
	case "BlockHash":
		if data, ok := data.(common.Hash); ok {
			blockInfo.BlockHash = data
		} else {
			//err = errors.New("[BlockInfoHook Error] BlockHash not match common.Hash type")
			print("[BlockInfoHook Error] BlockHash not match common.Hash type")
		}
	case "GasLimit":
		if data, ok := data.(uint64); ok {
			blockInfo.GasLimit = data
		} else {
			//err = errors.New("[BlockInfoHook Error] GasLimit not match uint64 type")
			print("[BlockInfoHook Error] GasLimit not match uint64 type")
		}
	default:
		//err = TransactionInfoHook(name, data)
		TransactionInfoHook(name, data)
	}
	//return err
}

// Transaction information 的钩子函数。用于在当前Transaction执行过程中抛出信息,并修改全局变量 txInfo
func TransactionInfoHook(name string, data interface{}) {
	//var err error = nil
	switch name {
	case "TxHash":
		if data, ok := data.(common.Hash); ok {
			txInfo.TxHash = data
		} else {
			//err = errors.New("[TransactionInfoHook] TxHash not match common.Hash type")
			print("[TransactionInfoHook] TxHash not match common.Hash type")
		}
	case "From":
		if data, ok := data.(common.Address); ok {
			txInfo.From = data
		} else {
			//err = errors.New("[TransactionInfoHook] From not match common.Address type")
			print("[TransactionInfoHook] From not match common.Address type")
		}
	case "To":
		if data, ok := data.(common.Address); ok {
			txInfo.To = data
		} else {
			//err = errors.New("[TransactionInfoHook] To not match common.Address type")
			print("[TransactionInfoHook] To not match common.Address type")
		}
	case "Value":
		if data, ok := data.(*big.Int); ok {
			txInfo.Value = data
		} else {
			//err = errors.New("[TransactionInfoHook] Value not match big.Int type")
			print("[TransactionInfoHook] Value not match big.Int type")
		}
	case "Data":
		if data, ok := data.([]byte); ok {
			txInfo.Data = data
		} else {
			//err = errors.New("[TransactionInfoHook] Data not match []byte type")
			print("[TransactionInfoHook] Data not match []byte type")
		}
	case "GasPrice":
		if data, ok := data.(*big.Int); ok {
			txInfo.GasPrice = data
		} else {
			//err = errors.New("[TransactionInfoHook] GasPrice not match big.Int type")
			print("[TransactionInfoHook] GasPrice not match big.Int type")
		}
	default:
		//err = errors.New("[TransactionInfoHook] Name invalid error")
		print("[TransactionInfoHook] Name invalid error")
	}
	//return err
}

// 当前 Transaction执行完毕，需要重置 TxInfo 实例
// txIndex当前 Transaction 的索引值
func ResetTxInfo(txIndex int) {
	blockInfo.Tx = append(blockInfo.Tx, txInfo)
	txInfo = TransactionInfo{}
}
