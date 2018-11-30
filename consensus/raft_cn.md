# raft

raft是管理复制日志的一致性共识算法，它最终的结果与Paxos的结果一致，但是更加高效、易懂和易于实现。

新颖的功能：
* leader功能更强： raft中的leader的功能比其他共识算法相比更加强大，例如：日志只会从leader流向其
他server。
* 选举leader：使用随机的timer选举leader，减少了选举过程的复杂度和冲突的可能性。
* 成员变更：使用join consensus来完成成员的变更，允许系统在成员变更期间对外提供服务。

# raft角色/行为简述

raft中的角色主要分为leader、candidate、follower三类。

## 术语解释

* leader: raft中的核心，接收客户端请求，将自己的日志同步给其他server等。
* candidate: 只在选举leader的期间才存在的角色，有可能被选举为leader。
* follower: 跟随leader，同步leader发送的日志，相应leader和candidate的各类请求；在收到用户请求时，
只会返回leader的相关信息，而不会自己处理该请求，需要客户端重新向leader请求。
* term: 与pbft中的view类似，可以理解为一个周期，在每个term中最多只有一个leader，每当leader更换时，
term需要自增。
* log entries: 同步日志的具体内容，包含客户端请求的数据和共识配置更新内容等。
* index: 该日志的索引序号(唯一)。

## State

各类角色中对应state的存储及处理。

1. 所有server上的persistent state：
(在响应rpc之前，需要更新其storage的内容)
* currentTerm: server当前最新的term(从0开始且单调递增)
* votedFor: 在当前term中，该server已经投选的候选者id。
* log[]: 从leader获取的日志信息，每一个log entry包含了发送给状态机的命令和收到entry的term。

2. 所有server上的volatile state:
* commitIndex: 当前已知的已提交日志的最高的index(从0开始，单调递增)
* lastApplied: 当前已知的已经应用到状态机日志的最高index(从0开始，单调递增；lastApplied <= commitIndex)

3. leader上的volatile state:
(在选举过后会重新初始化)
* nextIndex[]: 要发给server(follower)下一个日志的index。
* matchIndex[]: server(follower)从leader同步且备份了的日志的最高的index。

## AppendEntries RPC(添加entry的rpc)

leader向其他follower发出复制/备份日志的rpc请求，当日志的内容为空时用于心跳以保持连接。

### 变量

* term： leader的当前term编号，严格递增。
* leaderID： leader的id，便于follower对客户端进行重定向。
* prevLogIndex： 最新日志前一个日志的index。
* prevLogTerm： prevLogIndex的term。
* entries[]： 需要存储的日志(如果日志为空，则用于心跳)。
* leaderCommit： leader已经提交的日志的index

### 结果

* term： follower当前的term，leader可以据此更新自己。
* success： 如果follower的含有可以和prevLogIndex、prevLogTerm匹配的entry，则返回true。

### 接收端需实现内容

1. 如果term小于follower的currentTerm，返回false
2. 如果follower本地的日志不包含和prevLogIndex、prevLogTerm匹配的entry，返回false
3. 如果follower现有的entry和leader发送的日志冲突（同样index，但不同的term），则以leader发送的日志为准，且删除该不符的
日志及其之后的所有的日志。(即leader的日志永远是合法的，其他和leader中日志不同的日志均需要被删除)
4. 将未有的日志添加到日志中。
5. 如果leaderCommit > commitIndex，将commitIndex设置为min(leaderCommit, index of last new entry)

## RequestVote RPC（收集投票）

候选者发送请求给其他server用于收集投票成为leader。

### 变量

* term： 候选者的term。
* candidateId： 发送请求的候选者id。
* lastLogIndex： 候选者最新日志的index。
* lastLogTerm： 候选者最新日志的term。

### 结果

* term： 当前term，用于候选者更新自己的信息。
* voteGranted： true表明follower保证将它的票投给该candidate。

### 接收端需实现内容

1. 如果candidate的term小于server自己本地的currentTerm，返回false。
2. 如果server中的votedFor为空或者是候选者的id，candidate的日志新于server本地的日志或一致，则投票
给candidate。

