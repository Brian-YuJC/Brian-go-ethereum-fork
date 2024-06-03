package parallel

import (
	"fmt"
	"sort"
	"strconv"
)

// 判断一前一后两个交易是否有依赖
func HaveRelation(tx string, txBefore string, tx_account_map map[string]map[string]map[string]bool) bool {
	for k1, _ := range tx_account_map[tx] {
		for k2, v2 := range tx_account_map[txBefore] {
			if k1 == k2 && v2["Write"] { //如果两个交易同时访问了相同账户，而且之前的交易对账户进行了读写，则这两个交易有依赖
				return true
			}
		}
	}
	return false
}

func BuildTxRelationGraph() {
	//先建立一张关系图，从关系图中再建立 Transaction 之间的关系
	g := BuildGraph()

	var txList []TxNode = g.TxNodeList
	tx_account_map := make(map[string]map[string]map[string]bool) // tx->account->R/W
	tx_tx_map := make(map[string]map[string]bool)                 // 说明Transaction 依赖哪个 Transaction的图结构

	//初始化两个 map
	for _, t := range txList {
		tx_account_map[strconv.Itoa(t.ID)] = make(map[string]map[string]bool)
		tx_tx_map[strconv.Itoa(t.ID)] = make(map[string]bool)
	}

	//维护tx_account_map
	for _, edge := range g.EdgeList {
		tx := edge.From
		account := edge.To
		op := edge.Op
		if _, ok := tx_account_map[tx][account]; !ok { //如果 account 第一次被访问
			tx_account_map[tx][account] = make(map[string]bool)
		}
		if op == "Transfer" || op == "Create" || op == "SelfDestruct" || op == "Read & Write" {
			tx_account_map[tx][account]["Read"] = true
			tx_account_map[tx][account]["Write"] = true
		} else if op == "Read" {
			tx_account_map[tx][account]["Read"] = true
		} else if op == "Write" {
			tx_account_map[tx][account]["Write"] = true
		} else {
			fmt.Println("[BuildTxRelationGraph()] Op invalid")
		}
	}

	//判断后一个 Transaction与前面所有的 Transaction 有无关连
	for i := 1; i < len(txList); i++ {
		tx := txList[i]
		for j := 0; j < i; j++ {
			txBefore := txList[j]
			if HaveRelation(strconv.Itoa(tx.ID), strconv.Itoa(txBefore.ID), tx_account_map) {
				tx_tx_map[strconv.Itoa(tx.ID)][strconv.Itoa(txBefore.ID)] = true //当前 Transaction 与在他之前的 Transaction 有依赖
			}
		}
	}

	//Print Tx Tx relationship
	fmt.Println("-------------------------------Print Dependency-------------------------------")
	num := len(txList)
	var list [][]string = make([][]string, num)
	for k1, v1 := range tx_tx_map {
		var t []string
		for k2, _ := range v1 {
			t = append(t, k2)
		}
		id, _ := strconv.Atoi(k1)
		list[id] = t
	}
	for i := 0; i < len(list); i++ {
		sort.Slice(list[i], func(j, k int) bool { //按 Tx 先后排个序（为了展示目的）
			J, _ := strconv.Atoi(list[i][j])
			K, _ := strconv.Atoi(list[i][k])
			return J < K
		})
		fmt.Printf("Tx: %d\nDependent Tx: ", i)
		for j := 0; j < len(list[i]); j++ {
			fmt.Print(list[i][j], " ")
		}
		fmt.Println("\n")
	}

	//打印并行顺序
	fmt.Println("-------------------------------Print Parallel Order-------------------------------")
	batch := 0
	frontTx := make(map[string]string) //存放合约执行的前置合约（就是必须先执行完前置合约才能执行当前合约，不然有冲突）
	for true {
		if len(tx_tx_map) == 0 { //如果未执行的 Transaction空了则跳出循环
			break
		}

		batch++
		var parallelSet []string //本批执行的列表
		for tx, dependency := range tx_tx_map {
			//fmt.Print(tx, "->", len(dependency), " ")
			if len(dependency) == 0 { //出度为 0
				parallelSet = append(parallelSet, tx) //加入并行
				delete(tx_tx_map, tx)                 //从 map 中移除
			}
		}

		//更新维护剩余 Transaction 与 Transaction 之间的关连(删除与已经加入并行的 Transaction 的关连)
		for _, deletedTx := range parallelSet {
			for tx, dependency := range tx_tx_map {
				if _, ok := dependency[deletedTx]; ok && len(dependency) == 1 { //如果只剩一个依赖，而且该依赖以及执行则该依赖为前置合约
					frontTx[tx] = deletedTx
				}
				delete(dependency, deletedTx)
			}
		}

		//Print Result
		sort.Slice(parallelSet, func(i, j int) bool { //按 Tx 先后排个序（为了展示目的）
			I, _ := strconv.Atoi(parallelSet[i])
			J, _ := strconv.Atoi(parallelSet[j])
			return I < J
		})
		fmt.Println("Parallel Execution Round: ", batch)
		fmt.Printf("The number of Transaction can be parallelly executed: %d\n", len(parallelSet))
		for _, tx := range parallelSet {
			if len(frontTx[tx]) > 0 {
				fmt.Print(tx, "\t(After Tx ", frontTx[tx], " is executed)\n")
			} else {
				fmt.Println(tx)
			}
		}
		fmt.Println("\n")
	}

}

