## lab2 RAFT

**raft的简要流程(无故障)**

所有的请求都将转发给leader

1. client向primary发送请求
2. primary对所有子节点proposal
3. 子节点投票，但不持久化数据，只是将该操作保存在自己的Log里。然后，将自己的投票结果返回primary
4. primary计算票数，如果是majority, 向子节点发送"可以持久化数"据消息
5. primary确定子节点已经持久化数据之后，将操作成功结果返回给client

#### 五条公理

1. 选举安全特性：对于给定的一个任期号, 最多只会有一个leader被选举出来。每一个replica只有一张选票
2. Leader只附加原则：



#### 1. leader election:

##### 1.1 流程

![image-20211228085923282](C:\Users\lsxku\Desktop\share_ubuntu\mit6.824\pic\raft-leaderElection.png)



1. 每个节点初始状态均为follower, 维护一个计时器, 如果leader的heartbeat在规定时间没到，则发起leader election请求, 且自己变为candidate状态尝试选自己为leader。 如果收到了heartbeat，则重置自己的heartbeat计时器。
2. 在发送选举请求之后,  维护一个选举计时器, 如果在计时器范围内收到majority同意(自己永远投给自己)，则成为leader。成为Leader后同步日志并发送自己的心跳。 如果没有收到majority同意, 则重新开始下一次选举。这里还有一个要注意的, 如果此时有两个以上的candidate试图成为leader，且另一个candidate收到了leader的心跳(在试图成为leader的过程中)，则candidate重新回归followers状态。
3. 如果leader变更之后, 旧的leader重新上线了, 此时集群内就有两个leader了。怎么避免这个问题？ 在leader向followers发送心跳的过程中, 会发送自己的term, 每个followers会记录最大的term, 如果一个followers成为了leader，则该followers的term+1。当旧leader重新上线之后, 旧leader与新leader一起发送心跳(包含自己的term)，旧leader发现自己的term比新的要小，则退位。

4. 在投票中还有一个问题, 如果所有followers的心跳计时器是一样的, 且followers都在同一个时间启动。当leader宕机的时候，所有的followers将会同时发起leader election请求。因为每次选举节点只会把票投给自己，就无法达成共识。要处理这种情况，可以将followers的心跳计时器和选举超时计时器randomlize，比如设置 在 150~300ms均可，这样就不会发生同时超时同时发起选举的问题。每次重置timer的时候，随机化timeout时间

   **restriction：** 

   在election过程中, 只有 voter发现自己的log中最大的term小于等于candidate竞选发来的term时，才会投"yes" 原因可以看2.x的第三个例子

   

   #### 2. log replication

   

   ##### 2.x一些可能发生的问题分析以及解决

   1）simplest example

   ```
   s1: 3
   s2: 3	3	
   s3:	3	3	
   ```

   假设s3为leader, 为什么此时Log不统一？

   当s3为term3的leader时，request连续来了两个，每次s3先写入自己的日志，然后要求followers复制它，s1丢了一个(网络分区)

   2）leader crash

   ```
   	10	11	12			10	11	12			10	11	12
   s1:	3				s1:	3				s1:	3
   s2:	3	3			s2:	3	3	4		s2:	3	3	4
   s3:	3	3			s3:	3	3			s3:	3	3	5
   ```

   **问题复现**

   s1 一直因为网络分区, 丢失（11,3）

   s2当选, term = 4, 收到一个请求, 还没来得及复制, crash

   s3选举, 且当选, term = 5, 收到request

   注意 （11,3）这个log应该是被commited的, 因为被majority所记录。

   3）leader crash and solution，leader的slot值高于其他的情况

   ```
   	10	11	12	13
   s1:	3
   s2:	3	3	4
   s3:	3	3	5	6
   ```

   假设此时s3是leader, 现在到了term6, s3将(13,6)写进log, 并试图让其他子节点进行apply 

   leader初始化nextIdx为自己收到最新request的槽位, leader的nextIdx[s2] = 13	nextIdx[s1] = 13

   1）leader试图让s2固话(13, 6)并向其发送previous log(12, 5), s2发现自己的最后一个log并非是 (12,5), 所以reject了这个请求, 同理 s1也拒接了这个请求

   2)   leader尝试降低nextIdx值去匹配, 如 nextIdx[s2] = 12 此时试图让s2固话(12, 5)(13, 6)并发送previous log(11, 53)，s2发现匹配上了, 接受该请求。并overwrite自己的Log 变为 3 3 5 6

   3）leader将继续降低nextIdx[s1]的值 重复上述步骤 直到s1接受请求

   **问题：**

   1)s2的4被移除，有关系吗？ 无 因为4并非是majority记录的数据 

   **2) 重点：假设s1为leader会怎么样？它没有足够的log,也不是majority?**

   解释：因为在election过程中, 只有 voter发现自己的log中最大的term小于等于candidate竞选发来的term时，才会投"yes", 所以 选出来的leadr一定是term比majority要大的, 是可以对majority进行覆盖的

   

   

   

   

   

   ##### 1.2数据结构

   **持久化的数据**, 保存进disk里，重启也能记忆

   1. currentTerm：server已知的最大任期, 通过心跳进行更新
   2. votedFor:  在leader election中投票给了谁
   3. log[]:  <operation, term, idx>

   **通用性易失数据**

   1.  

   

   



## 数据结构





## 细节部分

1. 由于多个workers要向coordinator请求, 所以coordinator的rpc handler里要加锁 处理concurrency问题





## Appendix

set up go

https://go.dev/doc/install?download=go1.14.4.linux-amd64.tar.gz 

linux脚本修复

sed -i 's/\r$//' test-mr.sh 

