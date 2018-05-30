# Privacy
合约的隐私性
```solidity
pragma solidity ^0.4.18;

contract Privacy {

  bool public locked = true;
  uint256 public constant ID = block.timestamp;
  uint8 private flattening = 10;
  uint8 private denomination = 255;
  uint16 private awkwardness = uint16(now);
  bytes32[3] private data;

  function Privacy(bytes32[3] _data) public {
    data = _data;
  }
  
  function unlock(bytes16 _key) public {
    require(_key == bytes16(data[2]));
    locked = false;
  }

  /*
    A bunch of super advanced solidity algorithms...

      ,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`
      .,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,
      *.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^         ,---/V\
      `*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.    ~|__(o.o)
      ^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'^`*.,*'  UU  UU
  */
}
```

### 代码分析
只要传入正确的key后，即可解锁账号；需要通过分析storage来获取data的具体数据。  
例如：  
instance：0x7ecbfb9ef1497373de4fc8ecb84801bea9f6e872
storage[0]:0x0000000000000000000000000000000000000000000000000000007d00ff0a00
storage[1]:0x8e7cd92580800357d3a6f7c90a899a674ba3301ec216e1ff0cf7146d11c3126a
storage[2]:0xb0843069456c2e4da3fbfed16c35ad1c4f5658e9da1d13018db91025716a75fe
storage[3]:0x3f725ed1e0c50cd58368edf185a9080c0745b62de7698668cfdd16f902b19be2
```solidity
bool public locked = true;
uint256 public constant ID = block.timestamp;
uint8 private flattening = 10;
uint8 private denomination = 255;
uint16 private awkwardness = uint16(now);
bytes32[3] private data;
```
1. ID为constant，存于栈上；
2. locked=true; 为
2. flattening=10; 为0x0a
3. denomination=255; 为0xff
0x
3f725ed1e0c50cd58368edf185a9080c
0745b62de7698668cfdd16f902b19be2
推出data[2]为0x3f725ed1e0c50cd58368edf185a9080c0745b62de7698668cfdd16f902b19be2，转换为
bytes16为0x4f5658e9da1d13018db91025716a75fe00000000000000000000000000000000；
所以传入的key应该为0x4f5658e9da1d13018db91025716a75fe00000000000000000000000000000000

### 攻击方法
1. 调用unlock(key)；unlock("0x4f5658e9da1d13018db91025716a75fe")
