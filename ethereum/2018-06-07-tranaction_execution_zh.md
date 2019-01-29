---
layout: post
title: 以太坊交易执行流程
---

# 以太坊交易执行流程
以太坊交易主要分为两种：创建合约交易和普通交易（含合约调用），下面分别介绍这创建合约的执行流程。

# 创建合约
当交易中to的地址为空时，就默认为创建合约的交易；
1. 计算合约地址`crypto.CreateAddress(caller.Address(), nonce)`，传入参数为caller(msg.sender),nonce(msg.sender的nonce)
    1. `data, _ := rlp.EncodeToBytes([]interface{}{b, nonce});return common.BytesToAddress(Keccak256(data)[12:])`
    因为合约地址是通过创建者的地址和nonce生成的，所以可以预先生成出合约地址，但还未发布该合约，之后在发布合约；或
    预先向可能的合约地址发送eth等行为。
2. 向合约地址发送eth
3. 执行合约代码（此时的代码为创建合约时的代码，执行构造函数），返回执行结果（发布后的代码）
4. 将发布后的代码设置到合约地址上。
5. 合约创建完成。

通过上述流程，可以发现合约的构造函数只执行一次，且创建合约是的data代码和发布完成后的代码不是一致的。

## 以太坊合法地址判断
```go
// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Ethereum address or not.
func IsHexAddress(s string) bool {
	if hasHexPrefix(s) {
		s = s[2:]
	}
	return len(s) == 2*AddressLength && isHex(s)
}
```
一个hex字符串，如果长度为40（不含0x）（20个byte）就为一个合法的以太坊地址。
