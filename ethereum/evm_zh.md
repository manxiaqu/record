# EVM详解
主要介绍以太坊虚拟机(EVM)的特点、限制、执行机制和solidity之间的关系等。

### 基本特点

1. 准图灵完备语言，通过gas来限制计算量，每执行一步操作，均会消耗相应的gas(可能为0).
2. 栈中的数据均以32个字节形式进行存储，即2^256(go中使用big.Int进行存储)

### Opcodes
opcodes是以太坊中预先定义好的操作，智能合约编译的代码及相关调用最终都会转换为opcodes来执行，每个步骤的opcodes
根据实际情况会消耗一定数量的gas，在gas消耗完毕后，执行会终止，并且所有的修改会被回滚。
以下为opcodes操作表，参考[junm_table.go](https://github.com/ethereum/go-ethereum/blob/master/core/vm/jump_table.go)，
对照代码得出，*源代码未有类似表格直接列出*。
```go
// schema: [opcode, pop, push, gasUsed]

var opcodes = {
	    // arithmetic
        0x00: ["STOP", 0, 0, 0],
        0x01: ["ADD", 2, 1, 3],
        0x02: ["MUL", 2, 1, 5],
        0x03: ["SUB", 2, 1, 3],
        0x04: ["DIV", 2, 1, 5],
        0x05: ["SDIV", 2, 1, 5],
        0x06: ["MOD", 2, 1, 5],
        0x07: ["SMOD", 2, 1, 5],
        0x08: ["ADDMOD", 3, 1, 8],
        0x09: ["MULMOD", 3, 1, 8],
        0x0a: ["EXP", 2, 1, gasExp], // evm-tools中显示为10
        0x0b: ["SIGNEXTEND", 2, 1, 5],
    
        // boolean
        0x10: ["LT", 2, 1, 3],
        0x11: ["GT", 2, 1, 3],
        0x12: ["SLT", 2, 1, 3],
        0x13: ["SGT", 2, 1, 3],
        0x14: ["EQ", 2, 1, 3],
        0x15: ["ISZERO", 1, 1, 3],
        0x16: ["AND", 2, 1, 3],
        0x17: ["OR", 2, 1, 3],
        0x18: ["XOR", 2, 1, 3],
        0x19: ["NOT", 1, 1, 3],
        0x1a: ["BYTE", 2, 1, 3],
        0x1b: ["SHL", 2, 1, 3],
        0x1c: ["SHR", 2, 1, 3],
        0x1d: ["SAR", 2, 1, 3],
    
        // crypto
        0x20: ["SHA3", 2, 1, gasSha3], // evm-tools中显示为30
        
        // contract context
        0x30: ["ADDRESS", 0, 1, 2],
        0x31: ["BALANCE", 1, 1, 400], // Homestead为20；EIP150/EIP158为400；
        0x32: ["ORIGIN", 0, 1, 2],
        0x33: ["CALLER", 0, 1, 2],
        0x34: ["CALLVALUE", 0, 1, 2],
        0x35: ["CALLDATALOAD", 1, 1, 3],
        0x36: ["CALLDATASIZE", 0, 1, 2],
        0x37: ["CALLDATACOPY", 3, 0, gasCallDataCopy], // evm-tools中显示为3
        0x38: ["CODESIZE", 0, 1, 2],
        0x39: ["CODECOPY", 3, 0, gasCodeCopy], // evm-tool中显示为3
        0x3a: ["GASPRICE", 0, 1, 2],
        0x3b: ["EXTCODESIZE", 1, 1, 700], // Homestead为20；EIP150/EIP158为700；
        0x3c: ["EXTCODECOPY", 4, 0, gasExtCodeCopy], // evm-tools中显示为20
        0x3d: ["RETURNDATASIZE", 0, 1, 2],
        0x3f: ["RETURNDATACOPY", 3, 0, gasReturnDataCopy],
    
        // blockchain context
        0x40: ["BLOCKHASH", 1, 1, 20],
        0x41: ["COINBASE", 0, 1, 2],
        0x42: ["TIMESTAMP", 0, 1, 2],
        0x43: ["NUMBER", 0, 1, 2],
        0x44: ["DIFFICULTY", 0, 1, 2],
        0x45: ["GASLIMIT", 0, 1, 2],
      
        // storage and execution
        0x50: ["POP", 1, 0, 2],
        0x51: ["MLOAD", 1, 1, gasMLoad], // evm-tools中显示为3
        0x52: ["MSTORE", 2, 0, gasMStore], // evm-tools中显示为3
        0x53: ["MSTORE8", 2, 0, gasMStore8], // evm-tools中显示为3
        0x54: ["SLOAD", 1, 1, 200], // Homestead为50；EIP150/EIP158为200；
        0x55: ["SSTORE", 2, 0, gasSStore], // evm-tools中显示为0
        0x56: ["JUMP", 1, 0, 8],
        0x57: ["JUMPI", 2, 0, 10],
        0x58: ["PC", 0, 1, 2],
        0x59: ["MSIZE", 0, 1, 2],
        0x5a: ["GAS", 0, 1, 2],
        0x5b: ["JUMPDEST", 0, 0, 1],
    
        // logging
        0xa0: ["LOG0", 2, 0, makeGasLog(0)], // evm-tools中显示为375
        0xa1: ["LOG1", 3, 0, makeGasLog(1)], // evm-tools中显示为750
        0xa2: ["LOG2", 4, 0, makeGasLog(2)], // evm-tools中显示为1125
        0xa3: ["LOG3", 5, 0, makeGasLog(3)], // evm-tools中显示为1500
        0xa4: ["LOG4", 6, 0, makeGasLog(4)], // evm-tools中显示为1875
        
        // unofficial opcodes used for parsing
        0xb0: ["PUSH"],
        0xb1: ["DUP"],
        0xb2: ["SWAP"],
        
        // closures
        0xf0: ["CREATE", 3, 1, gasCreate32000],
        0xf1: ["CALL", 7, 1, gasCall40],
        0xf2: ["CALLCODE", 7, 1, gasCallCode40],
        0xf3: ["RETURN", 2, 0, gasReturn0],
        0xf4: ["DELEGATECALL", 6, 1, gasDelegateCall],
        
        0xfa: ["STATICCALL", 6, 1, gasStaticCall],
        0xfd: ["REVERT", 2, 0, gasRevert],
        0xff: ["SELFDESTRUCT", 1, 0, gasSuicide], 
    	
        // arbitrary length storage (proposal for metropolis hardfork)
        0xe1: ["SLOADBYTES", 3, 0, 50],
        0xe2: ["SSTOREBYTES", 3, 0, 0],
        0xe3: ["SSIZE", 1, 1, 50],
}

// i 代表是一个字节个数，如PUSH1代表压入1个单字节的数，PUSH2代表压入一个双字节的数，下面同此。
for i := 1; i <= 32; i++ {
    opcodes[0x60 + i - 1] = ["PUSH" + string(i), 0, 1, 3];
}

for i := 1; i <= 16; i++ {
    opcodes[0x80 + i - 1] = ["DUP" + string(i), i, i+1, 3]
    opcodes[0x90 + i - 1] = ["SWP" + string(i), i+1, i+1, 3]
}
```

上面的表格中列出了当前所有的opcode，并列出了操作需要从栈中取出/放入多少个参数和相应需要消耗的gas数量。

### EVM限制
EVM对执行的代码和使用的栈进行了一些限制，包括：
1. 栈最大的深度为1024
2. call调用深度最高为1024
3. 如果evm执行过程中发生错误，则会回滚数据库的所有操作，如果错误不是revert，则会消耗完所有的gas。
4. 合约代码大小最大为24576（在EIP158之后）
5. EVM执行过程中，共有3中存储方式，memory([]byte，为字节数组,通过offset+size方式存取)，storage(梅克尔树)，
stack([]*big.Int，int类型数组)

### 执行过程
下面我们将通过一些实际的例子来介绍evm的相关执行过程，如果想亲自动手实践的话，请务必保证好已经编译好evm命令行工具。

这里有`evm --code 60ff60ff --debug`非常简单的命名（可在go-ethereum readme.md中找到），但经实际实践，该命令无法运行
。但以该代码为例。  
通过`echo "60ff60ff" >> codefile`将代码保存为文件，后运行`./evm disasm codefile`输出反汇编的结果
```bash
60ff60ff
000000: PUSH1 0xff
000002: PUSH1 0xff
```
可以看出该段代码共执行了两部操作，均为将0xff压入栈。
下面分析`60ff60ff`具体内容：  
首先`60ff60ff`为十六进制字符串即`0x60ff60ff`，`0x60`根据上面的opcodes表，可以看出其为PUSH1(1个字节)操作，后面的0xff即为对应的参数。  

下面使用`./evm --debug --code 6005600401 run`进行调试：
```bash
0x
#### TRACE ####
PUSH1           pc=00000000 gas=10000000000 cost=3

PUSH1           pc=00000002 gas=9999999997 cost=3
Stack:
00000000  00000000000000000000000000000000000000000000000000000000000000ff

STOP            pc=00000004 gas=9999999994 cost=0
Stack:
00000000  00000000000000000000000000000000000000000000000000000000000000ff
00000001  00000000000000000000000000000000000000000000000000000000000000ff

#### LOGS ####
```
通过结果可以看出，上面两步操作分别消耗了3gas，并且栈中已经保存了相应的数据。

### 预编译合约
预编译合约是硬编码在客户端中的合约，列表如下：
```go
// PrecompiledContractsByzantium contains the default set of pre-compiled Ethereum
// contracts used in the Byzantium release.
var PrecompiledContractsByzantium = map[common.Address]PrecompiledContract{
	common.BytesToAddress([]byte{1}): &ecrecover{},
	common.BytesToAddress([]byte{2}): &sha256hash{},
	common.BytesToAddress([]byte{3}): &ripemd160hash{},
	common.BytesToAddress([]byte{4}): &dataCopy{},
	common.BytesToAddress([]byte{5}): &bigModExp{},
	common.BytesToAddress([]byte{6}): &bn256Add{},
	common.BytesToAddress([]byte{7}): &bn256ScalarMul{},
	common.BytesToAddress([]byte{8}): &bn256Pairing{},
}
```

### 合约代码执行
这里对合约的创建、调用相应接口的具体流程进行具体的分析，找出交易/调用是如何调用合约接口、代码具体执行
流程。
// TODO 添加对具体的合约生成的代码进行分析，梳理出调用合约接口的具体交易流程和call的流程。


### 参考
1. [以太坊黄皮书](https://github.com/ethereum/yellowpaper)
2. [evm-tools](https://github.com/CoinCulture/evm-tools/blob/master/analysis/guide.md)