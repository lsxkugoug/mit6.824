package mr

import (
	"fmt"
	"log"
	"sync"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

// Coordinator c rpc注册对象
type Coordinator struct {
	// Your definitions here.
	files              []string // 需要处理的文件名
	mapTask            []int    // mapTask的完成情况 0_ 没有派发 1_已派发 2_已完成
	reduceTask         []int    // mapTask的完成情况 0_ 没有派发 1_已派发 2_已完成
	completedMapNum    int      //已完成Map任务数量
	completedReduceNum int      //已完成的reduce任务数量
	nReduce            int      //需要的reduce数量
	mu                 sync.Mutex
}

// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) MapHandler(req *WkToCor, reply *CorToWk) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// worker告知已经完成map任务
	if req.ReqType == 1 {
		completedId := req.CompletedTaskId
		if c.mapTask[completedId] == 1 {
			c.mapTask[completedId] = 2
			c.completedMapNum++
		}
		return nil
	}
	reply.NReduce = c.nReduce
	reply.RespType = 0
	reply.NMap = len(c.mapTask)
	if c.completedMapNum < len(c.mapTask) {
		//表示没有需要的worker handle的map任务,
		//workers将会循环请求直到cordinator发送wokers map任务结束，即所有map任务均已完成
		reply.TaskName = "*&……%noMapTask"
	} else {
		reply.TaskName = "*&……%mapTaskCompleted"
	}
	//尝试找到没有分配的任务并分配过去
	for i := 0; i < len(c.mapTask); i++ {
		if c.mapTask[i] == 0 {
			c.mapTask[i] = 1
			reply.TaskName = c.files[i]
			reply.TaskId = i
			fmt.Fprintf(os.Stdout, "coordinator: distribute map "+
				"task, filename: %v, taskId: %v\n", reply.TaskName, reply.TaskId)
			break
		}
	}
	return nil
}
func (c *Coordinator) ReduceHandler(req *WkToCor, reply *CorToWk) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// worker告知已经完成任务
	if req.ReqType == 1 {
		completedId := req.CompletedTaskId
		if c.reduceTask[completedId] == 1 {
			c.reduceTask[completedId] = 2
			c.completedReduceNum++
		}
		return nil
	}
	reply.RespType = 1
	reply.NReduce = c.nReduce
	if c.completedReduceNum < c.nReduce {
		//表示没有需要的worker handle的map任务,
		//workers将会循环请求直到cordinator发送wokers map任务结束，即所有map任务均已完成
		reply.TaskName = "*&……%noReduceTask"
	} else {
		reply.TaskName = "*&……%reduceTaskCompleted"
	}
	for i := 0; i < len(c.reduceTask); i++ {
		if c.reduceTask[i] == 0 {
			c.reduceTask[i] = 1
			reply.TaskName = ""
			reply.TaskId = i
			reply.NMap = len(c.mapTask)
			fmt.Fprintf(os.Stdout, "coordinator: distribute reduce "+
				"task, filename: %v, taskId: %v\n", reply.TaskName, reply.TaskId)
			break
		}
	}
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	// 注册rpc handler
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// Done
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := false
	fmt.Fprintf(os.Stdout, "coordinator: completedMapNum: %d, completedReduceNum: %d \n",
		c.completedMapNum, c.completedReduceNum)
	if c.completedMapNum == len(c.mapTask) && c.completedReduceNum == c.nReduce {
		return true
	}

	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	mapTaskNum := len(files)
	var mapTask = make([]int, mapTaskNum)
	var reduceTask = make([]int, nReduce)

	c := Coordinator{files: files, mapTask: mapTask, reduceTask: reduceTask, completedMapNum: 0,
		completedReduceNum: 0, nReduce: nReduce}
	fmt.Fprintf(os.Stdout, "coordinator: coordinator parameter: %v\n", c)
	/* process map request*/
	c.server()

	return &c
}
