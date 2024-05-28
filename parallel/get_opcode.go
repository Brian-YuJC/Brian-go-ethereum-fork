package parallel

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// opcode 和读写关系的 Map
var OpcodeMap map[string]string = map[string]string{
	"BALANCE": "[Read] ",
	"SLOAD":   "[Read] ",
	"SSTORE":  "[Write] ",
	"CREATE":  "[Read&Write] ",
	"CALL":    "[Read&Write] ",
	//"CALLCODE":     "[Read] ", //可能只会读写合约中不可改变的 code 部分
	//"DELEGATECALL": "[Read] ", //可能只会读写合约中不可改变的 code 部分
	"CREATE2": "[Read&Write] ",
	//"STATICCALL":   "[Read] ", //可能只会读写合约中不可改变的 code 部分
	"SELFDESTRUCT": "[Read&Write] ",
	"SELFBALANCE":  "[Read] ",
}

// Debug print
func print(item ...interface{}) { //利用 interface{} 来传递任意参数, 用...表示不限参数的个数
	//fmt.Print("[Debug]")
	fmt.Printf("%c[31;40;5m%s%c[0m", 0x1B, "[Debug Print]", 0x1B) //打印高亮文本
	for i := range item {
		fmt.Print(" ", item[i])
	}
	fmt.Print("\n")
}

// ContractInfo结构体记录了调用的合约和调用到的opcode
type ContractInfo struct {
	isNewTx      bool           //用于标记当前是否进入新的Transaction，因为要做空值判断，而判断结构体是否为空有点麻烦（应该）
	Layer        int            //该合约所在的调用的层数
	ContractAddr common.Address // Contract 的 Hash
	OpcodeList   []string       `json:"-"` // 调用的 opcode 列表 （导出 Json 时忽略这个属性）
	KeyOpcode    []string       //重点关注的 opcode（涉及读写操作）
}

// Transaction information 结构体里面记录了我们想要记录的数据
type TransactionInfo struct {
	TxHash    common.Hash    //Transaction 的 Hash
	From      common.Address // Transaction sender 的 Address
	To        common.Address //Transaction 发往的Address
	Value     *big.Int       //Transaction 中交易金额
	Fee       *big.Int
	GasPrice  *big.Int
	Data      []byte         //Transaction 携带的 Data
	CallQueue []ContractInfo //该 Transaction 调用smart contract的调用栈
}

type BlockInfo struct {
	BlockHash common.Hash //block 的 Hash
	GasLimit  uint64
	Tx        []TransactionInfo //该区块中的 Transaction information 列表
}

// 堆栈结构体
type Stack struct {
	//ptr  uint64        //堆栈指针，指向栈顶
	item []interface{} //堆栈
}

// push方法
func (s *Stack) push(item interface{}) {
	s.item = append(s.item, item)
}

// pop方法
func (s *Stack) pop() interface{} {
	r := s.item[len(s.item)-1]
	s.item = s.item[:len(s.item)-1]
	return r
}

// Top 方法
func (s *Stack) top() interface{} {
	return s.item[len(s.item)-1]
}

// clear 方法
func (s *Stack) clear() {
	s = &Stack{}
}

// isEmpty 方法
func (s *Stack) isEmpty() bool {
	return len(s.item) == 0
}

// size 方法
func (s *Stack) size() int {
	return len(s.item)
}

// 实例化一个全局 Block Information 对象
var blockInfo BlockInfo = BlockInfo{}

// blockInfo getter
func GetBlockInfo() *BlockInfo {
	return &blockInfo
}

// 实例化一个全局 Transaction Information 对象
var txInfo TransactionInfo = TransactionInfo{}

// 实例化一个全局 Contract Information 对象
var contractInfo = ContractInfo{}

// 实例化一个Addr栈该栈指示了当前的 Run 的 contract 的地址
var AddrStack = Stack{}

// Block information 的钩子函数，用于在当前Block执行过程中抛出信息,并修改全局变量 blockInfo
// name 抛出参数的名称
// data 抛出参数的内容
func BlockInfoHook(name string, data interface{}) {
	switch name {
	case "BlockHash":
		if data, ok := data.(common.Hash); ok {
			blockInfo.BlockHash = data
		} else {
			print("👎[BlockInfoHook Error] BlockHash not match common.Hash type")
		}
	case "GasLimit":
		if data, ok := data.(uint64); ok {
			blockInfo.GasLimit = data
		} else {
			print("👎[BlockInfoHook Error] GasLimit not match uint64 type")
		}
	default:
		TransactionInfoHook(name, data)
	}
}