// 只保留会导致Transaction并行冲突的 Account（如果一个 Account 与两个 Transaction 关连则需保留这个节点）
// 用于画图
func BuildDependencyGraph() *Graph {

	//先建立一张关系图，从关系图中再找Dependency
	g := BuildGraph()

	// 获取图的节点和边的信息
	txList := g.TxNodeList
	accountNodeList := g.AccountNodeList
	edgeList := g.EdgeList

	// 维护一个Account 和 Transaction 关连的 Map
	accountTxMap := make(map[string]map[string]bool)

	// 初始化 Map
	for _, account := range accountNodeList {
		accountTxMap[account.Address] = make(map[string]bool)
	}

	// 如果一个 Transaction 指向一个 Account，那么这个 Transaction 与这个 Account 有关连，加入 Map
	for _, edge := range edgeList {
		from := edge.From
		to := edge.To
		accountTxMap[to][from] = true
	}

	// 根据 Transaction 和 Account 的关连 Map，只保留与多个 Transaction 相连的 Account
	newAccountNodeList := []AccountNode{}
	for _, account := range accountNodeList {
		if len(accountTxMap[account.Address]) > 1 { //说明 Account 与多个 Transaction 有关连
			newAccountNodeList = append(newAccountNodeList, account)
		}
	}

	// 根据 Transaction 和 Account 的关连 Map，只保留与多个 Transaction 相连的 Account 有关的边
	newEdgeList := []Edge{}
	for _, edge := range edgeList {
		if len(accountTxMap[edge.To]) > 1 { //说明Edge To的 Account 与多个 Transaction 有关连，影响并行，需要保留这条边
			newEdgeList = append(newEdgeList, edge)
		}
	}

	//构成新的关系图
	newGraph := Graph{}
	newGraph.TxNodeList = txList
	newGraph.AccountNodeList = newAccountNodeList
	newGraph.EdgeList = newEdgeList

	return &newGraph
}

// func Parallel() {
// 	//先建立一张关系图，从关系图中再找Dependency
// 	g := BuildGraph()

// 	// 获取图的节点和边的信息
// 	txList := g.TxNodeList
// 	accountNodeList := g.AccountNodeList
// 	edgeList := g.EdgeList

// 	// 维护一个Account 和 Transaction 关连的 Map
// 	accountTxMap := make(map[string]map[string]bool)

// 	//记录Transaction访问过 哪些Account
// 	txAccountMap := make(map[string]map[string]bool)

// 	// 初始化 Map
// 	for _, account := range accountNodeList {
// 		accountTxMap[account.Address] = make(map[string]bool)
// 	}

// 	// 初始化 Map
// 	for _, tx := range txList {
// 		txAccountMap[strconv.Itoa(tx.ID)] = make(map[string]bool)
// 	}

