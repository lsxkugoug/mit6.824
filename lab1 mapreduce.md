## lab1 mapreduce

**word count 实现**

#### map部分：

在map部分中，map任务的数量等于input文件分割的数量。

**coordinator 部分：**

告诉workers处理的文件，在实际工业环境中，分割的input文件将分散在GFS中。并通过rpc向workers传递命令, 告知workers需要使用的map函数, 这个map函数编译完成之后应该存储在GFS里。 

**workers部分：**

workers主动向coordinate 申请任务

workers会使用map函数将file里的数据转换成 {key, v}键值对, 如 "hello hello word" -> {hello, 1} {word, 1} {hello, 1}.

之后，再对其按照key sort，转换成 {hello, 1} {hello, 1} {world, 1}， 原因是在reduce的时候, 可以简单的通过双指针算法统计相同key的值

之后workers将这些键值对保存在存在GFS里, 这些文件被称为 intermediate

**难理解的部分**

但是这里有一个问题，intermediate文件要怎么保存？

假设在GFS中，一个大文件被拆分成m个小文件，我们reduce的最终文件数量是 r 个。那么intermediate的数量就应该是 m * r个

现在设 mr-x-y x为第一阶段任务名，y为第二阶段任务名

{hello, 1} 应该存放在 y = hash(hello) % r, 假设这个键值对是由第一个小文件得出的, 那 x = 1

为什么要这样做？

因为按照论文，经历过reduce之后, 最终我们将得到 r 个最终文件。r个文件将由coordinator进行分配, 也就是 r 个 reduce任务。每个worker领取一个任务, 然后从 mr-x-y 中读取，如 一个worker领了 1 号任务, 就得从 mr-\*-1多个文件中读取所有 {word, 1}键值对。因为之前map 的时候，所有相同的键值对会分到同一个*

mr-*-y里去，这将会导致每一个worker都能简单的操作所有相同key的键值对(因为在一个文件夹里)。 最后，每一个worker执行完reduce之后，将文件保存为

mr-y. 此时应该会有r个最终文件, 对应每一个reduce 任务。

**协议部分：**

coordinator：

coordinator向workers分派任务：

1. "noMapTask"表示所有task都已经派送完，但是有的没有完成，如果没有完成的任务超时，将会给请求map的任务派送
2. "mapTaskCompleted" 表示所有map task均已完成，workers不需要去请求map任务了 可以进入下一个reduce阶段了
3. 正常派送map任务，发送filename给workers 让workers处理

workers：

​	1. 如果没有接收到 "mapTaskCompleted"，就一直请求map任务（workers向coordinator申请map任务：不需要发送任何东西，只需要使用 rpc 调用 coordinator 的 map任务handler）

#### **reduce部分**

reduce任务的数量等于n

**coordinator 部分：**

当待统计的文件被全部map完毕之后，也就是所有的参与的workers 的报告已经将文件分割部分全部完成之后，系统进入reduce部分

现在有的文件是一堆intermediate, 里面存放了{hello, 1} {hello, 1} {world, 1}

coordinator对map任务进行记录 等待workers申请

**workers部分：**

workers主动向coordinate 申请任务

处理存在于coordinator里的intermediate文件。

原本的intermediate文件为{hello, 1} {hello, 1} {world, 1} 处理之后应该为：{hello, 2} {world, 1} 

将final文件存放在GFS中



mapreduce过程结束



对于通信, 使用RPC

对于容错,在超过设定值 n 秒之后，若部分workers还未回复coordinator, coordinator将会把该任务派给其他的workers。





## 数据结构





## 细节部分

1. 由于多个workers要向coordinator请求, 所以coordinator的rpc handler里要加锁 处理concurrency问题





## Appendix

set up go

https://go.dev/doc/install?download=go1.14.4.linux-amd64.tar.gz 

linux脚本修复

sed -i 's/\r$//' test-mr.sh 

