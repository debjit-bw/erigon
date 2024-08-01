// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package stagedsync

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"github.com/erigontech/erigon-lib/common/datadir"
	"github.com/erigontech/erigon-lib/kv/temporal"
	"github.com/erigontech/erigon-lib/log/v3"
	"github.com/erigontech/erigon/turbo/stages/headerdownload"

	"github.com/erigontech/erigon-lib/commitment"
	"github.com/erigontech/erigon-lib/kv/rawdbv3"
	"github.com/erigontech/erigon/common/math"
	"github.com/erigontech/erigon/turbo/services"

	libcommon "github.com/erigontech/erigon-lib/common"
	"github.com/erigontech/erigon-lib/kv"
	"github.com/erigontech/erigon-lib/state"
	"github.com/erigontech/erigon/core/types"
	"github.com/erigontech/erigon/turbo/trie"
)

func collectAndComputeCommitment(ctx context.Context, db kv.RwDB, tx kv.RwTx, agg *state.Aggregator, tmpDir string, toTxNum uint64) ([]byte, error) {
	domains, err := state.NewSharedDomains(tx, log.New())
	if err != nil {
		return nil, err
	}
	defer domains.Close()
	//ac := domains.AggTx().(*state.AggregatorRoTx)

	// has to set this value because it will be used during domain.Commit() call.
	// If we do not, txNum of block beginning will be used, which will cause invalid txNum on restart following commitment rebuilding
	blockFound, blockNum, err := rawdbv3.TxNums.FindBlockNum(tx, toTxNum)
	if err != nil {
		return nil, err
	}
	if !blockFound {
		return nil, fmt.Errorf("block not found for txnum %d", toTxNum-1)
	}
	domains.SetTxNum(toTxNum - 1)
	domains.SetBlockNum(blockNum)
	step := (toTxNum - 1) / agg.StepSize()

	sdCtx := state.NewSharedDomainsCommitmentContext(domains, commitment.ModeDirect, commitment.VariantHexPatriciaTrie)
	rh, err := sdCtx.ComputeCommitment(ctx, true, domains.BlockNum(), "Finalizing")
	if err != nil {
		return nil, err
	}

	logger := log.New()
	logger.Info("Commitment has been reevaluated",
		"block", domains.BlockNum(),
		"tx", domains.TxNum(),
		"root", hex.EncodeToString(rh),
	)
	//"processed", processed.Load(),
	//"total", totalKeys.Load())

	logger.Info("flushing latest step on disk", "step", step)

	if err = domains.CommitmentInMem(agg, step, toTxNum-agg.StepSize(), toTxNum, logger); err != nil {
		return nil, err
	}
	return nil, nil
	//
	//logger.Info("Collecting account/storage keys", "block", domains.BlockNum(), "txnum", toTxNum)
	//collector := etl.NewCollector("CollectKeys", tmpDir, etl.NewSortableBuffer(etl.BufferOptimalSize/2), logger)
	//defer collector.Close()
	//
	//var totalKeys atomic.Uint64
	//it, err := ac.DomainRangeLatest(tx, kv.AccountsDomain, nil, nil, -1)
	//if err != nil {
	//	return nil, err
	//}
	//for it.HasNext() {
	//	k, _, err := it.Next()
	//	if err != nil {
	//		return nil, err
	//	}
	//	if err := collector.Collect(k, nil); err != nil {
	//		return nil, err
	//	}
	//	totalKeys.Add(1)
	//	//if totalKeys.Load() > 500000 {
	//	//	break
	//	//}
	//}
	//
	//logger.Info("Collected account keys", "count", totalKeys.Load())
	//
	////it, err = ac.DomainRangeLatest(tx, kv.CodeDomain, nil, nil, -1)
	////if err != nil {
	////	return nil, err
	////}
	////for it.HasNext() {
	////	k, _, err := it.Next()
	////	if err != nil {
	////		return nil, err
	////	}
	////	if err := collector.Collect(k, nil); err != nil {
	////		return nil, err
	////	}
	////	totalKeys.Add(1)
	////}
	//
	//it, err = ac.DomainRangeLatest(tx, kv.StorageDomain, nil, nil, -1)
	//if err != nil {
	//	return nil, err
	//}
	//for it.HasNext() {
	//	k, _, err := it.Next()
	//	if err != nil {
	//		return nil, err
	//	}
	//	if err := collector.Collect(k, nil); err != nil {
	//		return nil, err
	//	}
	//	totalKeys.Add(1)
	//	//if totalKeys.Load() > 1000000 {
	//	//	break
	//	//}
	//}
	//logger.Info("Collected storage keys", "total keys", totalKeys.Load())
	//
	//lastStep := ac.EndTxNumNoCommitment() / agg.StepSize()
	//
	//batchSteps := uint64(32)
	//bigBatches := lastStep / batchSteps
	//
	//smallBatchSize := totalKeys.Load() / lastStep
	//batchSize := smallBatchSize * batchSteps
	//
	//cStep := batchSteps - 1
	//cTxFrom, cTxTo := uint64(0), batchSteps*agg.StepSize()
	//
	//domains.SetTxNum(cTxFrom)
	//ok, bn, err := rawdbv3.TxNums.FindBlockNum(tx, cTxTo-1)
	//if err == nil && ok {
	//	domains.SetBlockNum(bn)
	//} else {
	//	domains.SetBlockNum(1)
	//}
	//
	//var processed atomic.Uint64
	//logger.Warn("Begun commitment", "lastStep", lastStep, "batchSize", batchSize, "totalKeys", totalKeys.Load(), "bigBatches", bigBatches)
	//
	//sdCtx := state.NewSharedDomainsCommitmentContext(domains, commitment.ModeDirect, commitment.VariantHexPatriciaTrie)
	//loadKeys := func(k, v []byte, table etl.CurrentTableReader, next etl.LoadNextFunc) error {
	//	if sdCtx.KeysCount() >= batchSize {
	//		rh, err := sdCtx.ComputeCommitment(ctx, true, domains.BlockNum(), fmt.Sprintf("applying shard %d", cStep))
	//		if err != nil {
	//			return err
	//		}
	//		logger.Info("Committing batch",
	//			"processed", fmt.Sprintf("%dM/%dM (%.2f%%)", processed.Load()/1_000_000, totalKeys.Load()/1_000_000, float64(processed.Load())/float64(totalKeys.Load())*100),
	//			"intermediate root", fmt.Sprintf("%x", rh))
	//
	//		if err = domains.CommitmentInMem(agg, cStep, cTxFrom, cTxTo, logger); err != nil {
	//			return err
	//		}
	//		domains.ClearRam(false)
	//
	//		bigBatches--
	//		if bigBatches == 0 {
	//			batchSize = smallBatchSize
	//			batchSteps = 1
	//		}
	//
	//		cStep += batchSteps
	//		cTxFrom = cTxTo
	//		cTxTo = cTxFrom + batchSteps*agg.StepSize()
	//
	//		domains.SetTxNum(cTxFrom)
	//		ok, bn, err := rawdbv3.TxNums.FindBlockNum(tx, cTxTo-1)
	//		if err == nil && ok {
	//			domains.SetBlockNum(bn)
	//		}
	//	}
	//	processed.Add(1)
	//	sdCtx.TouchKey(kv.AccountsDomain, string(k), nil)
	//
	//	return nil
	//}
	//err = collector.Load(nil, "", loadKeys, etl.TransformArgs{Quit: ctx.Done()})
	//if err != nil {
	//	return nil, err
	//}
	//collector.Close()
	//
	//rh, err := sdCtx.ComputeCommitment(ctx, true, domains.BlockNum(), "Finalizing")
	//if err != nil {
	//	return nil, err
	//}
	//
	//logger.Info("Commitment has been reevaluated",
	//	"block", domains.BlockNum(),
	//	"tx", domains.TxNum(),
	//	"root", hex.EncodeToString(rh),
	//	"processed", processed.Load(),
	//	"total", totalKeys.Load())
	//
	//logger.Info("flushing latest step on disk", "step", lastStep)
	//
	//cStep++
	//cTxFrom = cTxTo
	//cTxTo = cTxFrom + agg.StepSize()
	//if err = domains.CommitmentInMem(agg, cStep, cTxFrom, cTxTo, logger); err != nil {
	//	return nil, err
	//}
	//
	//return rh, nil
}

