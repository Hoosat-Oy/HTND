package difficultymanager

import (
	"math"
	"math/big"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/util/difficulty"
)

type difficultyBlock struct {
	timeInMilliseconds int64
	Bits               uint32
	hash               *externalapi.DomainHash
	blueWork           *big.Int
}

type blockWindow []difficultyBlock

func (dm *difficultyManager) getDifficultyBlock(
	header externalapi.BlockHeader, blockHash *externalapi.DomainHash) (difficultyBlock, error) {
	return difficultyBlock{
		timeInMilliseconds: header.TimeInMilliseconds(),
		Bits:               header.Bits(),
		hash:               blockHash,
		blueWork:           header.BlueWork(),
	}, nil
}

// blockWindow returns a blockWindow of the given size that contains the
// blocks in the past of startingNode, the sorting is unspecified.
// If the number of blocks in the past of startingNode is less then windowSize,
// the window will be padded by genesis blocks to achieve a size of windowSize.
func (dm *difficultyManager) blockWindow(
	stagingArea *model.StagingArea,
	startingNode *externalapi.DomainHash,
	windowSize int,
) (blockWindow, []*externalapi.DomainHash, error) {

	window := make(blockWindow, 0, windowSize)
	windowHashes, err := dm.dagTraversalManager.BlockWindow(stagingArea, startingNode, windowSize)
	if err != nil {
		return nil, nil, err
	}

	for _, hash := range windowHashes {
		header, err := dm.headerStore.BlockHeader(dm.databaseContext, stagingArea, hash)
		if err != nil {
			return nil, nil, err
		}
		block, err := dm.getDifficultyBlock(header, hash)
		if err != nil {
			return nil, nil, err
		}
		window = append(window, block)
	}

	// Check if the last header has DAAScore == 43334187
	if len(windowHashes) > 0 {
		lastHash := windowHashes[len(windowHashes)-1]
		lastHeader, err := dm.headerStore.BlockHeader(dm.databaseContext, stagingArea, lastHash)
		if err != nil {
			return nil, nil, err
		}
		if lastHeader.DAAScore() <= 43334187 {
			lastBlock, err := dm.getDifficultyBlock(lastHeader, lastHash)
			if err != nil {
				return nil, nil, err
			}
			singleWindow := blockWindow{lastBlock}
			return singleWindow, []*externalapi.DomainHash{lastHash}, nil
		}
	}

	return window, windowHashes, nil
}

func ghostdagLess(blockA *difficultyBlock, blockB *difficultyBlock) bool {
	switch blockA.blueWork.Cmp(blockB.blueWork) {
	case -1:
		return true
	case 1:
		return false
	case 0:
		return blockA.hash.Less(blockB.hash)
	default:
		panic("big.Int.Cmp is defined to always return -1/1/0 and nothing else")
	}
}

func (window blockWindow) minMaxTimestamps() (min, max int64, minIndex int) {
	min = math.MaxInt64
	minIndex = 0
	max = 0
	for i, block := range window {
		// If timestamps are equal we ghostdag compare in order to reach consensus on `minIndex`
		if block.timeInMilliseconds < min ||
			(block.timeInMilliseconds == min && ghostdagLess(&block, &window[minIndex])) {
			min = block.timeInMilliseconds
			minIndex = i
		}
		if block.timeInMilliseconds > max {
			max = block.timeInMilliseconds
		}
	}
	return
}

func (window *blockWindow) remove(n int) {
	(*window)[n] = (*window)[len(*window)-1]
	*window = (*window)[:len(*window)-1]
}

func (window blockWindow) averageTarget() *big.Int {
	averageTarget := new(big.Int)
	targetTmp := new(big.Int)
	for _, block := range window {
		difficulty.CompactToBigWithDestination(block.Bits, targetTmp)
		averageTarget.Add(averageTarget, targetTmp)
	}
	return averageTarget.Div(averageTarget, big.NewInt(int64(len(window))))
}
