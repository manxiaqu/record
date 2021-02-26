# 反事实合约实例化/Counterfactual Instantiation

当前create：在一个合约发布前，可以通过合约发布者地址和合约发布者nonce计算出合约的地址。
`Keccak256(sender_addr, sender_nonce)`

Create2：允许用户和当前链上不存在的地址进行交互。
`sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]`

允许用户与还不存在但是可以通过特定代码创建的地址进行交互。对于涉及与合同进行反事实交互的状态通道很重要。
**如果生成的地址nonce或者code不为空，则抛出异常**
**如果合约是在交易中创建的，即该交易不是创建合约的交易，但是执行过程中创建了合约，则
被创建的合约的nonce从1开始**

// solidity 0.6.2中可以使用
EIP1014(create2): 

# Zero Confirmation Transactions/零确认交易

1.

# SMS-based Payments/基于短信的付款

