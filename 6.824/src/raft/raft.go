package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"math/rand"
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"
	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type raftLog struct {
	Command interface{}
	Term    int
	Index   int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu          sync.Mutex          // Lock to protect shared access to this peer's state
	peers       []*labrpc.ClientEnd // RPC end points of all peers
	persister   *Persister          // Object to hold this peer's persisted state
	me          int                 // this peer's index into peers[]
	dead        int32               // set by Kill()
	status      int                 // 0 followers, 1 candidate, 2 leader
	currentTerm int                 // 自己的term
	voteFor     int                 // 投过的candidateId 每轮term只投给一个人
	logs        []raftLog           // log
	applyCh     chan ApplyMsg       // A channel to send apply message

	/* 2A leader election */
	// hearBeat计时器
	lastHeartBeat    int64 //上一次收到heartbeat时间
	heartBeatFre     int64 // leader发送heartbeat的频率20ms
	heartBeatTimeout int64 // 心跳超时时间 同时也是选举超时时间
	//选举计时器
	electionStartTime int64 //选举开始时间
	// 选票计数器
	grantNum int

	/* 2B log replication */
	commitIndex int   // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int   // index of highest log entry applied to state machine (initialized to 0, increases monotonically)
	nextIndex   []int // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex  []int // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	/* 2C */
	/* 2D */

	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
}

/* 检查 leader 的heartbeat 超时, true代表未超, false代表超时 */
func (rf *Raft) checkHeartBeat() bool {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if time.Now().UnixMilli()-rf.lastHeartBeat > rf.heartBeatTimeout {
		return false
	}
	return true
}

// return currentTerm and whether this server
// believes it is the leader. used by tester, don't change it.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	var term int
	isleader := false
	// Your code here (2A).
	term = rf.currentTerm
	if rf.status == 2 {
		//fmt.Fprintf(os.Stdout, "我是%d, term为:%d, 为leader\n", rf.me, rf.currentTerm)
		isleader = true
	}
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

/*=================== rpc definition ========================*/
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	Term         int // candidate's term
	CandidateId  int // candidate requesting vote
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term	of candidate's	last log entry
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	Term       int  // candidate's term
	VoteGanted bool // true means candidate received vote
}
type AppendEntriesArgs struct {
	Type         int       // 0 heartBeat, 1 try to synchronize log, 2 notify followers to apply logs
	Term         int       //leader's term
	LeaderId     int       //so follower can redirect client
	PrevLogIndex int       //index of log entry immediately preceding new ones
	PrevLogTerm  int       // term of prevLogIndex entry
	Entries      []raftLog // entries to store (empty for heartbeat may send more than one for efficiency)
	LeaderCommit int       //leader's commitIndex
}
type AppendEntriesReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
}

/*======================== rpc structure end =================================== */

/*======================== RPC handlers ===========================*/

// RequestVoteHandler handler 用于处理vote请求
func (rf *Raft) RequestVoteHandler(args *RequestVoteArgs, reply *RequestVoteReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Your code here (2A, 2B).
	reply.Term = rf.currentTerm
	// 如果请求的term小于自己当前的term,消息过期
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGanted = false
		return
	}
	//正常处理逻辑
	// 如果收到了比自己大的term 请求, 则修改自己的term并voteFor置为-1, 如果身份是candidate,回退为followers
	if args.Term > rf.currentTerm {
		if rf.status != 0 {
			//fmt.Fprintf(os.Stdout, "我是%d号,我的term是:%d, 我的status是%d 我回退为follwers, 我收到了%d的vote请求,"+
			//	"他的term为%d \n",
			//	rf.me, rf.currentTerm, rf.status, args.CandidateId, args.Term)
		}

		rf.voteFor = -1
		rf.currentTerm = args.Term
		rf.status = 0
	}
	myLastLogIndex := -1
	myLastLogTerm := -1
	if len(rf.logs) > 0 {
		myLastLogTerm = rf.logs[len(rf.logs)-1].Term
		myLastLogIndex = len(rf.logs) - 1
	}
	// If votedFor is null or candidateId, and candidate’s log is at least as up-to-date as receiver’s log, grant vote
	if (rf.voteFor == -1 || rf.voteFor == args.CandidateId) &&
		args.LastLogIndex >= myLastLogIndex && args.LastLogTerm >= myLastLogTerm {
		rf.voteFor = args.CandidateId
		reply.VoteGanted = true
		//rf.status = 1
		//fmt.Fprintf(os.Stdout, "我是%d号,我的term是:%d, 我的status是%d, 我投给%d号candiidate\n",
		//	rf.me, rf.currentTerm, rf.status, args.CandidateId)
	} else {
		reply.VoteGanted = false
	}
}

