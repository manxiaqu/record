# block validator

该部分主要为块验证器，主要功能是对块和state进行验证。

## 代码详细分析

首先BlockValidator数据结构：
```go
// BlockValidator的主要工作主要是对块头部，uncle和state进行验证
// 主要在收到新块的时候调用
type BlockValidator struct {
	config *params.ChainConfig // 链的配置参数，创世块中指定
	bc     *BlockChain         // 当前合法的链
	engine consensus.Engine    // 当前链使用的共识引擎
}
```

验证块具体内容：
```go
// 对块的uncle、交易、uncle root进行验证，在调用该方法时，假设已经对块头部进行过验证了
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// 验证当前的块是否已知
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	// 验证父块是否已知
	if !v.bc.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		if !v.bc.HasBlock(block.ParentHash(), block.NumberU64()-1) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	// 在该位置，header已经验证过了，需要验证交易及uncle
	header := block.Header()
	// 调用共识引擎的方法验证uncle
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	// 重新计算uncle的hash，是否与header中一致
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	// 重新计算交易root，是否与header中一致
	if hash := types.DeriveSha(block.Transactions()); hash != header.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, header.TxHash)
	}
	return nil
}
```

对state进行验证：
```go
// 
func (v *BlockValidator) ValidateState(block, parent *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	header := block.Header()
	// 检查使用的gas是否已知
	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// 重新通过receipt计算bloom，是否与header中的一致
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// 重新计算receipt的root hash，是否与header中一致
	receiptSha := types.DeriveSha(receipts)
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// 重新计算state root，判断其是否和header中的state root中一致
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}
```

gasLimit计算方法：
```go
func CalcGasLimit(parent *types.Block) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
	    策略：gasLimit是基于父块的gasUsed进行计算得到的。if parentGasUsed > parentGasLimit * (2/3)
	    ，那么就增加，否则减少；增加和减少的数据根据gasUsed和parentGasLimit * (2/3)之间相差的多少来计算
	*/
	// 不能低于最小的gasLimit5000
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// 如果矿工设置了TargetGasLimit参数，那么就增加直到达到了该目标值
	if limit < params.TargetGasLimit {
		limit = parent.GasLimit() + decay
		if limit > params.TargetGasLimit {
			limit = params.TargetGasLimit
		}
	}
	return limit
}
```

