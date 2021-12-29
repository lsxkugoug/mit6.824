package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

// WkToCor workers向coordinator返回自己已经完成的任务
type WkToCor struct {
	ReqType         int // 0为请求 1为告知已经完成任务
	CompletedTaskId int // 在master 数组里对应filename的 i
}

// CorToWk coordinator向workers颁发任务
type CorToWk struct {
	RespType int // 0为请求map， 1为请求reduce
	/**
	taskName:
		for map phase:
			&……%noMapTask 意思是没有给你的MapTask了，但你得等待，可能有worker挂了.
			如果是 &……%mapTaskCompleted，代表可以进入reduce阶段
			正常情况, 为需要分割的小文件名称
		for reduce phase:
			&……%noReduceTask 意思是没有给你的ReduceTask了，但你得等待，可能有worker挂了.
			如果是 &……%reduceTaskCompleted，代表可以结束了
	*/

	TaskName string //maptask or reducetask, mr-x-y 为y
	NMap     int    // map任务的数量，用于给reduce阶段的machine提供 -x
	NReduce  int    //最终要拆分成的文件数量 mr-x-y y=hash(key) % nReduce
	TaskId   int    // 在master 数组里对应filename的 i

}

// Add your RPC definitions here.

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