type blockBorders struct {
	Number    uint64
	FirstTx   uint64
	CurrentTx uint64
	LastTx    uint64
}

func (b blockBorders) Offset() uint64 {
	if b.CurrentTx > b.FirstTx && b.CurrentTx < b.LastTx {
		return b.CurrentTx - b.FirstTx
	}
	return 0
}

func countBlockByTxnum(ctx context.Context, tx kv.Tx, blockReader services.FullBlockReader, txnum uint64) (bb blockBorders, err error) {
	var txCounter uint64 = 0

	for i := uint64(0); i < math.MaxUint64; i++ {
		if i%1000000 == 0 {
			fmt.Printf("\r [%s] Counting block for txn %d: cur block %dM cur txn %d\n", "restoreCommit", txnum, i/1_000_000, txCounter)
		}

		h, err := blockReader.HeaderByNumber(ctx, tx, i)
		if err != nil {
			return blockBorders{}, err
		}

		bb.Number = i
		bb.FirstTx = txCounter
		txCounter++
		b, err := blockReader.BodyWithTransactions(ctx, tx, h.Hash(), i)
		if err != nil {
			return blockBorders{}, err
		}
		txCounter += uint64(len(b.Transactions))
		txCounter++
		bb.LastTx = txCounter

		if txCounter >= txnum {
			bb.CurrentTx = txnum
			return bb, nil
		}
	}
	return blockBorders{}, fmt.Errorf("block with txn %x not found", txnum)
}