// AppendEntriesHandler 用于处理appendEntries和heartBeat请求
func (rf *Raft) AppendEntriesHandler(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 不论是什么情况 每次都进行heartbeat控制
	if rf.currentTerm > args.Term {
		reply.Term = rf.currentTerm
		return
	}
	rf.lastHeartBeat = time.Now().UnixMilli()
	// 假设此时状态为candidate且收到了其他人的心跳或者append请求，且其他人的心跳还大于>=此时自己的,说明发出该心跳的也是一个candidate,并胜选
	if args.Term > rf.currentTerm {
		//fmt.Fprintf(os.Stdout, "我是%d号,我的term是:%d, 我的status是%d 我回退为follwers, 我收到了%d的心跳,"+
		//	"他的term为%d \n",
		//rf.me, rf.currentTerm, rf.status, args.LeaderId, args.Term)
		rf.currentTerm = args.Term //变为新的term
		rf.status = 0              // candidate竞选失败 回退至followers 或者 旧leader上线变为followers
	}
	// 2.如果是日志同步
	reply.Success = false
	if args.Type == 1 {
		// 如果preLogIndex == -1, 直接进行复制
		if args.PrevLogIndex == -1 {
			rf.logs = args.Entries
			//fmt.Printf("I am %d, 数组长度%d复制成功 %v\n", rf.me, len(rf.logs), rf.logs)
			reply.Success = true
			return
		}

		// 正常情况, 遍历一遍判断preLog是否存在
		for i := len(rf.logs) - 1; i >= 0; i-- {
			if rf.logs[i].Index == args.PrevLogIndex && rf.logs[i].Term == args.PrevLogTerm {
				rf.logs = append(rf.logs[:i+1], args.Entries...)
				reply.Success = true
				break
			}
		}
		return
	}
	// 3. 如果是notify commit
	if args.Type == 2 {
		// verify whether I have the leaderCommitIdex, since the machine may offline before
		CommitIdx := -1
		for i := len(rf.logs) - 1; i >= 0; i-- {
			if rf.logs[i].Index == args.LeaderCommit {
				CommitIdx = i
			}
		}
		if CommitIdx == -1 {
			return
		}
		go rf.ApplyLogs(CommitIdx)
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVoteHandler", args, reply)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if reply.VoteGanted == true {
		rf.grantNum += 1
	}
	// 自己回退为followers 另一个candidate的term更大,表示比自己更快
	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.status = 0
	}
	return ok
}

