# blockchain

该部分为十分核心的内容，包含的链分叉的规则，插入新的块等各项操作。

## 代码详细分析

首先看到类型/参数定义：
```go
const (
	bodyCacheLimit      = 256   // 存储块body内容的cache上限
	blockCacheLimit     = 256   // 存储块的cache上限
	maxFutureBlocks     = 256   // 
	maxTimeFutureBlocks = 30    //
	badBlockLimit       = 10    //
	triesInMemory       = 128   //

	// BlockChainVersion可以强制不兼容的数据库从头开始重新同步。
	BlockChainVersion = 3
)

// CacheConfig包含了链中trie（梅克尔树）的缓存/pruning 的相关配置
type CacheConfig struct {
	Disabled      bool          // 是否关闭trie写缓存
	TrieNodeLimit int           // 当内存容量（MB）达到了多少后将缓存写入硬盘
	TrieTimeLimit time.Duration // 当时间达到了多少后，将缓存写入硬盘
}

// BlockChain代表的是从给定的创世块开始合法的链，主要包含链导入，修复，分叉等功能。
//
// 在将块导入链时，根据设置的validator规则进行相应的操作。主要由Processor完成对块和交易的处理。
// 第二部分的Validator完成对state的验证。验证及处理过程中出错会终止块的导入。

// BlockChain会保存所有的合法块数据内容，包含分叉的块。使用GetBlock可以返回分叉的块（不在当前最长链上）；
// 使用GetBlockByNumber则总会返回当前最长链上的块。
type BlockChain struct {
	chainConfig *params.ChainConfig // 链及网络配置（创世块）
	cacheConfig *CacheConfig        // 缓存配置

	db     ethdb.Database // 存储数据的底层数据库
	triegc *prque.Prque   // Priority queue mapping block numbers to tries to gc
	gcproc time.Duration  // Accumulates canonical block processing for trie dumping

	hc            *HeaderChain  // 头链
	rmLogsFeed    event.Feed    // 
	chainFeed     event.Feed    //
	chainSideFeed event.Feed    //
	chainHeadFeed event.Feed    //
	logsFeed      event.Feed    //
	scope         event.SubscriptionScope
	genesisBlock  *types.Block  // 创世块

	mu      sync.RWMutex // 链操作的全局锁
	chainmu sync.RWMutex // 插入链的锁
	procmu  sync.RWMutex // 处理块的锁

	checkpoint       int          // checkpoint counts towards the new checkpoint
	currentBlock     atomic.Value // 当前链的头部（原子操作）
	currentFastBlock atomic.Value // 当前快同步的头部（高度可能比当前链头部更高）

	stateCache   state.Database // state数据库
	bodyCache    *lru.Cache     // 最近块body的缓存
	bodyRLPCache *lru.Cache     // 最近块body rlp编码的缓存
	blockCache   *lru.Cache     // 最近块的缓存
	futureBlocks *lru.Cache     // 将来需要处理的块的缓存

	quit    chan struct{} // blockchain quit channel
	running int32         // running must be called atomically
	// procInterrupt must be atomically called
	procInterrupt int32          // interrupt signaler for block processing
	wg            sync.WaitGroup // chain processing wait group for shutting down

	engine    consensus.Engine  // 当前链使用的共识引擎
	processor Processor // 块处理对象
	validator Validator // 块验证对象
	vmConfig  vm.Config // 虚拟机配置

	badBlocks *lru.Cache // 坏块的缓存
}
```

