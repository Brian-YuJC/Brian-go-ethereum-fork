package parallel

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
)

//---------------------------------------------------------
// 本文件用于从 Hook 的信息获取Account 与 Transaction 之间的关系
// 维护一个点集与一个边集
//---------------------------------------------------------

// 图的节点
type AccountNode struct {
	Address string //账号的地址
	// WhoWrite  string          //被谁进行了读写操作
	// WhoRead   *map[string]int //读该账号的账号的地址以及访问时序
	// WhoCreate *map[string]int
}

// 交易节点
type TxNode struct {
	ID    int      //Transaction 编号
	From  string   //From 地址
	To    string   //To 地址
	Value *big.Int //转账金额
}

// 图的边
type Edge struct {
	From string //图的起点(默认为 Transaction 的 ID)
	To   string //图的终点
	Op   string //进行的操作（读/写/创建/转账/自毁）
	// Opcode string //发起的操作码
}

// 图的数据结构
type Graph struct {
	TxNodeList      []TxNode      //交易点集
	AccountNodeList []AccountNode //账号点集
	EdgeList        []Edge        //边集
}

// 往图里添加Tx 节点的方法
func (g *Graph) AddTxNode(txNode TxNode) {
	g.TxNodeList = append(g.TxNodeList, txNode)
}

// 维护一张 Account 节点的注册表
var AccountNodeMap map[string]bool = make(map[string]bool)

// 往图里添加 Account 节点的方法（判断账号有无被添加过）
func (g *Graph) AddAccountNode(address string) {
	if _, ok := AccountNodeMap[address]; !ok { //该账号节点没有被注册过
		AccountNodeMap[address] = true                                               //注册Account节点
		g.AccountNodeList = append(g.AccountNodeList, AccountNode{Address: address}) //写入
	}
}

// 维护一张 Edge 注册表
var EdgeMap map[string]bool = make(map[string]bool)

// 往图里添加 边的方法（判断相同属性的边有无被添加过）
func (g *Graph) AddEdge(from int, to string, op string) {
	key := strconv.Itoa(from) + to + op
	if _, ok := EdgeMap[key]; !ok { //该边没有被注册过
		EdgeMap[key] = true
		g.EdgeList = append(g.EdgeList, Edge{From: strconv.Itoa(from), To: to, Op: op})
	}
}

// 根据 Hook 的信息建图
func BuildGraph() *Graph {

	//创建图数据结构
	graph := Graph{}

	for i, tx := range GetBlockInfo().Tx {

		//Transaction 的信息
		from := tx.From.Hex()
		to := tx.To
		value := tx.Value

		//建立交易节点
		txNode := TxNode{ID: i, From: from, To: to, Value: value}
		graph.AddTxNode(txNode)

		//Transaction调用的第一个地址From，肯定会读取和改变其 Nonce 所以有依赖关系
		graph.AddAccountNode(from)
		graph.AddEdge(i, from, "Read & Write")

		//处理创建合约的特殊情况
		if tx.To == "nil" {
			graph.AddAccountNode(tx.NewContractAddr.Hex())
			graph.AddEdge(i, tx.NewContractAddr.Hex(), "Create")
		}

		//不管有无调用其他合约，只要 Transaction 的 value 不为空就会进行转账操作(创建合约的情况前面处理过了)
		if value.Sign() > 0 && tx.To != "nil" { //	value > 0 需要转账
			graph.AddAccountNode(to)
			graph.AddEdge(i, to, "Transfer")
		}

		if len(tx.CallQueue) > 0 { //有合约调用情况
			for _, contractInfo := range tx.CallQueue { //进入每个调用过程的循环，如果调用的合约有对自身的读写操作，则 tx 节点直接指向它
				for _, keyOpcode := range contractInfo.KeyOpcode {
					split := strings.Split(keyOpcode, " ")
					opcode := split[1]
					if opcode == "BALANCE" || opcode == "SELFBALANCE" { //opcode 为BALANCE需要记录BALANCE访问的地址，因为涉及读操作
						//对本调用合约之外的合约进行的操作
						accountAddress := split[2]
						graph.AddAccountNode(accountAddress)
						graph.AddEdge(i, accountAddress, "Read")
					}
					if opcode == "SLOAD" {
						//对本调用合约进行的操作
						graph.AddAccountNode(contractInfo.ContractAddr.Hex())
						graph.AddEdge(i, contractInfo.ContractAddr.Hex(), "Read")
					}
					if opcode == "SSTORE" {
						//对本调用合约进行的操作
						graph.AddAccountNode(contractInfo.ContractAddr.Hex())
						graph.AddEdge(i, contractInfo.ContractAddr.Hex(), "Write")
					}
					if opcode == "CREATE" || opcode == "CREATE2" {
						//对本调用合约之外的合约进行的操作
						accountAddress := split[2]
						graph.AddAccountNode(accountAddress)
						graph.AddEdge(i, accountAddress, "Create")
						if split[3] == "doTransfer_true" { //create 有转账发生
							graph.AddEdge(i, accountAddress, "Transfer")
							//转账操作对本调用合约也会进行读和写操作
							graph.AddAccountNode(contractInfo.ContractAddr.Hex())
							graph.AddEdge(i, contractInfo.ContractAddr.Hex(), "Read & Write")
						}
					}
					if opcode == "CALL" {
						if split[3] == "doTransfer_true" { //需要转账
							//对本调用合约之外的合约进行的操作
							accountAddress := split[2]
							graph.AddAccountNode(accountAddress)
							graph.AddEdge(i, accountAddress, "Transfer")
							if accountAddress != contractInfo.ContractAddr.Hex() { //如果不是自己给自己转账（其实 Transfer == Read & Write）
								//转账操作对本调用合约也会进行读和写操作
								graph.AddAccountNode(contractInfo.ContractAddr.Hex())
								graph.AddEdge(i, contractInfo.ContractAddr.Hex(), "Read & Write")
							}
						}
					}
					if opcode == "SELFDESTRUCT" {
						//对本调用合约之外的合约进行的操作(将本调用合约的钱全部转移至目的地址)
						accountAddress := split[2]
						graph.AddAccountNode(accountAddress)
						graph.AddEdge(i, accountAddress, "Transfer")
						//给本调用合约一个自毁标签
						graph.AddAccountNode(contractInfo.ContractAddr.Hex())
						graph.AddEdge(i, contractInfo.ContractAddr.Hex(), "SelfDestruct")
					}
				}
			}

		}
	}

	return &graph
}

// 将 Graph 导出为 Json 格式
func OutputGraph() {
	graph := BuildGraph()

	jsonData, err := json.Marshal(graph)
	if err != nil {
		fmt.Println(err)
	}

	file, err := os.Create("./output/relationship_graph.json")
	if err != nil {
		fmt.Println(err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		fmt.Println(err)
	}

}
