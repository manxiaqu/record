# POA(Clique)
POA共识算法详解

# 简介
POA是一个基于许可的共识，只有经过认证的地址才能进行挖矿生成块（没有块奖励）；认证的地址可以通过协议
动态增减

#　参数介绍
1. EPOCH_LENGTH: 经过多少个块后，设置检查点
2. BLOCK_PERIOD: 平均出块时间
3. EXTRA_VANITY： 在extra-data字段的头部数据中，为signer vanity留出的固定字节的数量 
4. EXTRA_SEAL： 在extra-data字段的尾部数据中，为signer seal留出的固定字节的数量
5. NONCE_AUTH： 添加一个新的signer时，将nonce设置为"0xffffffffffffffff"
6. NONCE_DROP: 删除一个signer时，将nonce设置为"0x0000000000000000"
7. UNCLE_HASH: uncle hash，值为Keccak256(RLP([]))，因为在poa中，uncle没有任何意义
8. DIFF_NOTURN：使用block中的difficulty字段，1代表outturn：代表顺序，当前的块应该有另外一个signer进行签名，但是
9. DIFF_INTURN：使用block中的difficulty字段，2代表inturn：表示按照顺序，该块应该由当前的signer进行签名。
10. SIGNER_COUNT：在一个特定的时间点，链中合法的signer个数
11. SIGNER_INDEX：合法signer的索引
12. SIGNER_LIMIT：

# 块header结构
1. difficulty：1代表：；2代表
2. extraData：额外数据，含signer的签名数据
3. gasLimit：当前块的gas限制，同pow
4. gasUsed：使用的gas数量，同pow
5. hash：该块hash，同pow
6. logsBloom：同pow
7. miner：不进行投票时，为0，投票时，被投票表决（删除或者加入）的地址。（可设为任意值）
8. mixHash：保留字段，正常情况下为0;
9. nonce："0x0000000000000000"代表删除一个singer，"0xffffffffffffffff"代表添加一个signer
10. number：块高度，同pow
11. parentHash：父块hash，同pow
12. receiptsRoot：同pow
13. sha3Uncles：为固定字符串"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"
    1. 该字符串为web3.sha3("0xc0",{encoding:'hex'})的结果，其中0xc0为RLP([])的结果（即空数组的rlp编码）
14. size：块大小，同pow
15. stateRoot：同pow
16. timestamp：出块时间，>=parentBlock.timestamp+BLOCK_PERIOD
17. totalDifficulty：同pow，但基本无实际意义
18. transactions：同pow
19. transactionsRoot：同pow
20. uncles：应该始终为空数组[]

# POA出块流程
POA的块是由signer轮流进行签名的

1. 尝试生成新的块，调用Seal接口
```go
func (c *Clique) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if c.config.Period == 0 && len(block.Transactions()) == 0 {
		return nil, errWaitTransactions
	}
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	if _, authorized := snap.Signers[signer]; !authorized {
		return nil, errUnauthorized
	}
	// If we're amongst the recent signers, wait for the next block
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among recents, only wait if the current block doesn't shift it out
			if limit := uint64(len(snap.Signers)/2 + 1); number < limit || seen > number-limit {
				log.Info("Signed recently, must wait for others")
				<-stop
				return nil, nil
			}
		}
	}
	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(header.Time.Int64(), 0).Sub(time.Now()) // nolint: gosimple
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Signers)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle)))

		log.Trace("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))

	select {
	case <-stop:
		return nil, nil
	case <-time.After(delay):
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	return block.WithSeal(header), nil
}
```
* 对于period为0的链，不允许生成空块
* 获取上个块的快照，仅允许授权的用户生成块
* 

