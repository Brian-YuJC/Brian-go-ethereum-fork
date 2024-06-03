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

// 打印并行顺序
func PrintParallelOrder(orderMap []int, txNum int, originalOrderMap []int) {

	//-----------------------Method 2 Using Order Map------------------------------
	fmt.Println("-------------------------------Print Parallel Order-------------------------------")
	batch := 0
	execTxNum := 0 //执行完的交易数量
	for true {
		//fmt.Println(execTxNum)
		if execTxNum == txNum { //全部 Tx 执行完毕
			break
		}

		batch++
		var parallelSet = make([]bool, txNum) //判断当前轮次内执行过的 Tx
		var n = 0                             //本轮执行的 Tx 数量
		for tx, value := range orderMap {
			if value == -1 { //交易 i 没有前置交易，可以执行
				execTxNum++
				n++
				orderMap[tx] = -2 //已经执行过的标志
				parallelSet[tx] = true
			}
		}

		//维护orderMap
		for tx, lastDependency := range orderMap {
			if lastDependency != -2 {
				if parallelSet[lastDependency] { //如果前置的依赖已经被执行了，即表示该交易下一轮可以执行了
					orderMap[tx] = -1
				}
			}
		}

		fmt.Println("Parallel Execution Round: ", batch)
		fmt.Printf("The number of Transaction can be parallelly executed: %d\n", n)
		for tx, value := range parallelSet {
			if value {
				fmt.Print(tx)
				if originalOrderMap[tx] != -1 {
					fmt.Print("\t(After Tx ", originalOrderMap[tx], " is executed)")
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}
}

// 获取并行最优加速比（cpu 数量没有上限）
func GetMaxiumSpeedUp(orderMap []int) float64 {

	//Check for correctness!!!!!!!!!!!!
	fmt.Println("\nGetMaxiumSpeedUp()")                                             //Check for correctness!!!!!!!!!!!!
	print("-------------------------Print Execution Time-------------------------") //Check for correctness!!!!!!!!!!!!
	for i, value := range TxExecuteTime {
		fmt.Printf("Tx %d execution time: ", i)
		fmt.Print(value, "\n")
	}

	// 计算加速比
	var LinearExecuteTime float64 = 0 //线性执行时间
	for _, time := range TxExecuteTime {
		LinearExecuteTime += time
	}

	var MaxParallelExecTime float64 = 0 //并行的最大运行时间
	for i, _ := range orderMap {
		var t float64 = 0
		currentTx := i
		for true {
			fmt.Print(currentTx, " <- ") //Check for correctness!!!!!!!!!!!!
			t += TxExecuteTime[currentTx]
			if orderMap[currentTx] == -1 {
				break
			}
			currentTx = orderMap[currentTx]
		}
		fmt.Print("Total Time: ", t, "\n") //Check for correctness!!!!!!!!!!!!
		MaxParallelExecTime = max(MaxParallelExecTime, t)
	}

	fmt.Print("MaxParallelExecTime: ", MaxParallelExecTime, "\n")       //Check for correctness!!!!!!!!!!!!
	fmt.Print("LinearExecuteTime: ", LinearExecuteTime, "\n")           //Check for correctness!!!!!!!!!!!!
	fmt.Print("SpeedUp: ", LinearExecuteTime/MaxParallelExecTime, "\n") //Check for correctness!!!!!!!!!!!!

	var SpeedUp float64 = LinearExecuteTime / MaxParallelExecTime

	return SpeedUp
}

// 计算 Transaction 两两之间有无依赖关系
// 返回值是tx-tx之间关系，调用顺序 map 和 加速比
func BuildTxRelationGraph() ([][]string, []int, float64) {
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
	txNum := len(txList) //交易总数
	//-------------------------------------------------Final Result----------------------------------------------------------
	var list [][]string = make([][]string, txNum) //存放地 i 个 Transaction 和与之关连的 Transaction
	var orderMap = make([]int, txNum)             //存放执行某个Transaction 之前必须完成执行的最后一个依赖交易（即执行完此依赖交易才可以执行此交易） tx-dependent tx
	//-------------------------------------------------Final Result----------------------------------------------------------
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

		//维护 orderMap
		if len(list[i]) == 0 { //没有前置依赖交易
			orderMap[i] = -1
		} else {
			t, _ := strconv.Atoi(list[i][len(list[i])-1])
			orderMap[i] = t //选取序号最大的交易
		}

		//打印依赖结果
		fmt.Printf("Tx: %d\nDependent Tx: ", i)
		for j := 0; j < len(list[i]); j++ {
			fmt.Print(list[i][j], " ")
		}
		fmt.Println("\n")
	}

	//	打印信息
	var orderMapCopy = make([]int, txNum)
	copy(orderMapCopy, orderMap)
	PrintParallelOrder(orderMapCopy, txNum, orderMap)

	//-----------------------Get Parallel Order Method 1------------------------------
	//打印并行顺序
	// fmt.Println("-------------------------------Print Parallel Order-------------------------------")
	// batch := 0
	// frontTx := make(map[string]string) //存放合约执行的前置合约（就是必须先执行完前置合约才能执行当前合约，不然有冲突）
	// for true {
	// 	if len(tx_tx_map) == 0 { //如果未执行的 Transaction空了则跳出循环
	// 		break
	// 	}

	// 	batch++
	// 	var parallelSet []string //本批执行的列表
	// 	for tx, dependency := range tx_tx_map {
	// 		//fmt.Print(tx, "->", len(dependency), " ")
	// 		if len(dependency) == 0 { //出度为 0
	// 			parallelSet = append(parallelSet, tx) //加入并行
	// 			delete(tx_tx_map, tx)                 //从 map 中移除
	// 		}
	// 	}

	// 	//更新维护剩余 Transaction 与 Transaction 之间的关连(删除与已经加入并行的 Transaction 的关连)
	// 	for _, deletedTx := range parallelSet {
	// 		for tx, dependency := range tx_tx_map {
	// 			if _, ok := dependency[deletedTx]; ok && len(dependency) == 1 { //如果只剩一个依赖，而且该依赖以及执行则该依赖为前置合约
	// 				frontTx[tx] = deletedTx
	// 			}
	// 			delete(dependency, deletedTx)
	// 		}
	// 	}

	// 	//Print Result
	// 	sort.Slice(parallelSet, func(i, j int) bool { //按 Tx 先后排个序（为了展示目的）
	// 		I, _ := strconv.Atoi(parallelSet[i])
	// 		J, _ := strconv.Atoi(parallelSet[j])
	// 		return I < J
	// 	})
	// 	fmt.Println("Parallel Execution Round: ", batch)
	// 	fmt.Printf("The number of Transaction can be parallelly executed: %d\n", len(parallelSet))
	// 	for _, tx := range parallelSet {
	// 		if len(frontTx[tx]) > 0 {
	// 			fmt.Print(tx, "\t(After Tx ", frontTx[tx], " is executed)\n")
	// 		} else {
	// 			fmt.Println(tx)
	// 		}
	// 	}
	// 	fmt.Println("\n")
	// }

	return list, orderMap, GetMaxiumSpeedUp(orderMap)
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
