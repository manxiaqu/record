# Hard Fork
以太坊Hard Fork事件记录

### 以太坊Hard Fork

#### Homestead
##### 启用块高度
1. 主网： 1,150,000
2. modern： 494,000

##### 版本更新内容
1. [EIP2](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md),增加通过交易创建合约的gas消耗；
任何交易签名中的s值大于secp256k1n/2被认为是不合法的；创建合约时，gas不够的话，交易会失败，而不是留下一个空合约；
修改difficultly调整方式
2. [EIP7](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-7.md),添加一个新的opcode
3. [EIP8](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-8.md),devp2p Wire Protocol的实现

#### DAO Fork
##### 启用块高度
1. 主网高度：1,920,000

##### 更新内容
1. DAO fork，将原DAO中丢失的资金找回：将原DAO中所有用户的资金转移到一个特殊的合约中。

#### Tangerine Whistle
##### 启用块高度
1. 主网： 2,463,000

##### 版本更新内容
1. [EIP150](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-150.md), evm的opcodes从state
 tree中读取数据和其他的opcodes相关联（即读操作的opcodes可能受到其他opcodes的影响），解决方法是增加了读取操作的
 gas花费
 
#### spurious dragon
##### 启用块高度
1. 主网：2,675,000块高度
2. morden：1,885,000块高度
  
##### 版本更新的内容
1. [EIP155](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md)预防replay攻击：防止在以太坊某条链（测试链or私链）已经成功执行的交易重新在主链或其他
链上进行广播、使用；如，你在morden测试链发送了100个eth给某人，在主链上不能重复执行该条交易。
中提出了，对交易进行签名时，签名的参数中需要包含chainId，这样该条交易就无法在
其他链上重复使用。（如果没有对chainid进行签名，任何人可以获取你发送交易的rawTransaction
并将它在其他链上进行重复广播）
2. [EIP160](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-160.md)增加exp操作消耗的gas
3. [EIP161](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-161.md)在早期，因dos攻击产生了许多的
空账号，现在可以以很小的代价把他们从链上state中移除，从而极大的减少链的大小。
4. [EIP170](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-170.md)，限制智能合约代码的大小，
目前合约代码最大设置为24576个字节。如果创建的合约大小超过这个，则会弹出out of gas错误。

#### Byzantium
##### 块启用高度
1. 主网： 4,370,000
2. Ropsten： 1,700,000

##### 版本更新内容
1. [EIP100](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-100.md)修改difficultly调整策略
2. [EIP140](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-140.md)在evm中添加REVERT指令，
可以在不需要消耗完所有gas的情况下，停止交易的执行，并且返回异常原因
3. [EIP196](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-196.md)添加零知识证明需要的预编译合约
4. [EIP197](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-197.md)添加能验证零知识证明的预编译合约
5. [EIP198](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-198.md)BigInt模幂运算
6. [EIP211](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-211.md)添加新的opcode
7. [EIP214](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-214.md)添加新的opcode
8. [EIP649](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-649.md)减少块奖励
9. [EIP658](https://github.com/ethereum/EIPs/blob/master/EIPS/eip-658.md)在交易receipts中添加状态标识字段
