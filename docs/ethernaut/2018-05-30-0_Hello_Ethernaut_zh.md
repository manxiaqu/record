---
layout: post
title: Hello Ethernaut
tags: [ethernaut]
---

# Hello Ethernaut
Ethernaut的欢迎合约。
```solidity
pragma solidity ^0.4.18;

contract Instance {

  string public password;
  uint8 public infoNum = 42;
  string public theMethodName = 'The method name is method7123949.';
  bool private cleared = false;

  // constructor
  function Instance(string _password) public {
    password = _password;
  }

  function info() public pure returns (string) {
    return 'You will find what you need in info1().';
  }

  function info1() public pure returns (string) {
    return 'Try info2(), but with "hello" as a parameter.';
  }

  function info2(string param) public pure returns (string) {
    if(keccak256(param) == keccak256('hello')) {
      return 'The property infoNum holds the number of the next info method to call.';
    }
    return 'Wrong parameter.';
  }

  function info42() public pure returns (string) {
    return 'theMethodName is the name of the next method.';
  }

  function method7123949() public pure returns (string) {
    return 'If you know the password, submit it to authenticate().';
  }

  function authenticate(string passkey) public {
    if(keccak256(passkey) == keccak256(password)) {
      cleared = true;
    }
  }

  function getCleared() public view returns (bool) {
    return cleared;
  }
}
```
均为简单的合约方法调用，在此不进行相关介绍，初学者可参考[solidity官方文档](http://solidity.readthedocs.io/en/v0.4.24/index.html)