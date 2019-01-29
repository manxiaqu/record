---
layout: post
title: Fallback
tags: [ethernaut]
---

# Fallback
关于Fallback函数的使用。
```solidity
pragma solidity ^0.4.18;

import 'zeppelin-solidity/contracts/ownership/Ownable.sol';

contract Fallback is Ownable {

  mapping(address => uint) public contributions;

  function Fallback() public {
    contributions[msg.sender] = 1000 * (1 ether);
  }

  function contribute() public payable {
    require(msg.value < 0.001 ether);
    contributions[msg.sender] += msg.value;
    if(contributions[msg.sender] > contributions[owner]) {
      owner = msg.sender;
    }
  }

  function getContribution() public view returns (uint) {
    return contributions[msg.sender];
  }

  function withdraw() public onlyOwner {
    owner.transfer(this.balance);
  }

  function() payable public {
    require(msg.value > 0 && contributions[msg.sender] > 0);
    owner = msg.sender;
  }
}
```

### 代码分析
```solidity
function() payable public {
    require(msg.value > 0 && contributions[msg.sender] > 0);
    owner = msg.sender;
  }
```
上述方法为该合约的fallback函数，payable代表其可以接收eth。`require(msg.value > 0 && contributions[msg.sender] > 0);`
说明需要向该合约发送大于0的eth数量，并且`contributions[msg.sender] > 0`即可获得owner管理权限。

### 攻击方法
1. 调用contribute()方法，并发送小于0.001的eth
2. 向合约发送>0的eth
3. 调用withdraw()，该方法可选