// logPrc: log正在处理的log的index 注意不是log[x].Index, 是x
func (rf *Raft) synLogs(server int, logPrcIdx int) {
	// 除非CALL失败 否则一直CALL
	for {
		// 说明leader挂了
		if logPrcIdx == -1 {
			return
		}
		rf.mu.Lock()
		preLogIdx := -1
		preLogTerm := -1
		if logPrcIdx > 1 {
			preLogIdx = rf.logs[logPrcIdx-1].Index
			preLogTerm = rf.logs[logPrcIdx-1].Term
		}
		args := AppendEntriesArgs{
			Type:         1,
			Term:         rf.currentTerm,
			LeaderId:     rf.me,
			PrevLogIndex: preLogIdx,
			PrevLogTerm:  preLogTerm,
			Entries:      rf.logs[logPrcIdx:],
		}
		reply := AppendEntriesReply{}
		rf.mu.Unlock()
		//fmt.Printf("args = %v\n", args)
		callRes := rf.peers[server].Call("Raft.AppendEntriesHandler", &args, &reply)

		// 如果断线
		if !callRes {
			return
		}

		rf.mu.Lock()
		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.status = 0
			return
		}
		if reply.Term > rf.currentTerm {
			rf.status = 0
			rf.currentTerm = reply.Term
			break
		}
		// 如果失败 准备下一轮的共识
		if reply.Success == false {
			print("共识失败\n")
			logPrcIdx -= 1
		}

		if reply.Success == true {
			//fmt.Printf("共识成功%d, %d号机, 共有%d号\n", logPrcIdx, server, len(rf.peers))
			rf.nextIndex[server] = rf.logs[len(rf.logs)-1].Index + 1 //指向下一个要发送的index
			rf.matchIndex[server] = rf.logs[len(rf.logs)-1].Index    // 已经同步的Index
			rf.mu.Unlock()
			//check logs that whether should be committed(majority)
			//Besides, if committed, notify other nodes to keep up
			rf.checkCommit()
			break
		}
		rf.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}

}

func (rf *Raft) checkCommit() {
	rf.mu.Lock()
	for i := len(rf.logs) - 1; i >= 0; i-- {
		vote := 0
		for j := 0; j < len(rf.matchIndex); j++ {
			if rf.matchIndex[j] >= rf.logs[i].Index {
				vote++
			}
		}
		if rf.logs[i].Term == rf.currentTerm && vote >= len(rf.peers)/2 {
			rf.commitIndex = rf.logs[i].Index
			rf.mu.Unlock()
			go rf.ApplyLogs(i)
			go rf.notifyCommit()
			rf.mu.Lock()
			break
		}
	}
	rf.mu.Unlock()
}

// when leadr commit a log, notify other nodes to commit it either.
func (rf *Raft) notifyCommit() {
	rf.mu.Lock()
	for server := range rf.peers {
		args := AppendEntriesArgs{
			Type:         2,
			Term:         rf.currentTerm,
			LeaderId:     rf.me,
			LeaderCommit: rf.commitIndex,
		}
		reply := AppendEntriesReply{}
		rf.mu.Unlock()
		Call := rf.peers[server].Call("Raft.AppendEntriesHandler", &args, &reply)
		if Call == false {
			fmt.Printf("Call 失败")
		}
		rf.mu.Lock()
	}
	rf.mu.Unlock()
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//TODO 如果发现log里没有保存数据 是不是要导入 不然不知道上次的index是多少了 2B不用考虑
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 如果log里没有数据, 则设置为0,0 TODO 在2D的时候 如果发现logs没数据 应该要加载 然后通过上一个的index获取这一轮的index
	index := 0
	term := rf.currentTerm
	if len(rf.logs) > 0 {
		index = 0
	}
	isLeader := true
	if rf.status != 2 {
		isLeader = false
		return index, term, isLeader
	}
	fmt.Printf("我是%d leader,start: 我的status:%d, 且我的term是 %d\n", rf.me, rf.status, rf.currentTerm)

	// Your code here (2B).
	// add new log entry to local field
	newLogIdx := 1
	if len(rf.logs) > 0 {
		newLogIdx = rf.logs[len(rf.logs)-1].Index + 1
	}
	fmt.Printf("append %v log\n", raftLog{Term: term, Command: command, Index: newLogIdx})
	rf.logs = append(rf.logs, raftLog{Term: term, Command: command, Index: newLogIdx})
	// 尝试同步到其他节点上 nextIndex在leaderBH里初始化
	lastLogIndex := -1
	if len(rf.logs) > 0 {
		lastLogIndex = rf.logs[len(rf.logs)-1].Index
	}
	for i := range rf.nextIndex {
		if lastLogIndex >= rf.nextIndex[i] {
			//fmt.Println("syblogs ", i, rf.nextIndex[i])
			go rf.synLogs(i, len(rf.logs)-1)
		}
	}

	return index, term, isLeader
}