构建新的BLockChain对象：
```go
// 
func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, engine consensus.Engine, vmConfig vm.Config) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = &CacheConfig{
		    // 
			TrieNodeLimit: 256 * 1024 * 1024,
			// 5分钟刷新缓存，将数据写入db
			TrieTimeLimit: 5 * time.Minute,
		}
	}
	// 初始快缓存对象
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)
	badBlocks, _ := lru.New(badBlockLimit)

    // 创建BlockChain对象
	bc := &BlockChain{
		chainConfig:  chainConfig,
		cacheConfig:  cacheConfig,
		db:           db,
		triegc:       prque.New(),
		stateCache:   state.NewDatabase(db),
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		engine:       engine,
		vmConfig:     vmConfig,
		badBlocks:    badBlocks,
	}
	// 设置块验证对象和state处理对象
	bc.SetValidator(NewBlockValidator(chainConfig, bc, engine))
	bc.SetProcessor(NewStateProcessor(chainConfig, bc, engine))

    // 创建头链
	var err error
	bc.hc, err = NewHeaderChain(db, chainConfig, engine, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	// 获取创世块
	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}
	// 导入最新的state
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	// 确保链中没有包含badHash的块
	for hash := range BadHashes {
	    // 尝试从当前链及分叉中获取坏块的header
		if header := bc.GetHeaderByHash(hash); header != nil {
			// 从当前链中获取坏块对应的number的块（一致说明当前坏块在最长链上，不一致说明处于分叉上）
			headerByNumber := bc.GetHeaderByNumber(header.Number.Uint64())
			// 如果坏块处在当前链上，将头部设置在前父块上
			if headerByNumber != nil && headerByNumber.Hash() == header.Hash() {
				log.Error("Found bad hash, rewinding chain", "number", header.Number, "hash", header.ParentHash)
				bc.SetHead(header.Number.Uint64() - 1)
				log.Error("Chain rewind was successful, resuming normal operation")
			}
		}
	}
	// 开始处理未来块的线程，每5s处理一次，收到quit信号后停止
	go bc.update()
	return bc, nil
}
```

导入最新的state：
```go
func (bc *BlockChain) loadLastState() error {
	// 获取当前头部块
	head := rawdb.ReadHeadBlockHash(bc.db)
	if head == (common.Hash{}) {
		// 从头开始初始化数据库
		log.Warn("Empty database, resetting chain")
		return bc.Reset()
	}
	// 确保头部块在当前链上可以获取
	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		// 丢失了头部块，重新设置链
		log.Warn("Head block missing, resetting chain", "hash", head)
		return bc.Reset()
	}
	// 确保可以得到跟块关联的state
	if _, err := state.New(currentBlock.Root(), bc.stateCache); err != nil {
		// 头部块的state丢失，尝试修复链
		log.Warn("Head state missing, repairing chain", "number", currentBlock.Number(), "hash", currentBlock.Hash())
		if err := bc.repair(&currentBlock); err != nil {
			return err
		}
	}
	// 设置当前的块
	bc.currentBlock.Store(currentBlock)

	// 设置当前块头部
	currentHeader := currentBlock.Header()
	if head := rawdb.ReadHeadHeaderHash(bc.db); head != (common.Hash{}) {
		if header := bc.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	bc.hc.SetCurrentHeader(currentHeader)

	// 导入快同步的头部块
	bc.currentFastBlock.Store(currentBlock)
	if head := rawdb.ReadHeadFastBlockHash(bc.db); head != (common.Hash{}) {
		if block := bc.GetBlockByHash(head); block != nil {
			bc.currentFastBlock.Store(block)
		}
	}

	currentFastBlock := bc.CurrentFastBlock()

    // 获取对应块的总难度
	headerTd := bc.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64())
	blockTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	fastTd := bc.GetTd(currentFastBlock.Hash(), currentFastBlock.NumberU64())

	log.Info("Loaded most recent local header", "number", currentHeader.Number, "hash", currentHeader.Hash(), "td", headerTd)
	log.Info("Loaded most recent local full block", "number", currentBlock.Number(), "hash", currentBlock.Hash(), "td", blockTd)
	log.Info("Loaded most recent local fast block", "number", currentFastBlock.Number(), "hash", currentFastBlock.Hash(), "td", fastTd)

	return nil
}
```

