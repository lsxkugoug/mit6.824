package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
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

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// Worker main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	/* 处理map任务 */
	// 直到master告知已经结束, 否则轮询
	fmt.Fprintf(os.Stdout, "================== workers: start map tasks ===================\n")

	for {
		req := WkToCor{0, -1}           //处理map之后完报告任务
		resp := CorToWk{0, "", 0, 0, 0} //接收coordinator分派的任务
		// 申请任务
		fmt.Fprintf(os.Stdout, "workers: woker apply map task!\n")
		call("Coordinator.MapHandler", &req, &resp)
		if resp.TaskName == "*&……%mapTaskCompleted" {
			break
		}
		fmt.Fprintf(os.Stdout, "workers: get respon: %v from coordinator\n", resp)
		// map处理文件
		if resp.TaskName != "*&……%noMapTask" && resp.TaskName != "*&……%mapTaskCompleted" {
			res := mapTaskProcess(mapf, &resp) //使用map函数处理小文件
			if res == false {
				log.Fatalf("mapTask failed taskFileName = %v", resp.TaskName)
			}
			req.ReqType = 1 // 告知coordinator已经完成任务
			req.CompletedTaskId = resp.TaskId
			fmt.Fprintf(os.Stdout, "workers: complete map task %d and inform coordinator\n", resp.TaskId)
			call("Coordinator.MapHandler", &req, &resp)
		} else {
			//fmt.Fprintf(os.Stdout, "workers: get %s, sleep one second\n", resp.TaskName)
			time.Sleep(time.Second)
		}
	}

	fmt.Fprintf(os.Stdout, "==================workers: start reduce task======================\n")

	/* 处理reduce任务 */
	req := WkToCor{0, -1}           //处理map之后完报告任务
	resp := CorToWk{0, "", 0, 0, 0} //接收coordinator分派的任务
	for {
		req = WkToCor{0, -1}           //处理map之后完报告任务
		resp = CorToWk{0, "", 0, 0, 0} //接收coordinator分派的任务
		// 申请任务
		fmt.Fprintf(os.Stdout, "workers: woker apply reduce task!\n")
		call("Coordinator.ReduceHandler", &req, &resp)
		if resp.TaskName == "*&……%reduceTaskCompleted" {
			break
		}
		// reduce处理intermediate
		if resp.TaskName != "*&……%noReduceTask" && resp.TaskName != "&……%reduceTaskCompleted" {
			res := reduceTaskProcess(reducef, &resp) //使用map函数处理小文件
			if res == false {
				log.Fatalf("reduceTask failed, reduceTaskId = %d", resp.TaskId)
			}
			req.ReqType = 1 // 告知coordinator已经完成任务
			req.CompletedTaskId = resp.TaskId
			call("Coordinator.ReduceHandler", &req, &resp)
		} else {
			fmt.Fprintf(os.Stdout, "workers: get %s, sleep one second\n", resp.TaskName)
			time.Sleep(time.Second)
		}
	}
}

/*
将一个fileName里的word转化为 {word, 1} {word, 1} 这种形式进行保存
*/
func mapTaskProcess(mapf func(string, string) []KeyValue, resp *CorToWk) bool {
	fileName := resp.TaskName
	intermediate := []KeyValue{}
	nReduce := resp.NReduce
	// 创建 r 个file, 与json对象数组
	intermediatsFiles := []*os.File{}
	jsonEncoderList := []*json.Encoder{}
	X := resp.TaskId
	for i := 0; i < nReduce; i++ {
		tmpFileName := "mr-" + strconv.Itoa(X) + "-" + strconv.Itoa(i)
		interFile, err := os.Create(tmpFileName)
		if err != nil {
			log.Fatalf("cannot create %v", tmpFileName)
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
		err := intermediatesfile.Close()
		if err != nil {
			return false
		}
	}
	return true
}

/* 拿到taskId之后,读取所有mr-X-taskId的内容 排序 双指针处理 */
func reduceTaskProcess(reducef func(string, []string) string, resp *CorToWk) bool {
	// 保存读取所有 mr-X-taskId 的数据,并保存为键值对
	intermediate := []KeyValue{}
	nMap := resp.NMap
	Y := resp.TaskId
	// 读取所有与Y适配的文件，将文件中的键值对写入intermediate数组里
	for X := 0; X < nMap; X++ {
		tmpFileName := "mr-" + strconv.Itoa(X) + "-" + strconv.Itoa(Y)
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

	oname := "mr-out-" + strconv.Itoa(Y)
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

	err := ofile.Close()
	if err != nil {
		return false
	}

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
