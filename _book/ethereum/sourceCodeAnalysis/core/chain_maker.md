# chain maker

该代码主要功能是生成链用于对分叉规则等进行测试，但需要注意的是使用代码生成的块不一定会符合实际的
pow或者poa等共识规则，该部分只是对块头部的必要字段进行了填充，连接了首尾块等，并未按共识规则对
内容进行处理。

# 代码详细分析

1. BlockGen对象，该对象提供了一些方法可以填充一些特定的内容到需要生成的块中，提供的方法有：
    1. SetCoinbase：设置coinbase字段，该方法在每个块设定时仅能调用一次
    2. SetExtra：设置extra字段
    3. AddTx：添加交易至块
    4. AddTxWithChain： 将交易添加至生成的块中，在无法执行时会panic
    5. Number： 返回块号
    6. AddUncheckedReceipt： 添加收据至块中，该收据对应的交易不需要包含在块中
    7. TxNonce：返回账号的nonce，当前数据中没有该账号信息时会panic
    8. AddUncle： 添加uncle块
    9. PrevBlock：返回上一个块
    10. OffsetTime：修改块时间
```go
type BlockGen struct {
	i           int
	parent      *types.Block
	chain       []*types.Block
	chainReader consensus.ChainReader
	header      *types.Header
	statedb     *state.StateDB

	gasPool  *GasPool
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header

	config *params.ChainConfig
	engine consensus.Engine
}
```

2. GenerateChain生成块的主要方法

```go
// 调用该方法时，需要传入
// config： 当前链的配置信息
// parent： 父块，在该块的基础之上构建新块
// engine： 使用的共识引擎
// db： 数据库db
// n： 生成的数量
// gen： 自定以的方法，可以对块的内容进行自定义的修改
func GenerateChain(config *params.ChainConfig, parent *types.Block, engine consensus.Engine, db ethdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	if config == nil {
		config = params.TestChainConfig
	}
	// 首先生成n个空块和收集
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	// 生成块的主体方法，该方法会返回对应的块及收据
	genblock := func(i int, parent *types.Block, statedb *state.StateDB) (*types.Block, types.Receipts) {
		// TODO(karalabe): This is needed for clique, which depends on multiple blocks.
		// It's nonetheless ugly to spin up a blockchain here. Get rid of this somehow.
		// 常见blockchain
		blockchain, _ := NewBlockChain(db, nil, config, engine, vm.Config{})
		defer blockchain.Stop()

        // 构建BlockGen对象
		b := &BlockGen{i: i, parent: parent, chain: blocks, chainReader: blockchain, statedb: statedb, config: config, engine: engine}
		// 生成块头部
		b.header = makeHeader(b.chainReader, parent, statedb, b.engine)

		// Mutate the state and block according to any hard-fork specs
		// 应用dao block的相关内容
		if daoBlock := config.DAOForkBlock; daoBlock != nil {
			limit := new(big.Int).Add(daoBlock, params.DAOForkExtraRange)
			if b.header.Number.Cmp(daoBlock) >= 0 && b.header.Number.Cmp(limit) < 0 {
				if config.DAOForkSupport {
					b.header.Extra = common.CopyBytes(params.DAOForkBlockExtra)
				}
			}
		}
		if config.DAOForkSupport && config.DAOForkBlock != nil && config.DAOForkBlock.Cmp(b.header.Number) == 0 {
			misc.ApplyDAOHardFork(statedb)
		}
		// Execute any user modifications to the block and finalize it
		// 执行用户自定以的内容修改块和块头部，该方法由用户调用该方法时传入
		if gen != nil {
			gen(i, b)
		}

        // 应用共识引擎的Finalize方法对state进行必要的修改
		if b.engine != nil {
			block, _ := b.engine.Finalize(b.chainReader, b.header, statedb, b.txs, b.uncles, b.receipts)
			// Write state changes to db
			root, err := statedb.Commit(config.IsEIP158(b.header.Number))
			if err != nil {
				panic(fmt.Sprintf("state write error: %v", err))
			}
			if err := statedb.Database().TrieDB().Commit(root, false); err != nil {
				panic(fmt.Sprintf("trie write error: %v", err))
			}
			return block, b.receipts
		}
		return nil, nil
	}
	// 实际生成块的循环
	for i := 0; i < n; i++ {
	    // 依据父块root 构建statedb
		statedb, err := state.New(parent.Root(), state.NewDatabase(db))
		if err != nil {
			panic(err)
		}
		// 生成块和交易收据
		block, receipt := genblock(i, parent, statedb)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}
```

从以上部分看，该生成方法仅调用了共识部分的Finalize()方法，但实际上共识部分共有prepare(), Finalize(), seal()
等方法，其中prepare及finalize主要设置共识字段及state，该方法未调用prepare，所以生成的块不一定符合具体的共识
规则。