重新设置块头部：
```go
// SetHead将当前块头部设置为一个新的值。任何高度大于该块的块均会被删除。
func (bc *BlockChain) SetHead(head uint64) error {
	log.Warn("Rewinding blockchain", "target", head)

    // 开启链操作全局锁
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 设置头链的块为head，该该块之后的数据删除
	delFn := func(db rawdb.DatabaseDeleter, hash common.Hash, num uint64) {
		rawdb.DeleteBody(db, hash, num)
	}
	bc.hc.SetHead(head, delFn)
	currentHeader := bc.hc.CurrentHeader()

	// 清除缓存中的所有数据
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	//重新设置链, 确保当前的head block是有state的
	if currentBlock := bc.CurrentBlock(); currentBlock != nil && currentHeader.Number.Uint64() < currentBlock.NumberU64() {
		bc.currentBlock.Store(bc.GetBlock(currentHeader.Hash(), currentHeader.Number.Uint64()))
	}
	if currentBlock := bc.CurrentBlock(); currentBlock != nil {
		if _, err := state.New(currentBlock.Root(), bc.stateCache); err != nil {
			// 块state丢失, 回滚至创世块
			bc.currentBlock.Store(bc.genesisBlock)
		}
	}
	// 傻瓜式方法设置快同步的头部块
	if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock != nil && currentHeader.Number.Uint64() < currentFastBlock.NumberU64() {
		bc.currentFastBlock.Store(bc.GetBlock(currentHeader.Hash(), currentHeader.Number.Uint64()))
	}
	// 链头部块或快同步块为空，均回滚至创世块
	if currentBlock := bc.CurrentBlock(); currentBlock == nil {
		bc.currentBlock.Store(bc.genesisBlock)
	}
	if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock == nil {
		bc.currentFastBlock.Store(bc.genesisBlock)
	}
	currentBlock := bc.CurrentBlock()
	currentFastBlock := bc.CurrentFastBlock()

    // 重新记录当前块及快同步的块头部
	rawdb.WriteHeadBlockHash(bc.db, currentBlock.Hash())
	rawdb.WriteHeadFastBlockHash(bc.db, currentFastBlock.Hash())

	return bc.loadLastState()
}
```

从创世块开始重新设置链：
```go
// ResetWithGenesisBlock 删除整个链的内容，重新从创世块进行初始化
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// 将头部设置为创世块
	if err := bc.SetHead(0); err != nil {
		return err
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 从创世块开始初始化链
	if err := bc.hc.WriteTd(genesis.Hash(), genesis.NumberU64(), genesis.Difficulty()); err != nil {
		log.Crit("Failed to write genesis block TD", "err", err)
	}
	rawdb.WriteBlock(bc.db, genesis)

	bc.genesisBlock = genesis
	bc.insert(bc.genesisBlock)
	bc.currentBlock.Store(bc.genesisBlock)
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())
	bc.currentFastBlock.Store(bc.genesisBlock)

	return nil
}
```

尝试对链进行修复：
```go
// repair尝试修复当前链，将块进行回滚，直至块的state可以找到。
// 当前header和快同步的块不会发生变化
func (bc *BlockChain) repair(head **types.Block) error {
	for {
		// 可以找到对应块的state，则停止
		if _, err := state.New((*head).Root(), bc.stateCache); err == nil {
			log.Info("Rewound blockchain to past state", "number", (*head).Number(), "hash", (*head).Hash())
			return nil
		}
		// 回滚块
		(*head) = bc.GetBlock((*head).ParentHash(), (*head).NumberU64()-1)
	}
}
```