// for leader and followers and send the logs to the tester
// logIdx is the idx of rf.logS corresponding to commitedIdx
func (rf *Raft) ApplyLogs(logIdx int) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// fmt.Printf("%d, %d, %d, %d, %v\n", rf.me, rf.status, rf.lastApplied, rf.commitIndex, rf.logs)
	// i 为lastApplied
	i := logIdx
	for ; i >= 0; i-- {
		if rf.logs[i].Index == rf.lastApplied {
			break
		}
	}
	// 若已经apply过了
	if i == logIdx {
		return
	}
	// if no log
	if i == -1 {
		i = 0
	}
	for ; i <= logIdx; i++ {
		rf.applyCh <- ApplyMsg{
			CommandValid: true,
			Command:      rf.logs[i].Command,
			CommandIndex: rf.logs[i].Index,
		}
		rf.lastApplied = rf.logs[i].Index
		fmt.Printf("I am %d, apply %v\n", rf.me, ApplyMsg{
			CommandValid: true,
			Command:      rf.logs[i].Command,
			CommandIndex: rf.logs[i].Index,
		})
	}
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

/* ==============三个角色=============== */
func (rf *Raft) followerBh() {
	rf.mu.Lock()
	rf.lastHeartBeat = time.Now().UnixMilli() //每次转化成followers的时候 重置timeout时间
	rf.heartBeatTimeout = GetRdHbTo()
	rf.mu.Unlock()
	for {
		// 1. 查看是否需要进行选举 如果是转化为candidate状态
		if !rf.checkHeartBeat() {
			rf.mu.Lock()
			rf.status = 1
			rf.mu.Unlock()
			return
		}
		// 2. 处理leader election信息 已由RequestVoteHandler处理
		time.Sleep(10 * time.Millisecond)
	}
}

func (rf *Raft) candidateBh() {
	// candidate 有三种情况
	// 1.选举失败：如果接收到心跳且term大于自己, 变回成followers
	// 2.如果electionTimeout则重复选举
	// 3.选举成功：自己成为leader

	//1. 退出条件 收到新的leader心跳并发现收到的心跳大于等于自己的currentTerm, 这个逻辑在AppendEntriesHandler里处理,发现这种情况且状态为candidate时, 改变candidate状态
	//2. 发送leader election rpc
	//变量设置
	rf.mu.Lock()
	rf.heartBeatTimeout = GetRdHbTo()
	rf.electionStartTime = time.Now().UnixMilli()
	rf.currentTerm += 1
	rf.grantNum = 1
	rf.voteFor = rf.me // 不投给其他人
	lastLogTerm := -1
	lastLogIndex := -1
	//fmt.Fprintf(os.Stdout, "我是%d candidate, 尝试选举, term为 %d\n", rf.me, rf.currentTerm)
	if len(rf.logs) > 0 {
		lastLogTerm = rf.logs[len(rf.logs)-1].Term
		lastLogIndex = len(rf.logs) - 1
	}
	rf.mu.Unlock()
	// 发送请求 这里对于每个接收者均应开一个线程去处理结果, 不然一个时间等待很长会直接导致选举超时
	for peer, _ := range rf.peers {
		if peer == rf.me {
			continue
		}
		rf.mu.Lock()
		args := RequestVoteArgs{Term: rf.currentTerm, CandidateId: rf.me,
			LastLogIndex: lastLogIndex, LastLogTerm: lastLogTerm}
		reply := RequestVoteReply{}
		rf.mu.Unlock()
		go rf.sendRequestVote(peer, &args, &reply)
	}
	// 超时逻辑
	rf.mu.Lock()
	rfElectionStartTime := rf.electionStartTime
	rfElectionTimeout := rf.heartBeatTimeout
	rf.mu.Unlock()
	for time.Now().UnixMilli()-rfElectionStartTime <= rfElectionTimeout {
		// 成功竞选 leader
		rf.mu.Lock()
		if rf.grantNum >= (len(rf.peers)+1)/2 {
			// 有可能在这里的时候已经变回了followers
			if rf.status == 0 {
				return
			}
			//fmt.Fprintf(os.Stdout, "我是%d号,我的term是:%d, 一共%d个人, 收到%d张票\n",
			//	rf.me, rf.currentTerm, len(rf.peers), rf.grantNum)
			rf.status = 2
			rf.mu.Unlock()
			return
		}
		rf.mu.Unlock()
		//如果不睡眠,因为这个锁覆盖了很大一部分循环,所以会导致其他地方的线程无法运行不能更新其变量,共识难以达成
		time.Sleep(15 * time.Microsecond)
	}
	// 否则 1.重新开一轮选举同时随机化heartbeat timeout 2.竞选失败 回退为followers
}

