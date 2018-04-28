# 以太坊交易、块头部等数据结构分析

### 交易

#### 发送RawTransaction
数据字段：    
* nonce: 当前地址交易计数，从零开始，且只包含发送出去的交易  
* gasPrice: gas价格，以wei为单位// 实际消耗的费用=gasPrice * gasUsed  
* gasLimit: 该笔交易消耗gas上限  
* from： 发出交易的地址 // web3j的rawTransaction没有此字段  
* to: 将交易发给谁 //空则是创建合约；合约地址则是调用合约接口；普通地址则是以太坊交易转账  
* value: 转账金额，以wei为单位  
* data: 传递数据，可为空；// 调用合约及创建合约时，均将相关数据放在此位置  

交易发编码/签名/发送:  
1. 将上述数据字段转为合并为rlp列表，其中data先转为hex字符串。  
2. 使用rlp编码方法，将其编码为字节。  
3. 使用秘钥对上述字节进行签名  
4. 将签名结果中的V、R、S也加入rlp列表中  
5. 将rlp列表编码为字节  
6. 将上述字节转换为字符串  
7. 使用sendRawTransaction()发送交易  

*下面中提到的数据类型均为abi中的数据类型*
调用合约时，data字段编码详解：  
1. 第一部分(调用方法id) :   
    1. 将方法名及需传递的参数类型连接成字符串
    2. 使用sha3对其进行编码.
    3. 转换为hex字符串，返回前十个组成的字符串  
2. 随后部分(调用方法参数)：  
    1. 计算动态数据偏量位置，*参数为静态数组则会计算其数组长度*，后乘以32。
    2. 对参数进行编码，判断数据类型
         * 动态数据类型： 将动态偏量编码后字符串append在data中，
         缓存其在动态数据中，动态数据偏量加上该动态数据的个数后(如[1,2]则加2)右移一位  
         * 静态数据类型： 直接使用abi对其进行编码，并依次添加  
         
例：
调用```batchTransfer(address[] _receivers, uint256 _value)```，data的编码过程及结果如下：  
传递参数为：
address[] : ["0xb4d30cac5124b46c2df0cf3e3e1be05f42119033", "0x0e823ffe018727585eaf5bc769fa80472f76c3d7"]
_value :  2^255
1. sha3("batchTransfer(address[],uint256)")，
2. 动态数据偏量为2 * 32;(address[]不是static array)
3. address[]为动态数据类型，将64转换为hex，添加至data
4. 缓存address[]变量的内容
5. _value编码为string，添加至data
6. 添加缓存内容

对应data结果为：
1. "0x83f12fec"  
2. "0x83f12fec"  
3. "0x83f12fec0000000000000000000000000000000000000000000000000000000000000040"  
4. "0x83f12fec0000000000000000000000000000000000000000000000000000000000000040"  
5. "0x83f12fec00000000000000000000000000000000000000000000000000000000000000408000000000000000000000000000000000000000000000000000000000000000"
6. "0x83f12fec00000000000000000000000000000000000000000000000000000000000000408000000000000000000000000000000000000000000000000000000000000000
0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000b4d30cac5124b46c2df0cf3e3e1be05f42119033
0000000000000000000000000e823ffe018727585eaf5bc769fa80472f76c3d7"

可以使用我编写的[小工具 in progress](./convert_tool_test.go)对结果进行测试，暂不支持数组类型。

#### Transaction (get from eth.getTransaction())
数据字段：
* blockHash: 包含该块的哈希值
* blockNumber: 包含该条交易的块号
* from: 发出该条交易的地址
* gas: 消耗的gas
* gasPrice: gas价格，单位为wei
* hash: 该条交易的hash
* input: 该条交易的输入，即rawTransaction中data字段的内容
* nonce: 交易nonce，同rawTransaction中nonce
* r: 交易签名信息R
* s: 交易签名信息S
* to: 交易接收方的地址
* transactionIndex: 该条交易在块中的位置
* v: 交易签名信息V
* value: 转账的以太坊个数，单位为wei


#### TransactionReceipt (get from eth.getTransactionReceipt())
数据字段：
* blockHash: 包含该交易的哈希值
* blockNumber: 包含该交易的块号
* contractAddress:
* cumulativeGasUsed: 估算需消耗的gas数量
* from: 发送交易地址
* gasUsed: 实际使用gas数量
* logs: 日志数组
* logsBoom:
* status: 交易状态 // 0x0失败，0x1成功
* to: 接收方地址
* transactionHash: 交易哈希
* transactionIndex: 交易在块中位置


##### Log (data in transactionReceipt())
数据字段：
* address: 生成该事件的合约地址
* blockHash: 派生字段，包含该条交易的块哈希
* blockNumber: 包含该条交易的块号
* data: 事件中，没有进行indexed的部分
* logIndex: 暂不清楚
* removed: 是否被移除，true表示因为链
* topics: 为一个数组，其中第一个为sha3(调用的事件名及参数合并后的字符串)，随后为indexed的数据部分
* transactionHash: 交易哈希
* transactionIndex: 交易在块中位置

如event Transfer(address indexed from, address indexed to, uint256 value)  
传入参数分别为：  
* from:0xa2678c61e7e5a25ecb975acae14962fd47490bd4  
* to:0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be  
* value:1e21

log中对应数据分别为：
```js
data:"00000000000000000000000000000000000000000000003635c9adc5dea00000" //1e21 十六进制表示形式
topics:[
  "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef", //sha3("Transfer(address,address,uint256)")；sha3是Keccak256方法
  "000000000000000000000000a2678c61e7e5a25ecb975acae14962fd47490bd4",
  "0000000000000000000000003f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be"
]
```

[交易例子查询](https://etherscan.io/tx/0x9115e37f2f3d6ba8487d426c25cc9e84dcae2e518c1e6ef40dfd1ead5857ae65#eventlog)


#### Block (get from eth.getBlock())
数据字段：
* difficulty: 当前块的难度
* extraData: 额外数据部分，最多32字节
* gasLimit: gas上限
* gasUsed: 消耗的gas数量
* hash: 块哈希
* logsBoom:
* miner: 矿工地址
* mixHash: 混杂hash，和nonce共同运算得出，用于证明经过了一定量的计算
* nonce: 生成该块的nonce值，证明经过了一定量的计算
* number: 块号
* parentHash: 父块的哈希值
* receiptsRoot: receipte的根节点hash
* sha3Uncles: 暂不清楚
* size: 块大小
* stateRoot: state的根节点hash
* timestamp: 时间戳
* totalDifficulty: 总难度
* transactions: 该块所包含的交易，为交易的哈希数组
* uncles: uncles的hash数组