停止链服务：
```go
// Stop关闭blockchain服务，使用procInterrupt关闭正在处理的导入。
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	// 取消所有已经订阅的消息
	bc.scope.Close()
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()

	// 确保最近块的state已经写入到硬盘了
	if !bc.cacheConfig.Disabled {
		triedb := bc.stateCache.TrieDB()

		for _, offset := range []uint64{0, 1, triesInMemory - 1} {
			if number := bc.CurrentBlock().NumberU64(); number > offset {
				recent := bc.GetBlockByNumber(number - offset)

				log.Info("Writing cached state to disk", "block", recent.Number(), "hash", recent.Hash(), "root", recent.Root())
				if err := triedb.Commit(recent.Root(), true); err != nil {
					log.Error("Failed to commit recent state trie", "err", err)
				}
			}
		}
		for !bc.triegc.Empty() {
			triedb.Dereference(bc.triegc.PopItem().(common.Hash))
		}
		if size, _ := triedb.Size(); size != 0 {
			log.Error("Dangling trie nodes after full cleanup")
		}
	}
	log.Info("Blockchain manager stopped")
}
```

写入块和state
```go
func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts []*types.Receipt, state *state.StateDB) (status WriteStatus, err error) {
	bc.wg.Add(1)
	defer bc.wg.Done()

	// 获取新块的父块的总难度
	ptd := bc.GetTd(block.ParentHash(), block.NumberU64()-1)
	// 未获取到，说明该块的父块不在链中
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	// 开启全局锁
	bc.mu.Lock()
	defer bc.mu.Unlock()

	currentBlock := bc.CurrentBlock()
	// 当前链的总难度
	localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	// 插入新块后，新块所处链的总难度
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// 先不管state，将块本身写入数据库（不管是分叉还是最长链，都需要保存该块）
	if err := bc.hc.WriteTd(block.Hash(), block.NumberU64(), externTd); err != nil {
		return NonStatTy, err
	}
	batch := bc.db.NewBatch()
	rawdb.WriteBlock(batch, block)

	root, err := state.Commit(bc.chainConfig.IsEIP158(block.Number()))
	if err != nil {
		return NonStatTy, err
	}
	triedb := bc.stateCache.TrieDB()

	// 未开始缓存，则总是直接将相关内容写入到硬盘
	if bc.cacheConfig.Disabled {
		if err := triedb.Commit(root, false); err != nil {
			return NonStatTy, err
		}
	} else {
		// Full but not archive node, do proper garbage collection
		triedb.Reference(root, common.Hash{}) // metadata reference to keep trie alive
		bc.triegc.Push(root, -float32(block.NumberU64()))

		if current := block.NumberU64(); current > triesInMemory {
			// 如果达到了内存限制，则将内容写入硬盘
			var (
				nodes, imgs = triedb.Size()
				limit       = common.StorageSize(bc.cacheConfig.TrieNodeLimit) * 1024 * 1024
			)
			if nodes > limit || imgs > 4*1024*1024 {
				triedb.Cap(limit - ethdb.IdealBatchSize)
			}
			// Find the next state trie we need to commit
			header := bc.GetHeaderByNumber(current - triesInMemory)
			chosen := header.Number.Uint64()

			// 如果达到了时间限制，则将内容写入硬盘
			if bc.gcproc > bc.cacheConfig.TrieTimeLimit {
				if chosen < lastWrite+triesInMemory && bc.gcproc >= 2*bc.cacheConfig.TrieTimeLimit {
					log.Info("State in memory for too long, committing", "time", bc.gcproc, "allowance", bc.cacheConfig.TrieTimeLimit, "optimum", float64(chosen-lastWrite)/triesInMemory)
				}
				// Flush an entire trie and restart the counters
				triedb.Commit(header.Root, true)
				lastWrite = chosen
				bc.gcproc = 0
			}
			for !bc.triegc.Empty() {
				root, number := bc.triegc.Pop()
				if uint64(-number) > chosen {
					bc.triegc.Push(root, number)
					break
				}
				triedb.Dereference(root.(common.Hash))
			}
		}
	}
	rawdb.WriteReceipts(batch, block.Hash(), block.NumberU64(), receipts)

	// 如果新加入的块所在的链难度比当前链的难度大，则进行切换，切换到难度更大的那条链上
	reorg := externTd.Cmp(localTd) > 0
	currentBlock = bc.CurrentBlock()
	// 该部分主要为了防止自私挖矿
	if !reorg && externTd.Cmp(localTd) == 0 {
		// 如果难度一致，则首先切换到块高度较小的链
		// 如果难度和快高度均一致，则使用随机数进行切换；
		// 注意：不同客户端在执行mrand.Float64()得到的结果可能不一致，但是切换到某一块的几率均为50%，所以在经过
		// 短暂的分叉后，整个链还是会最终切换到最长的链上去
		reorg = block.NumberU64() < currentBlock.NumberU64() || (block.NumberU64() == currentBlock.NumberU64() && mrand.Float64() < 0.5)
	}
	if reorg {
		// 如果新块的父块不是当前链的头部块，进行链的切换操作
		if block.ParentHash() != currentBlock.Hash() {
			if err := bc.reorg(currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		rawdb.WriteTxLookupEntries(batch, block)
		rawdb.WritePreimages(batch, block.NumberU64(), state.Preimages())

		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	if err := batch.Write(); err != nil {
		return NonStatTy, err
	}

	// CanonStatTy则插入链（表明新块为当前链头部块的子块）
	if status == CanonStatTy {
		bc.insert(block)
	}
	bc.futureBlocks.Remove(block.Hash())
	return status, nil
}
```