// 	// 如果一个 Transaction 指向一个 Account，那么这个 Transaction 与这个 Account 有关连，加入 Map
// 	for _, edge := range edgeList {
// 		from := edge.From
// 		to := edge.To
// 		accountTxMap[to][from] = true
// 		txAccountMap[from][to] = true
// 	}

// 	//循环返回并行的步数，并打印每一步的Transaction
// 	var Step = 0                  //并行的步数
// 	var RemainTxNum = len(txList) //剩余Transaction 数量
// 	for true {

// 		if RemainTxNum == 0 { //如果没有 Transaction 需要执行了跳出循环
// 			break
// 		}

// 		if Step == 2 { //debug
// 			break
// 		}

// 		Step++ //步数加一

// 		var parallelSet []string

// 		for key, value := range txAccountMap {
// 			if len(value) == 0 { //说明 Transaction 没有和任何 Account 有边相连，所以可以直接加入并行
// 				parallelSet = append(parallelSet, key) //加入执行
// 				delete(txAccountMap, key)              //移除执行后的 Transaction
// 				RemainTxNum--
// 				continue
// 			}
// 		}

// 		// 根据 Transaction 和 Account 的关连 Map，只保留与多个 Transaction 相连的 Account
// 		newAccountNodeList := []AccountNode{}
// 		for _, account := range accountNodeList {
// 			if len(accountTxMap[account.Address]) == 1 { //说明 Account 与多个 Transaction 有关连
// 				newAccountNodeList = append(newAccountNodeList, account)
// 			}
// 		}

// 		// 根据 Transaction 和 Account 的关连 Map，只保留与多个 Transaction 相连的 Account 有关的边
// 		newEdgeList := []Edge{}
// 		for _, edge := range edgeList {
// 			if len(accountTxMap[edge.To]) > 1 { //说明Edge To的 Account 与多个 Transaction 有关连，影响并行，需要保留这条边
// 				newEdgeList = append(newEdgeList, edge)
// 			}
// 		}

// 		//更新关系图
// 		accountNodeList = newAccountNodeList
// 		edgeList = newEdgeList

// 		fmt.Println("Step: ", strconv.Itoa(Step))
// 		fmt.Println(parallelSet)
// 	}

// //先建立一张依赖图
// g := BuildDependencyGraph()

// // 获取图的节点和边的信息
// txList := g.TxNodeList
// newAccountNodeList := g.AccountNodeList //筛选过的 Account 信息
// newEdgeList := g.EdgeList               //筛选过的 Edge

// // 记录Account被哪些 Transaction 访问过
// accountTxMap := make(map[string]map[string]bool)
// //记录Transaction访问过 哪些Account
// txAccountMap := make(map[string]map[string]bool)

// // 初始化 Map
// for _, account := range newAccountNodeList {
// 	accountTxMap[account.Address] = make(map[string]bool)
// }

// // 初始化 Map
// for _, tx := range txList {
// 	txAccountMap[strconv.Itoa(tx.ID)] = make(map[string]bool)
// }

// // 如果一个 Transaction 指向一个 Account，那么这个 Transaction 与这个 Account 有关连，加入 Map
// for _, edge := range newEdgeList {
// 	from := edge.From
// 	to := edge.To
// 	accountTxMap[to][from] = true
// 	txAccountMap[from][to] = true
// }

// //循环返回并行的步数，并打印每一步的Transaction
// var Step = 0                  //并行的步数
// var RemainTxNum = len(txList) //剩余Transaction 数量
// for true {
// 	if RemainTxNum == 0 { //如果没有 Transaction 需要执行了跳出循环
// 		break
// 	}

// 	if Step == 10 { //debug
// 		break
// 	}

// 	Step++ //步数加一

// 	var parallelSet []string

// 	for key, value := range txAccountMap {
// 		if len(value) == 0 { //说明 Transaction 没有和任何 Account 有边相连，所以可以直接加入并行
// 			parallelSet = append(parallelSet, key) //加入执行
// 			delete(txAccountMap, key)              //移除执行后的 Transaction
// 			RemainTxNum--
// 			continue
// 		}
// 		if
// 	}

// 	fmt.Println("Step: ", strconv.Itoa(Step))
// 	fmt.Println(parallelSet)
// }

//}