func (rf *Raft) leaderBh() {
	//成为leader之后立即发送一次心跳
	//fmt.Fprintf(os.Stdout, "我是%d, term为%d leader\n", rf.me, rf.currentTerm)
	for peer, _ := range rf.peers {
		go rf.sendHeart(peer)
	}
	lastSendHeart := time.Now().UnixMilli()
	// 初始化nextIndex[]为最后的Index + 1, matchIndex[]
	rf.mu.Lock()
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for i := 0; i < len(rf.peers) && len(rf.logs) > 0; i++ {
		rf.nextIndex[i] = rf.logs[len(rf.logs)-1].Index + 1
	}
	rf.mu.Unlock()
	for {
		rf.mu.Lock()
		if rf.status != 2 {
			rf.mu.Unlock()
			break
		}
		rf.mu.Unlock()
		// 1.定时发送heartBeat 每一个peer都需要一个线程进行监听
		rf.mu.Lock()
		if time.Now().UnixMilli()-lastSendHeart > rf.heartBeatFre {
			for peer, _ := range rf.peers {
				go rf.sendHeart(peer)
			}
			lastSendHeart = time.Now().UnixMilli()
		}
		rf.mu.Unlock()

		// 2B 同步日志部分写在start方法里 代表有新的request进来
		time.Sleep(15 * time.Millisecond)
	}
}

/*===============================*/

/*============自定义线程函数================*/

// leader发送心跳
func (rf *Raft) sendHeart(server int) {
	args := AppendEntriesArgs{Type: 0, LeaderId: rf.me}
	reply := AppendEntriesReply{}
	ok := rf.peers[server].Call("Raft.AppendEntriesHandler", &args, &reply)
	if !ok {
		//fmt.Println("========== sendHeart call failed ================")
	}
	rf.mu.Lock()
	if reply.Term > rf.currentTerm {
		//说明leader是旧leader应该滚蛋
		rf.status = 0
		rf.currentTerm = reply.Term
	}
	rf.mu.Unlock()
}

/* =========================== */
// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep()
		// 根据节点属性处理不一样的事情
		rf.mu.Lock()
		raftStatus := rf.status
		rf.mu.Unlock()
		switch raftStatus {
		case 0:
			rf.followerBh()
		case 1:
			rf.candidateBh()
		case 2:
			rf.leaderBh()
		}
	}
}

// GetRdHbTo 获取随机heartBeatTimeout
func GetRdHbTo() int64 {
	return int64(400 + rand.Intn(400))
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	// 2A
	heartBeatTimeout := GetRdHbTo()
	rf := &Raft{peers: peers,
		persister:        persister,
		me:               me,
		dead:             0,
		status:           0,
		currentTerm:      0,
		voteFor:          -1,
		heartBeatFre:     100,
		heartBeatTimeout: heartBeatTimeout,
		commitIndex:      0,
		lastApplied:      0,
		applyCh:          applyCh}

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()
	// 建立线程去接收heartBeat,更新rf的变量, 由AppendEntriesHandler RPC去做

	return rf
}
