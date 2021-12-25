package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"time"
)
import "log"
import "net/rpc"
import "hash/fnv"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	/* 处理map任务 */
	req := wkToCor{0, -1}              //处理map之后完报告任务
	resp := corToWk{0, "", -1, -1, -1} //接收coordinator分派的任务
	// 直到master告知已经结束, 否则轮询
	for resp.taskName != "&……%mapTaskCompleted" {
		// 申请任务
		call("Coordinator.mapHandler", req, resp)
		// map处理文件
		if resp.taskName != "*&……%noMapTask" && resp.taskName != "&……%mapTaskCompleted" {
			res := mapTaskProcess(mapf, &resp) //使用map函数处理小文件
			if res == false {
				log.Fatalf("mapTask failed taskFileName = %v", resp.taskName)
			}
			req.reqType = 1 // 告知coordinator已经完成任务
			req.completedTaskId = resp.taskId
			call("Coordinator.mapHandler", req, resp)
		} else {
			time.Sleep(time.Second)
		}
	}
	/* 处理reduce任务 */
	for resp.taskName != "&……%reduceTaskCompleted" {
		// 申请任务
		call("Coordinator.reduceHandler", req, resp)
		// reduce处理intermediate
		if resp.taskName != "*&……%noReduceTask" && resp.taskName != "&……%ReduceTaskCompleted" {
			res := reduceTaskProcess(reducef, &resp) //使用map函数处理小文件
			if res == false {
				log.Fatalf("reduceTask failed, reduceTaskId = %d", resp.taskId)
			}
			req.reqType = 1 // 告知coordinator已经完成任务
			req.completedTaskId = resp.taskId
			call("Coordinator.reduceHandler", req, resp)
		} else {
			time.Sleep(time.Second)
		}
	}
}

/*
将一个fileName里的word转化为 {word, 1} {word, 1} 这种形式进行保存
*/
func mapTaskProcess(mapf func(string, string) []KeyValue, resp *corToWk) bool {
	fileName := resp.taskName
	intermediate := []KeyValue{}
	nReduce := resp.nReduce
	// 创建 r 个file, 与json对象数组
	intermediatsFiles := []*os.File{}
	jsonEncoderList := []*json.Encoder{}
	X := resp.taskId
	for i := 0; i < nReduce; i++ {
		tmpFileName := "mr-" + string(X) + "-" + string(i)
		interFile, err := os.Open(tmpFileName)
		if err != nil {
			log.Fatalf("cannot open %v", tmpFileName)
			return false
		}
		jsonEncoderList = append(jsonEncoderList, json.NewEncoder(interFile))
		intermediatsFiles = append(intermediatsFiles, interFile)
	}
	file, err := os.Open(fileName)
	if err != nil {
		log.Fatalf("cannot open %v", fileName)
		return false
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", fileName)
		return false
	}
	file.Close()
	kva := mapf(fileName, string(content)) // 获取一个file里的所有键值对 {word, 1}
	intermediate = append(intermediate, kva...)
	// 写入相应的键值对
	for _, kv := range intermediate {
		tmpY := ihash(kv.Key) % nReduce
		err := jsonEncoderList[tmpY].Encode(&kv)
		if err != nil {
			log.Fatalf("cannot read %v", fileName)
			return false
		}
	}
	for _, intermediatesfile := range intermediatsFiles {
		intermediatesfile.Close()
	}
	return true
}

/* 拿到taskId之后,读取所有mr-X-taskId的内容 排序 双指针处理 */
func reduceTaskProcess(reducef func(string, []string) string, resp *corToWk) bool {
	// 保存读取所有 mr-X-taskId 的数据,并保存为键值对
	intermediate := []KeyValue{}
	nMap := resp.nMap
	// 接收 r 个file, 与json对象数组
	Y := resp.taskId
	for X := 0; X < nMap; X++ {
		tmpFileName := "mr-" + string(X) + "-" + string(Y)
		interFile, err := os.Open(tmpFileName)
		if err != nil {
			log.Fatalf("cannot open %v", tmpFileName)
			return false
		}
		dec := json.NewDecoder(interFile)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
	}

	sort.Sort(ByKey(intermediate))

	oname := "mr-out-" + string(Y)
	ofile, _ := os.Create(oname)

	i := 0
	for i < len(intermediate) {
		j := i + 1
		for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}

	ofile.Close()

	return true
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, req interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, req, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
