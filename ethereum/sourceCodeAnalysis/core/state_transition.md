# state transition

本部分主要完成在执行交易后需要进行的各类state状态变化。[代码位置](https://github.com/ethereum/go-ethereum/blob/master/core/state_transition.go)
1. 处理nonce
2. 预支付eth支付gas
3. 交易eth
 * 创建合约交易
    * 尝试运行交易的data
    * 成功后，将结果作为新state对象的代码
5. 计算新的state root

## 代码详细解析

首先看数据结构的定义：

StateTransition的定义：
```go
type StateTransition struct {
    // 全局的gas数量，注意
	gp         *GasPool
	// 消息类型（由交易转换而来）
	msg        Message
	// 当前的gas数量
	gas        uint64
	// gas价格
	gasPrice   *big.Int
	// 初始提供的gas数量
	initialGas uint64
	// 交易的eth value
	value      *big.Int
	// 交易的data
	data       []byte
	// 执行时的stateDB
	state      vm.StateDB
	// evm执行环境
	evm        *vm.EVM
}
```

Message类型的定义：
```go
// Message代表的是发送给合约的消息
type Message interface {
    // 发送者
	From() common.Address
	// 发送给的地址
	To() *common.Address
    // gas价格
	GasPrice() *big.Int
	// gas数量
	Gas() uint64
	// eth数量
	Value() *big.Int
    // 消息nonce
	Nonce() uint64
	// 
	CheckNonce() bool
	// 交易data字段数据
	Data() []byte
}
```

StateTransition构建函数：
```go
func NewStateTransition(evm *vm.EVM, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:       gp,
		evm:      evm,
		msg:      msg,
		gasPrice: msg.GasPrice(),
		value:    msg.Value(),
		data:     msg.Data(),
		state:    evm.StateDB,
	}
}
```

执行交易：
```go
func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, uint64, bool, error) {
	return NewStateTransition(evm, msg, gp).TransitionDb()
}

// 具体执行交易的过程
func (st *StateTransition) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
	// 对交易进行预先检查
    if err = st.preCheck(); err != nil {
		return
	}
	msg := st.msg
	sender := vm.AccountRef(msg.From())
	homestead := st.evm.ChainConfig().IsHomestead(st.evm.BlockNumber)
	// 判断是否为合约创建交易
	contractCreation := msg.To() == nil

	// 支付intrinsic所需的gas
	gas, err := IntrinsicGas(st.data, contractCreation, homestead)
	if err != nil {
		return nil, 0, false, err
	}
	if err = st.useGas(gas); err != nil {
		return nil, 0, false, err
	}

	var (
		evm = st.evm
		// 除了"insufficient balance"错误，其他类型vmerr不会影响共识，所以不算是error
		vmerr error
	)
	if contractCreation {
	    // 执行创建合约的相关操作
		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// 增加nonce
		st.state.SetNonce(msg.From(), st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = evm.Call(sender, st.to(), st.data, st.gas, st.value)
	}
	if vmerr != nil {
		log.Debug("VM returned with error", "err", vmerr)
		// 唯一可能引起共识错误的evm错误.
		if vmerr == vm.ErrInsufficiezhugountBalance {
			return nil, 0, false, vmerr
		}
	}
	// 把未用完的gas对应的eth返还给caller，注意数量不是完全对应的
	st.refundGas()
	// 把手续费加到coinbase（矿工）地址上，注意该字段不一定和header.coinbase字段一致
	// evm.Coinbase是首先从共识的Author方法获取，所以在pow中，与header.coinbase字段不一致
	st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))

	return ret, st.gasUsed(), vmerr != nil, err
}
```

PreCheck，对交易的预先检查：
```go
func (st *StateTransition) buyGas() error {
    // 计算gas*gasPrice的eth数量
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
	// 发送交易者的金额不足则返回错误
	if st.state.GetBalance(st.msg.From()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	// 全局gas达到了限制，也返回错误。gp为当前正要打包的块中的gasLimit
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	// 初始化当前gas数量
	st.gas += st.msg.Gas()

    // 初始话总gas数量
	st.initialGas = st.msg.Gas()
	// 预先扣除相应的eth
	st.state.SubBalance(st.msg.From(), mgval)
	return nil
}

func (st *StateTransition) preCheck() error {
	// 检查nonce刚好为正确的nonce
	if st.msg.CheckNonce() {
		nonce := st.state.GetNonce(st.msg.From())
		if nonce < st.msg.Nonce() {
			return ErrNonceTooHigh
		} else if nonce > st.msg.Nonce() {
			return ErrNonceTooLow
		}
	}
	// 预支付eth购买自己提供的gasLimit
	return st.buyGas()
}
```

支付IntrinsicGas：
```go
func IntrinsicGas(data []byte, contractCreation, homestead bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	// 判断是否为创建合约交易，所需的gas不一致
	if contractCreation && homestead {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		// 计算非0的数据个数
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		if (math.MaxUint64-gas)/params.TxDataNonZeroGas < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * params.TxDataNonZeroGas

        // 计算0的数据个数
		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, vm.ErrOutOfGas
		}
		gas += z * params.TxDataZeroGas
	}
	// 总的来说 gas = 交易类型的gas + 0数据的gas×0数据的size + 非0数据的gas×非0数据的size
	return gas, nil
}
```

交易执行完成后，进行refund操作：
```go
func (st *StateTransition) refundGas() {
	// refund的数量为使用gas的一半
	refund := st.gasUsed() / 2
	// 取refund和GetRefund()中较小的一个
	// GetRefund()在evm执行过程中，仅有两个操作可能会增加该部分的值：
	// 1. SstoreRefundGas： 15000，执行Sstore操作，删除一个地址
	// 2. SuicideRefundGas： 合约自毁24000
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	// 当前剩余gas加上refund的gas
	st.gas += refund

	// 计算需要退回的金额
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	// 把这部分金额退还给发送交易的地址
	st.state.AddBalance(st.msg.From(), remaining)

	// 将剩下的gas加到全局的gas中，保证下一条交易使用的数据是对的。
	st.gp.AddGas(st.gas)
}
```

## state处理总流程

1. 构建对象，传递全局使用的gasLimit
2. 预先检查交易
    1. 检查交易nonce
    2. 预支付eth
3. 支付intrinsic gas
4. evm执行交易：
    * 创建合约交易：执行create（该操作会nonce+1）
    * 普通交易：执行call，并且nonce+1
5. 如果evm执行出错，并且错误是资金不足时，返回错误（交易执行失败，会回滚state，）
6. refund Gas：执行相应refund操作，将剩余资金（比实际使用的多一点）返回给交易发送者
