# txpools/tx_list详解
对于以太坊的交易打包流程进行详细的介绍，包括API发送交易、交易排队、打包及相关的参数校验等

### 常见交易返回错误
后台调用API发送交易时，成功后则会返回交易hash(不代表交易一定成功)，失败会返回相应的错误信息:  
```go
var (
	// ErrInvalidSender is returned if the transaction contains an invalid signature.
	// 交易签名不合法
	ErrInvalidSender = errors.New("invalid sender")

	// ErrNonceTooLow is returned if the nonce of a transaction is lower than the
	// one present in the local chain.
	// 交易中nonce字段值低于链中该地址下nonce字段的值。
	ErrNonceTooLow = errors.New("nonce too low")

	// ErrUnderpriced is returned if a transaction's gas price is below the minimum
	// configured for the transaction pool.
	// Gasprice低于矿工最低可接受的价格
	ErrUnderpriced = errors.New("transaction underpriced")

	// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
	// with a different one without the required price bump.
	// 如果，当前有一条交易处于pending状态，后又发送了同样地址，同样nonce字段的交易，但是交易价格不符合替换
    	// 原交易的规则时，返回该错误
	ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")

	// ErrInsufficientFunds is returned if the total cost of executing a transaction
	// is higher than the balance of the user's account.
	// 当前账号余额不足以支付gasPrice * gas + value
	ErrInsufficientFunds = errors.New("insufficient funds for gas * price + value")

	// ErrIntrinsicGas is returned if the transaction is specified to use less gas
	// than required to start the invocation.
	// gas的数量低于调用需求的数量
	ErrIntrinsicGas = errors.New("intrinsic gas too low")

	// ErrGasLimit is returned if a transaction's requested gas limit exceeds the
	// maximum allowance of the current block.
	// Gas数量超过了块的gas数量上限
	ErrGasLimit = errors.New("exceeds block gas limit")

	// ErrNegativeValue is a sanity error to ensure noone is able to specify a
	// transaction with a negative value.
	// value值小于0
	ErrNegativeValue = errors.New("negative value")

	// ErrOversizedData is returned if the input data of a transaction is greater
	// than some meaningful limit a user might use. This is not a consensus error
	// making the transaction invalid, rather a DOS protection.
	// 交易中inputData字段的大小超过了限制
	ErrOversizedData = errors.New("oversized data")
)
```
上述的ErrNonceTooLow、ErrInsufficientFunds等均是根据当前节点的本地数据进行计算的，即如果接收该
交易的节点(rpc调用的节点)并没有同步到最新状态，可能会判断错误或发出不会成功的交易。

###　 txpool

txpool中含以下交易列表：  
1. pending txlist：当前正在处理的交易池，`map[address]tx`;
2. queue txlist：正在排队的交易池（没有进行处理），`map[address]tx`;
3. txPricedList：当前所有的交易列表，按价格进行排序
4. all: 当前所有交易列表，`map[hash]tx`;   

当以太坊节点从本地或通过广播的方式收到了一条交易时，在完成合法性检验后，它会进入3交易列表和1、2列表的其中一个。
1条新的交易加入池的过程如下：  
1. 对交易进行合法性校验，其中gasLimit，gasPrice与矿工的设置有关，代码如下：
```go
func (pool *TxPool) validateTx(tx *types.Transaction, local bool) error {
	// Heuristic limit, reject transactions over 32KB to prevent DOS attacks
	if tx.Size() > 32*1024 {
		return ErrOversizedData
	}
	// Transactions can't be negative. This may never happen using RLP decoded
	// transactions but may occur if you create a transaction using the RPC.
	if tx.Value().Sign() < 0 {
		return ErrNegativeValue
	}
	// Ensure the transaction doesn't exceed the current block limit gas.
	if pool.currentMaxGas < tx.Gas() {
		return ErrGasLimit
	}
	// Make sure the transaction is signed properly
	from, err := types.Sender(pool.signer, tx)
	if err != nil {
		return ErrInvalidSender
	}
	// Drop non-local transactions under our own minimal accepted gas price
	local = local || pool.locals.contains(from) // account may be local even if the transaction arrived from the network
	if !local && pool.gasPrice.Cmp(tx.GasPrice()) > 0 {
		return ErrUnderpriced
	}
	// Ensure the transaction adheres to nonce ordering
	if pool.currentState.GetNonce(from) > tx.Nonce() {
		return ErrNonceTooLow
	}
	// Transactor should have enough funds to cover the costs
	// cost == V + GP * GL
	if pool.currentState.GetBalance(from).Cmp(tx.Cost()) < 0 {
		return ErrInsufficientFunds
	}
	intrGas, err := IntrinsicGas(tx.Data(), tx.To() == nil, pool.homestead)
	if err != nil {
		return err
	}
	if tx.Gas() < intrGas {
		return ErrIntrinsicGas
	}
	return nil
}
```
2. 如果当前池已经满了，把价格最低的那些交易从池子中去掉(不会去掉本地账号(私钥在该节点上的账号)发送的交易)。
3. 如果该条交易是一条替换交易的话，检查提供的gasPrice是否满足替换的条件
(newGasPrice > oldGasPrice * (100 + priceBump) / 100)，priceBump默认为10(即gasPrice比原价格高10%)
，满足则替换，不满足则返回错误。
4. 如果是一条新的交易，则加入queue列表，如果是本地账户发送的交易还会加入单独的本地交易列表。

*以太坊中的交易是按nonce顺序执行的，如当前A的nonce为4，后收到了A的nonce为6的交易后，该交易只会在queque
队列中，而不会处理，需要等到节点收到nonce为5的交易后，按顺序处理相关的交易*
