# How To Send Ether To Other Address On Contract
Solidity中向某个地址发送eth或者调用其他合约接口的异同之处

# Transfer
这个是比较常用的转账方法，直接使用`address.transfer(value)`就可以向某个地址完成转账操作。在合约中调用该方法
时，会直接从合约中转出相应value数量的eth给address，注意该方法如果没有成功的话，会自动抛出异常，调用该方法的函数
会自动失败，状态会进行回滚. *transfer的gas数量只有2300，如果address是合约且需要进行复杂的操作则会失败*

# Send
*该方法目前已被弃用，现在基本使用transfer来代替*。使用`address.send(value)`向某个地址发送eth，它的gas数量也
为2300和transfer不同的是，它执行失败时，会返回false，但是不会抛出异常。类似于`require(address.send(amount))`

# Call
可以以`address.call.value(value).gas(amount)()`的形式来向某个地址发送eth，或调用合约的payable方法等，amount
可以指定gas的上限（默认为所有剩余的gas），value指定需要发送的ether的数量，当执行失败时，会返回false  
可以使用`address.call.gas().value()(bytes4(keccak256("someFun(uint256...)")), params)`，someFun(uint256...)为调用的
方法，如transfer(address,uint256)，param为相应的参数值;该方法与`SomeContract.someFund.gas().value()`效果一致。

# DelegateCall
如：
```solidity
contract Test {
    function delegateCallOther(address contractB) public {
        contractB.delegateCall(msg.data);
    }
}
```
该代码调用DelegateCall时，contractB中的数据不会有所变化，而会将相应的变化应用到Test合约中。msg.sender
,也还是当初调用Test的人，不会变为Test合约的地址。  
DelegateCall和callCode基本一致，但是callCode的msg.sender会变化（目前callCode基本使用较少，之后可以会移除）。
delegateCall主要用于library的形式。