将块插入链
```go
// insertChain执行实际的链插入操作，并产生相应的事件
func (bc *BlockChain) insertChain(chain types.Blocks) (int, []interface{}, []*types.Log, error) {
	if len(chain) == 0 {
		return 0, nil, nil, nil
	}
	// 检查当前链是正确的排列、连接的
	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			log.Error("Non contiguous block insert", "number", chain[i].Number(), "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash(), "prevnumber", chain[i-1].Number(), "prevhash", chain[i-1].Hash())

			return 0, nil, nil, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
		}
	}
	bc.wg.Add(1)
	defer bc.wg.Done()

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	// 传播事件的队列，一般比直接发送事件要快，且需更少的锁
	var (
		stats         = insertStats{startTime: mclock.Now()}
		events        = make([]interface{}, 0, len(chain))
		lastCanon     *types.Block
		coalescedLogs []*types.Log
	)
	// 使用共识引擎对header进行验证
	headers := make([]*types.Header, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		seals[i] = true
	}
	abort, results := bc.engine.VerifyHeaders(bc, headers, seals)
	defer close(abort)

	// 使用并发线程并行对块签名进行recover，提高处理效率
	senderCacher.recoverFromBlocks(types.MakeSigner(bc.chainConfig, chain[0].Number()), chain)

	// 遍历块，当通过验证后，尝试插入块
	for i, block := range chain {
		// 在链停止后，停止对块的处理
		if atomic.LoadInt32(&bc.procInterrupt) == 1 {
			log.Debug("Premature abort during blocks processing")
			break
		}
		// 如果header是坏块的hash，停止导入操作
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBlacklistedHash)
			return i, events, coalescedLogs, ErrBlacklistedHash
		}
		// 等待块验证的完成
		bstart := time.Now()

        // 获取头部验证的结果
		err := <-results
		if err == nil {
		    // 头部验证通过后，对body进行验证
			err = bc.Validator().ValidateBody(block)
		}
		switch {
		case err == ErrKnownBlock:
			// block和state都已知，但，如果当前的块高度低于该块高度，我们忽略它，在之后在重新导入
			if bc.CurrentBlock().NumberU64() >= block.NumberU64() {
				stats.ignored++
				continue
			}

		case err == consensus.ErrFutureBlock:
			// 这是未来的块，允许接收的块据现在的块头部时间差小于等于35s。即接收的块据当前块的时间戳差在35s，则
			// 可以接收该块，之后由线程对其进行处理
			max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
			if block.Time().Cmp(max) > 0 {
				return i, events, coalescedLogs, fmt.Errorf("future block: %v > %v", block.Time(), max)
			}
			bc.futureBlocks.Add(block.Hash(), block)
			stats.queued++
			continue

		case err == consensus.ErrUnknownAncestor && bc.futureBlocks.Contains(block.ParentHash()):
		    // 收到了future block中的子块
			bc.futureBlocks.Add(block.Hash(), block)
			stats.queued++
			continue

		case err == consensus.ErrPrunedAncestor:
			// 收到了不在当前链上的新块，把它存入db，但不处理state
			currentBlock := bc.CurrentBlock()
			// 当前总难度
			localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
			// 新块总难度
			externTd := new(big.Int).Add(bc.GetTd(block.ParentHash(), block.NumberU64()-1), block.Difficulty())
			// 如果当前难度>新块总难度，则只插入block
			if localTd.Cmp(externTd) > 0 {
				if err = bc.WriteBlockWithoutState(block, externTd); err != nil {
					return i, events, coalescedLogs, err
				}
				continue
			}
			// 在localTd==externTd，可能发送切换，在localTd<externTd时肯定发生切换
			var winner []*types.Block
            // 将新块所在的链的state导入
			parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
			for !bc.HasState(parent.Root()) {
				winner = append(winner, parent)
				parent = bc.GetBlock(parent.ParentHash(), parent.NumberU64()-1)
			}
			for j := 0; j < len(winner)/2; j++ {
				winner[j], winner[len(winner)-1-j] = winner[len(winner)-1-j], winner[j]
			}
			// 插入新链的块
			bc.chainmu.Unlock()
			_, evs, logs, err := bc.insertChain(winner)
			bc.chainmu.Lock()
			events, coalescedLogs = evs, logs

			if err != nil {
				return i, events, coalescedLogs, err
			}

		case err != nil:
			bc.reportBlock(block, nil, err)
			return i, events, coalescedLogs, err
		}
		// 获取父块
		var parent *types.Block
		if i == 0 {
			parent = bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
		} else {
			parent = chain[i-1]
		}
		// 使用父块的state root构建state
		state, err := state.New(parent.Root(), bc.stateCache)
		if err != nil {
			return i, events, coalescedLogs, err
		}
		// 基于父块state对块进行处理
		receipts, logs, usedGas, err := bc.processor.Process(block, state, bc.vmConfig)
		if err != nil {
			bc.reportBlock(block, receipts, err)
			return i, events, coalescedLogs, err
		}
		// 对state进行验证
		err = bc.Validator().ValidateState(block, parent, state, receipts, usedGas)
		if err != nil {
			bc.reportBlock(block, receipts, err)
			return i, events, coalescedLogs, err
		}
		proctime := time.Since(bstart)

		// 将块和state写入，返回状态
		status, err := bc.WriteBlockWithState(block, receipts, state)
		if err != nil {
			return i, events, coalescedLogs, err
		}
		switch status {
		// 插入了新块
		case CanonStatTy:
			log.Debug("Inserted new block", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", common.PrettyDuration(time.Since(bstart)))

			coalescedLogs = append(coalescedLogs, logs...)
			blockInsertTimer.UpdateSince(bstart)
			events = append(events, ChainEvent{block, block.Hash(), logs})
			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

        // 在分叉上插入了块
		case SideStatTy:
			log.Debug("Inserted forked block", "number", block.Number(), "hash", block.Hash(), "diff", block.Difficulty(), "elapsed",
				common.PrettyDuration(time.Since(bstart)), "txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()))

			blockInsertTimer.UpdateSince(bstart)
			events = append(events, ChainSideEvent{block})
		}
		stats.processed++
		stats.usedGas += usedGas

		cache, _ := bc.stateCache.TrieDB().Size()
		// 报告相应的处理情况
		stats.report(chain, i, cache)
	}
	// 如果将链处理完了，添加单独的chanHeadEvent事件
	if lastCanon != nil && bc.CurrentBlock().Hash() == lastCanon.Hash() {
		events = append(events, ChainHeadEvent{lastCanon})
	}
	return 0, events, coalescedLogs, nil
}
```

