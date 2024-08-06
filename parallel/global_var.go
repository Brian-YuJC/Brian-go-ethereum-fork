package parallel

import (
	"fmt"
	"os"
)

// var Total_op_count_map = make(map[string]int64)

// var Total_op_time_map = make(map[string]int64)

// var op_channel = make(chan map[string]int64, 100000)
var Op_channel = make(chan Msg, 100000)

var WriteFile *os.File = nil

var Op_time_map map[string][]int64

var ch_finish = make(chan bool, 100000)

var ch_output = make(chan BlockOpcodeTime, 100000)

//var CurrentBlockNum int64 = 0

// var CountResMap []*map[string]int64 = make([]*map[string]int64, 20)
// var TimeResMap []*map[string]int64 = make([]*map[string]int64, 20)

//var MaxChanSize int = -1

type Msg struct {
	Opcode       string
	Run_time     int64
	WriteFilePtr *os.File
}

type BlockOpcodeTime struct {
	BlockNum     int64
	OpTimeMap    map[string][]int64
	WriteFilePtr *os.File
}

// func Update_total_op_count_and_time(opcode string, run_time int64) {
// 	// map_value := map[string]int64{
// 	// 	opcode: run_time,
// 	// }
// 	Op_channel <- Msg{opcode: opcode, run_time: run_time}
// }

func Start_channel() {
	go collect_data_from_channel()
	// for i := 0; i < thresdCnt; i++ {
	// 	go collect_data_from_channel(i)
	// }
}

//	func max(a int, b int) int {
//		if a >= b {
//			return a
//		}
//		return b
//	}
var blockOpTime BlockOpcodeTime = BlockOpcodeTime{BlockNum: 0}

func collect_data_from_channel() {
	// for {
	// 	ch_data := <-op_channel
	// 	for opcode, run_time := range ch_data {
	// 		_, op_ok := Total_op_count_map[opcode]
	// 		if !op_ok {
	// 			Total_op_count_map[opcode] = 0
	// 			Total_op_time_map[opcode] = 0
	// 		}
	// 		Total_op_count_map[opcode] += 1
	// 		Total_op_time_map[opcode] += run_time
	// 	}
	// }
	// op_count_map := make(map[string]int64)
	// op_time_map := make(map[string]int64)
	// TimeResMap[res_index] = &op_time_map
	// CountResMap[res_index] = &op_count_map

	for {
		//MaxChanSize = max(MaxChanSize, len(Op_channel))
		ch_data := <-Op_channel
		// if _, op_ok := Total_op_count_map[ch_data.Opcode]; !op_ok {
		// 	Total_op_count_map[ch_data.Opcode] = 0
		// 	Total_op_time_map[ch_data.Opcode] = 0
		// }
		// Total_op_count_map[ch_data.Opcode] += 1
		// Total_op_time_map[ch_data.Opcode] += ch_data.Run_time
		if ch_data.Opcode == "BlockNumber" {
			ch_output <- blockOpTime
			blockOpTime = BlockOpcodeTime{BlockNum: ch_data.Run_time, WriteFilePtr: ch_data.WriteFilePtr, OpTimeMap: make(map[string][]int64)}
			continue
		}
		//fmt.Fprintln(ch_data.WriteFilePtr, ch_data.Opcode, ch_data.Run_time)
		if _, ok := blockOpTime.OpTimeMap[ch_data.Opcode]; !ok {
			blockOpTime.OpTimeMap[ch_data.Opcode] = make([]int64, 0)
		}
		blockOpTime.OpTimeMap[ch_data.Opcode] = append(blockOpTime.OpTimeMap[ch_data.Opcode], ch_data.Run_time)
	}
}

//var opcodeCnt = int64(0)

func Output() {
	for {
		res := <-ch_output
		if res.BlockNum == 0 {
			continue
		}
		fmt.Fprintln(res.WriteFilePtr, "BlockNumber", res.BlockNum)

		for k, v := range res.OpTimeMap {
			fmt.Fprint(res.WriteFilePtr, k)
			for _, time := range v {
				//opcodeCnt++
				fmt.Fprint(res.WriteFilePtr, " ", time)
			}
			fmt.Fprint(res.WriteFilePtr, "\n")
		}
		//fmt.Fprintln(res.WriteFilePtr, "Opcode Count", opcodeCnt)
		//res.WriteFilePtr.Close()
		ch_finish <- true
	}

}

// func MergeMap() {
// 	for _, Map := range CountResMap {
// 		if Map == nil {
// 			break
// 		}
// 		for k, v := range *Map {
// 			if _, ok := Total_op_count_map[k]; !ok {
// 				Total_op_count_map[k] = 0
// 			}
// 			Total_op_count_map[k] += v
// 		}
// 	}
// 	for _, Map := range TimeResMap {
// 		if Map == nil {
// 			break
// 		}
// 		for k, v := range *Map {
// 			if _, ok := Total_op_time_map[k]; !ok {
// 				Total_op_time_map[k] = 0
// 			}
// 			Total_op_time_map[k] += v
// 		}
// 	}
// }

// func Print_total_op_count_and_time() {
// 	for {
// 		if len(Op_channel) == 0 {
// 			//MergeMap()
// 			for op_code, op_count := range Total_op_count_map {
// 				op_run_time := Total_op_time_map[op_code]
// 				fmt.Println("Opcode name is: ", op_code, ". Total Run time as nanos: ", op_run_time, ". Total Count is: ", op_count)
// 			}
// 			//fmt.Println("Max channel size: ", MaxChanSize)
// 			break
// 		}
// 	}
// }

func Hold(BlockNum int64) {
	count := uint64(0)
	for {
		if count == uint64(BlockNum)-1 {
			ch_output <- blockOpTime
			for {
				if <-ch_finish {
					//fmt.Println("opcode Cnt", opcodeCnt)
					return
				}
			}
		}
		if <-ch_finish {
			count++
		}
	}
}
