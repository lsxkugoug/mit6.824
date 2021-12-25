package mr

import (
	"log"
	"sync"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

// c Coordinator rpc注册对象
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
func (c *Coordinator) mapHandler(req *wkToCor, reply *corToWk) error {
	c.mu.Lock()
	// worker告知已经完成map任务
	if req.reqType == 1 {
		completedId := req.completedTaskId
		if c.mapTask[completedId] == 1 {
			c.mapTask[completedId] = 2
			c.completedMapNum++
		}
	}
	reply.respType = 0
	reply.nReduce = c.nReduce
	if c.completedMapNum < len(c.mapTask) {
		//表示没有需要的worker handle的map任务,
		//workers将会循环请求直到cordinator发送wokers map任务结束，即所有map任务均已完成
		reply.taskName = "*&……%noMapTask"
	} else {
		reply.taskName = "*&……%mapTaskCompleted"
	}
	for i := 0; i < len(c.mapTask); i++ {
		if c.mapTask[i] == 0 {
			c.mapTask[i] = 1
			reply.taskName = c.files[i]
			reply.taskId = i
			break
		}
	}
	c.mu.Unlock()
	return nil
}
func (c *Coordinator) reduceHandler(req *wkToCor, reply *corToWk) error {
	c.mu.Lock()
	// worker告知已经完成任务
	if req.reqType == 1 {
		completedId := req.completedTaskId
		if c.reduceTask[completedId] == 1 {
			c.reduceTask[completedId] = 2
			c.completedReduceNum++
		}
	}
	reply.respType = 1
	reply.nReduce = c.nReduce
	if c.completedReduceNum < c.nReduce {
		//表示没有需要的worker handle的map任务,
		//workers将会循环请求直到cordinator发送wokers map任务结束，即所有map任务均已完成
		reply.taskName = "*&……%noReduceTask"
	} else {
		reply.taskName = "*&……%reduceTaskCompleted"
	}
	for i := 0; i < len(c.reduceTask); i++ {
		if c.reduceTask[i] == 0 {
			c.reduceTask[i] = 1
			reply.taskName = ""
			reply.taskId = i
			reply.nMap = len(c.mapTask)
			break
		}
	}
	c.mu.Unlock()
	return nil
}

//
// start a thread that listens for RPCs from worker.go
//
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

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	ret := false

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
	var reduceTask []int = make([]int, mapTaskNum)
	c := Coordinator{files: files, mapTask: mapTask, reduceTask: reduceTask, completedMapNum: 0,
		completedReduceNum: 0, nReduce: nReduce}
	/* process map request*/
	for c.completedMapNum != mapTaskNum {
		c.server()
	}
	/* process reduce request*/
	for c.completedReduceNum != nReduce {
		c.server()
	}

	return &c
}
