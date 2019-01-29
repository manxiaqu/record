---
layout: post
title: 以太坊安全事件
---

# 以太坊智能合约安全事件记录

### BeautyChain 代币合约漏洞
发生时间：2018年4月22日  
造成后果：市值大降，基本清零  
应急措施：紧急关闭所有代币交易  
解决方法：重新发布代币合约，回滚ok交易所交易数据，并将原合约中的代币数量1:1转移至新合约  
[代币合约地址](https://etherscan.io/address/0xc5d105e63711398af9bbff092d4b6769c82f793d#code)：0xc5d105e63711398af9bbff092d4b6769c82f793d  
漏洞类型：无符号整数溢出  
[攻击交易ID](https://etherscan.io/tx/0xad89ff16fd1ebe3a0a7cf4ed282302c06626c1af33221ebe0d3a470aba4a660f)：0xad89ff16fd1ebe3a0a7cf4ed282302c06626c1af33221ebe0d3a470aba4a660f  

攻击调用源码函数：
```solidity
function batchTransfer(address[] _receivers, uint256 _value) public whenNotPaused returns (bool) {
    uint cnt = _receivers.length;
    uint256 amount = uint256(cnt) * _value;
    require(cnt > 0 && cnt <= 20);
    require(_value > 0 && balances[msg.sender] >= amount);

    balances[msg.sender] = balances[msg.sender].sub(amount);
    for (uint i = 0; i < cnt; i++) {
        balances[_receivers[i]] = balances[_receivers[i]].add(_value);
        Transfer(msg.sender, _receivers[i], _value);
    }
    return true;
  }
```
攻击者传入参数值（hex）（根据交易详情还原得出）：
address[] : ["000000000000000000000000b4d30cac5124b46c2df0cf3e3e1be05f42119033","0000000000000000000000000e823ffe018727585eaf5bc769fa80472f76c3d7"]
_value : "8000000000000000000000000000000000000000000000000000000000000000"

注意```uint256 amount = uint256(cnt) * _value;```amount计算为2^255*2，截断后为0；所以
```require(_value > 0 && balances[msg.sender] >= amount);```判断没有起到作用。


### SmartMesh 代币合约漏洞
发生时间：2018年4月25日  
造成后果：市值有所下降  
紧急措施：暂停所有交易  
解决方法：重新部署以太坊合约  
[代币合约地址](https://etherscan.io/address/0x55f93985431fc9304077687a35a1ba103dc1e081#code):0x55f93985431fc9304077687a35a1ba103dc1e081  
漏洞类型：无符号整数溢出  
[攻击交易ID](https://etherscan.io/tx/0x1abab4c8db9a30e703114528e31dee129a3a758f7f8abc3b6494aad3d304e43f):0x1abab4c8db9a30e703114528e31dee129a3a758f7f8abc3b6494aad3d304e43f  

攻击调用函数源码：
```solidity
function transferProxy(address _from, address _to, uint256 _value, uint256 _feeSmt,
        uint8 _v,bytes32 _r, bytes32 _s) public transferAllowed(_from) returns (bool){

        if(balances[_from] < _feeSmt + _value) revert();

        uint256 nonce = nonces[_from];
        bytes32 h = keccak256(_from,_to,_value,_feeSmt,nonce);
        if(_from != ecrecover(h,_v,_r,_s)) revert();

        if(balances[_to] + _value < balances[_to]
        || balances[msg.sender] + _feeSmt < balances[msg.sender]) revert();
        balances[_to] += _value;
        Transfer(_from, _to, _value);

        balances[msg.sender] += _feeSmt;
        Transfer(_from, msg.sender, _feeSmt);

        balances[_from] -= _value + _feeSmt;
        nonces[_from] = nonce + 1;
        return true;
    }
```
攻击者传入参数（均为hex值）：  
_from : 000000000000000000000000df31a499a5a8358b74564f1e2214b31bb34eb46f  
_to : 000000000000000000000000df31a499a5a8358b74564f1e2214b31bb34eb46f  
_value : 8fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff  
_feeSmt : 7000000000000000000000000000000000000000000000000000000000000001  
_v : 000000000000000000000000000000000000000000000000000000000000001b  
_r : 87790587c256045860b8fe624e5807a658424fad18c2348460e40ecf10fc8799  
_s : 6c879b1e8a0a62f23b47aa57a3369d416dd783966bd1dda0394c04163a98d8d8  

注意```if(balances[_from] < _feeSmt + _value) revert();```中未对_feeSmt+_value结果进行溢出检查，攻击者传入两值之和刚好为
2^256，截断后刚好为0，故条语句判断没有起到应有的作用，从而从没有币的账户中凭空"转"了天价的币给用户
