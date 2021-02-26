# Submarine Sends/潜艇交易

潜艇交易是解决frontrunning问题的一种解决方案。

## frontrunning问题

用户广播到以太坊的交易需要旷工打包后才能真正的成为区块链的一部分，这个过程一般需要
几分钟的时间，在此期间，其他人员(不仅仅是旷工)可以提前通过计算等方法知道该笔交易的
执行结果从而获取收益。

## frontrunning示例

Bancor合约：用户可以在Bancor合约中买入和卖出token，合约会自动根据买入和卖出的token
量来自动计算当前的token价格；当用户卖出token时，价格会降低；当用户买入时，价格会升高。

具体流程：
1. 用户通过运行全节点或者api等方式获取到正在被打包的买入或者卖出订单。
2. 当有大额买入订单时，自己发送买入交易并将gas设置的比获取的交易大一点，这样大概率能在
大额买入订单前被打包，从而获取收益； 有大额卖出订单时，则进行相反的操作。

## 潜艇交易流程

* A：用户
* B：潜艇地址
* C：Libsubmarine库地址

交易打包顺序：
1. Commit (A --> B) // B地址与c无任何关联，通过特定方式生成，所以应该会被正常打包
2. Reveal (A --> C)
3. Unlock (B --> C)

交易生成顺序：
1. Unlock (B --> C)
2. Commit (A --> B)
3. Reveal (A --> C)

### 生成TxUnlock

终端用户生成随机数w，并计算：
commit(submarineId) =  Keccak256(addr(End User) | addr(DApp Contract/LibSubmarine) | value | optionalDAppData | w | gasPrice | gasLimit)
随后生成txunlock，调用unlock(submarineId):
```
to: C
value: $value
nonce: 0
data: commit
gasPrice: $gp
gasLimit: $gl
r: Keccak256(commit | 1)
s: Keccak256(commit | 0)
v: 27 // This makes TxUnlock replayable across chains ¯\_(ツ)_/¯
```
A之后计算ECRECOVER(TxUnlock)，重复这一过程直到成功获取到commit address B。
**B地址没有私钥，但是ECRECOVER是可以正常获取到publickey从而获取到address B的**
签名的信息是包含在r，s，v中的，这里通过填写随机的r和s，来计算出一个随机的没有私钥的公钥。(该过程所需时间？)

### Commit

A向B转账value，广播该笔正常交易。value可能需要比实际发送给dapp的金额要高一点，因为B转账
时需要支付手续费。

### Reveal

A调用dapp的reveal( _commitTxBlockNumber, _embeddedDAppData, _witness, _rlpUnlockTxUnsigned, _proofBlob)方法:
* _commitTxBlockNumber: txcommit被打包的块
* _embeddedDAppData：提交给dapp的数据
* _witness：生成过程中使用的随机数
* _rlpUnlockTxUnsigned：rlp编码后的txUnlock的详细数据
* _proofBlob： Merkle Proof证明committx被打包(是否还需要验证交易的状态，还是默克尔证明可以证明交易的状态)

A重新计算submarineId，验证Merkle proof， 并可以调用dapp的onSubmarineReveal执行dapp的一些业务逻辑操作。
// 在reveal通过之后，任何人都可以通过合约查询到该笔unlock的具体参数和内容，所以任何人都可以这笔交易，但是因为这笔交易的所有参数都已经是固定的了，不会有任何影响。

### Unlock

A(或其他人)广播TxUnlock交易至C(该笔交易是生成阶段生成的unlocktx，无需签名)
// C中对unlock方法不进行任何验证(否则unlock在reveal被打包失败后，B地址的nonce增加，从而eth会全部丢失)：
```
function unlock(bytes32 _submarineId) public payable {
        // Required to prevent an attack where someone would unlock after an
        // unlock had already happened, and try to overwrite the unlock amount.
        require(
            sessions[_submarineId].amountUnlocked < msg.value,
            "You can never unlock less money than you've already unlocked."
        );
        sessions[_submarineId].amountUnlocked = uint96(msg.value);
        emit Unlocked(_submarineId, uint96(msg.value));
    }
```

### 之后

dapp可以通过revealedAndUnlocked等方法查询交易状态，然后执行dapp其他相应的操作。