## Rules for Servers（server的规则）

1. 所有server需遵守的规则：
* 若commitIndex > lastApplied：增长lastApplied，并将log[lastApplied]日志的内容应用到状态机。
* 如果在rpc的请求和回复中出现term T > currentTerm，将currentTerm设置为T，将自己的身份转换
为follower

2. Followers需遵守的规则：
* 回复leader和候选者的rpc请求
* 在选举过程中，选举超时且没有收到当前leader的AppendEntries请求或没有给其他候选者投票时，将自己
的角色转为候选者

3. 候选者需遵守的规则：
* 开始选举时：
    * 增加currentTerm值
    * 给自己投票
    * 重置选举时间
    * 向其他所有server发送RequestVote RPC请求
* 如果收到了大多数server的票(>1/2)：成为leader
* 如果收到了新leader的AppendEntries请求：成为follower
* 如果timeout：开始新一轮的选举。(在选举失败时，可能会出现term中没有leader的情况)

4. leader需遵守：
* 选举完成后：发送空的AppendEntries RPCs(心跳)给所有的server，重复该动作防止选举timeout。
* 如果收到了client的请求：将entry添加至本地的log，在状态机完成应用后，发送结果给客户端。
* 如果follower的lastLogIndex >= nextIndex：将从nextIndex开始的日志以AppendEntries RPC请求的
方式发送给follower。
    * 成功：为follower更新nextIndex和matchIndex。
    * 如果因为日志不同步失败：减少nextIndex并重试。(重试一直进行)
* 如果存在N > commitIndex且matchIndex[i]中的大部分>=N，且log[N].term == currentTerm，将commitIndex
设置为N

# leader选举

在刚开始启动时，所有server的角色均为follower，并且在收到leader和candidate的rpc请求后或timeout之前
维持其角色不变；在timeout后(每个follower的timeout时间是随机的)，它认为当前term没有leader，并增加
自己当前的term，同时启动leader选举流程。启动leader流程时需要进行的操作：
1. 增加currentTerm，并写入candidate state。
2. 给自己进行投票。
3. 向其他所有server发送rpc请求收集票选。


在以下三种情况发生后，candidate会结束其角色状态，根据票选转为follower或leader：
1. 自己赢得选举。
2. 其他candidate赢得选举。
3. 没有人赢得选举。

赢得选举的条件：任意candidate收到大多数的票(>1/2)，每个server最多给1个candidate进行投票(含自己)，
follower会给最先给其他送请求的合法的candidate投票。只有1个candidate收到>1/2的票后才能成为leader，
保证最多只有1个server可以赢取选举。

在candidate等待票选的时候，可能会收到其他server的appendEntries请求(成为leader后)，如果leader的
term大于等于candidate的term，则candidate将自己的角色变为follower并将其视为leader；否则，拒绝请求
保持candidate的角色不变。

任何人都没有赢的选举的情况：
(所有的server都几乎在同一时间成为candidate，该情况下，票将会被分散造成没有人可以获得>1/2的票)：
该种情况下，raft会使用随机数来选择timeout确保票不会被分散，从而快速选出leader；为了防止在第一次
选举的时候没办法正常选举出leader，server会在150-300ms中随机选取等待时间。(方法同重新开始选举一样)

# 日志复制/一致性

日志复制流程：
1. 选举leader
2. leader被选举出来后，它开始接收客户端的请求，并将其请求内容添加到日志中，同时向其他follower发送
AppendEntries RPC请求，当leader收到了>1/2个正确的回复后(含自己)，leader将日志commit至其state。
3. 若因网络或follower节点宕机等原因，leader无法收到足够多的回复时，它会重复发送rpc请求，直到收到
足够多的回复为止。

所有的日志在提交时，会同时将日志所在的term和index也提交用于冲突的检测。

# 原文连接

* [In Search of an Understandable Consensus Algorithm](https://ramcloud.stanford.edu/raft.pdf)
* [etcd-raft](https://github.com/etcd-io/etcd/tree/master/raft)