type TrieCfg struct {
	db                kv.RwDB
	checkRoot         bool
	badBlockHalt      bool
	tmpDir            string
	saveNewHashesToDB bool // no reason to save changes when calculating root for mining
	blockReader       services.FullBlockReader
	hd                *headerdownload.HeaderDownload

	historyV3 bool
	agg       *state.Aggregator
}

func StageTrieCfg(db kv.RwDB, checkRoot, saveNewHashesToDB, badBlockHalt bool, tmpDir string, blockReader services.FullBlockReader, hd *headerdownload.HeaderDownload, historyV3 bool, agg *state.Aggregator) TrieCfg {
	return TrieCfg{
		db:                db,
		checkRoot:         checkRoot,
		tmpDir:            tmpDir,
		saveNewHashesToDB: saveNewHashesToDB,
		badBlockHalt:      badBlockHalt,
		blockReader:       blockReader,
		hd:                hd,

		historyV3: historyV3,
		agg:       agg,
	}
}

type HashStateCfg struct {
	db   kv.RwDB
	dirs datadir.Dirs
}

func StageHashStateCfg(db kv.RwDB, dirs datadir.Dirs) HashStateCfg {
	return HashStateCfg{
		db:   db,
		dirs: dirs,
	}
}

var ErrInvalidStateRootHash = fmt.Errorf("invalid state root hash")

func RebuildPatriciaTrieBasedOnFiles(rwTx kv.RwTx, cfg TrieCfg, ctx context.Context, logger log.Logger) (libcommon.Hash, error) {
	useExternalTx := rwTx != nil
	if !useExternalTx {
		var err error
		rwTx, err = cfg.db.BeginRw(context.Background())
		if err != nil {
			return trie.EmptyRoot, err
		}
		defer rwTx.Rollback()
	}

	var foundHash bool
	toTxNum := rwTx.(*temporal.Tx).AggTx().(*state.AggregatorRoTx).EndTxNumNoCommitment()
	ok, blockNum, err := rawdbv3.TxNums.FindBlockNum(rwTx, toTxNum)
	if err != nil {
		return libcommon.Hash{}, err
	}
	if !ok {
		bb, err := countBlockByTxnum(ctx, rwTx, cfg.blockReader, toTxNum)
		if err != nil {
			return libcommon.Hash{}, err
		}
		blockNum = bb.Number
		foundHash = bb.Offset() != 0
	} else {
		firstTxInBlock, err := rawdbv3.TxNums.Min(rwTx, blockNum)
		if err != nil {
			return libcommon.Hash{}, fmt.Errorf("failed to find first txNum in block %d : %w", blockNum, err)
		}
		lastTxInBlock, err := rawdbv3.TxNums.Max(rwTx, blockNum)
		if err != nil {
			return libcommon.Hash{}, fmt.Errorf("failed to find last txNum in block %d : %w", blockNum, err)
		}
		if firstTxInBlock == toTxNum || lastTxInBlock == toTxNum {
			foundHash = true // state is in the beginning or end of block
		}
	}

	var expectedRootHash libcommon.Hash
	var headerHash libcommon.Hash
	var syncHeadHeader *types.Header
	if foundHash && cfg.checkRoot {
		syncHeadHeader, err = cfg.blockReader.HeaderByNumber(ctx, rwTx, blockNum)
		if err != nil {
			return trie.EmptyRoot, err
		}
		if syncHeadHeader == nil {
			return trie.EmptyRoot, fmt.Errorf("no header found with number %d", blockNum)
		}
		expectedRootHash = syncHeadHeader.Root
		headerHash = syncHeadHeader.Hash()
	}

	rh, err := collectAndComputeCommitment(ctx, cfg.db, rwTx, cfg.agg, cfg.tmpDir, toTxNum)
	if err != nil {
		return trie.EmptyRoot, err
	}

	if foundHash && cfg.checkRoot && !bytes.Equal(rh, expectedRootHash[:]) {
		logger.Error(fmt.Sprintf("[RebuildCommitment] Wrong trie root of block %d: %x, expected (from header): %x. Block hash: %x", blockNum, rh, expectedRootHash, headerHash))
		rwTx.Rollback()

		return trie.EmptyRoot, fmt.Errorf("wrong trie root")
	}
	logger.Info(fmt.Sprintf("[RebuildCommitment] Trie root of block %d txNum %d: %x. Could not verify with block hash because txnum of state is in the middle of the block.", blockNum, toTxNum, rh))

	if !useExternalTx {
		if err := rwTx.Commit(); err != nil {
			return trie.EmptyRoot, err
		}
	}
	return libcommon.BytesToHash(rh), err
}