切换链：
```go
// 依据两个块进行链的切换，新链会重新导入blocks，并估计丢失的交易，并广播相应的事件
func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		commonBlock *types.Block
		deletedTxs  types.Transactions
		deletedLogs []*types.Log
		
		// collectLogs用于收集处理hash对应的块的日志
		collectLogs = func(hash common.Hash) {
			// Coalesce logs and set 'Removed'.
			number := bc.hc.GetBlockNumber(hash)
			if number == nil {
				return
			}
			receipts := rawdb.ReadReceipts(bc.db, hash, *number)
			for _, receipt := range receipts {
				for _, log := range receipt.Logs {
					del := *log
					del.Removed = true
					deletedLogs = append(deletedLogs, &del)
				}
			}
		}
	)

	// 先将高块减少到低块的高度
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = bc.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

			collectLogs(oldBlock.Hash())
		}
	} else {
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = bc.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

    // 找到同样的祖先
	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}

		oldChain = append(oldChain, oldBlock)
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectLogs(oldBlock.Hash())

		oldBlock, newBlock = bc.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1), bc.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1)
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}
	// 当reorg的大小比较大时对用户可见
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Debug
		if len(oldChain) > 63 {
			logFn = log.Warn
		}
		logFn("Chain split detected", "number", commonBlock.Number(), "hash", commonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
	} else {
		log.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Number(), "oldhash", oldBlock.Hash(), "newnum", newBlock.Number(), "newhash", newBlock.Hash())
	}
	// Insert the new chain, taking care of the proper incremental order
	// 插入新链
	var addedTxs types.Transactions
	for i := len(newChain) - 1; i >= 0; i-- {
		bc.insert(newChain[i])
		// 插入交易、收据相关lookupEntries
		rawdb.WriteTxLookupEntries(bc.db, newChain[i])
		addedTxs = append(addedTxs, newChain[i].Transactions()...)
	}
	// 比较添加的和删除的交易的异同
	diff := types.TxDifference(deletedTxs, addedTxs)
	// 删除被删除的交易的收据相关的lookupEntries
	batch := bc.db.NewBatch()
	for _, tx := range diff {
		rawdb.DeleteTxLookupEntry(batch, tx.Hash())
	}
	batch.Write()

    // 广播日志
	if len(deletedLogs) > 0 {
		go bc.rmLogsFeed.Send(RemovedLogsEvent{deletedLogs})
	}
	if len(oldChain) > 0 {
		go func() {
			for _, block := range oldChain {
				bc.chainSideFeed.Send(ChainSideEvent{Block: block})
			}
		}()
	}

	return nil
}
```

## 链插入、切换过程

1. 收到新块
2. 对块进行验证
    * 共识引擎规则验证
    * 块body验证
    * 块state验证
    * uncle验证
    * ...
3. 如果新块通过了认证则将其插入链中
4. 当前链的难度 <= 新块所在链的难度
    * 新块的父块即为当前链的头部块，不进行切换操作
    * 不一致且新块链所在难度大于当前链难度进行分叉操作
    * 当前链难度==新链难度
        1. 新块的块高度更新，进行切换操作
        2. 新块和当前块高度一致，则随机（几率50%）进行切换操作;*所有的矿工的客户端随机，可能在此阶段会有短暂的
        分叉，但最终会切换到同一条难度最大的链*
    
    分叉操作：
    1. 将叫高度的块削减到和低块一样的高度
    2. 找寻两块共同的祖先
    3. 从祖先开始重新插入新链
    4. 比较新链、旧链之间交易的不同，广播相应的事件。
5. 插入完毕，广播相应事件。