// Transaction information 的钩子函数。用于在当前Transaction执行过程中抛出信息,并修改全局变量 txInfo
func TransactionInfoHook(name string, data interface{}) {
	switch name {
	case "TxHash":
		if data, ok := data.(common.Hash); ok {
			resetTxInfo()                              //接受到新的 Transaction Address 自动回调
			AddrStack.clear()                          //初始化调用栈
			contractInfo = ContractInfo{isNewTx: true} //初始化 ContractInfos
			txInfo.TxHash = data
		} else {
			print("👎[TransactionInfoHook] TxHash not match common.Hash type")
		}
	case "From":
		if data, ok := data.(common.Address); ok {
			txInfo.From = data
		} else {
			print("👎[TransactionInfoHook] From not match common.Address type")
		}
	case "To":
		if data, ok := data.(common.Address); ok {
			txInfo.To = data
		} else {
			print("👎[TransactionInfoHook] To not match common.Address type")
		}
	case "Value":
		if data, ok := data.(*big.Int); ok {
			txInfo.Value = data
		} else {
			print("👎[TransactionInfoHook] Value not match big.Int type")
		}
	case "Data":
		if data, ok := data.([]byte); ok {
			txInfo.Data = data
		} else {
			print("👎[TransactionInfoHook] Data not match []byte type")
		}
	case "GasPrice":
		if data, ok := data.(*big.Int); ok {
			txInfo.GasPrice = data
		} else {
			print("👎[TransactionInfoHook] GasPrice not match big.Int type")
		}
	default:
		ContractInfoHook(name, data)
	}
}

// Contract information 的钩子函数。用于在当前contract执行过程中抛出信息,并修改全局变量 contractInfo
func ContractInfoHook(name string, data interface{}) {
	switch name {
	case "ContractAddr":
		if data, ok := data.(common.Address); ok {

			//Save current contract information

			// 旧空值判断
			// if contractInfo.ContractAddr.String() != "0x0000000000000000000000000000000000000000" {
			// 	txInfo.CallQueue = append(txInfo.CallQueue, contractInfo)
			// }
			// 新空值判断
			if !contractInfo.isNewTx {
				txInfo.CallQueue = append(txInfo.CallQueue, contractInfo)
			}

			//build new contract information
			AddrStack.push(data) //当前的调用环境
			contractInfo = ContractInfo{ContractAddr: data, isNewTx: false}
			contractInfo.Layer = AddrStack.size()

			//print("Current Contract: ", data)
			//AddrStack.printStack()

		} else {
			print("👎[ContractInfoHook] ContractHash not match common.Hash type")
		}
	case "Opcode":
		if data, ok := data.(string); ok {
			contractInfo.OpcodeList = append(contractInfo.OpcodeList, data)
		} else {
			print("👎[ContractInfoHook] Opcode not match byte type")
		}
	case "KeyOpcode":
		if data, ok := data.(string); ok {
			contractInfo.KeyOpcode = append(contractInfo.KeyOpcode, data)
		} else {
			print("👎[ContractInfoHook] Opcode not match byte type")
		}
	default:
		print("👎[ContractInfoHook] Name Invalid error")
	}
}

// TransactionHook 回调方法：每当获取到一个新 Transaction Hash，需要重置 TxInfo 实例
// txIndex当前 Transaction 的索引值
func resetTxInfo() {
	//异常值判断
	if txInfo.TxHash.String() == "0x0000000000000000000000000000000000000000000000000000000000000000" {
		return
	}
	blockInfo.Tx = append(blockInfo.Tx, txInfo)
	txInfo = TransactionInfo{}
}

// Run 回调方法：当前 Run执行完毕，需要进行一些操作
func AfterRun() {

	// Save current contract information
	//结束当前代码段执行后将合约信息写入Transaction调用队列
	//注意一个地址的合约可能由于调用其他合约被划分为好几个contractInfo 实例，未来可以用 layer 调用的层数将他们拼起来
	txInfo.CallQueue = append(txInfo.CallQueue, contractInfo)
	AddrStack.pop() //退出调用环境

	//rebuild contract information
	if !AddrStack.isEmpty() {
		contractInfo = ContractInfo{ContractAddr: AddrStack.top().(common.Address), Layer: AddrStack.size(), isNewTx: false} //新建一个contract Info实例
	}
}
