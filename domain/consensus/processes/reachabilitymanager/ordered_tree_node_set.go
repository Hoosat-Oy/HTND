package reachabilitymanager

import (
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
)

// orderedTreeNodeSet is an ordered set of model.DomainHash ordered by the respectful intervals.
// Note that this type does not validate order validity. It's the
// responsibility of the caller to construct instances of this
// type properly.
type orderedTreeNodeSet []*externalapi.DomainHash

// findAncestorOfNode finds the reachability tree ancestor of `node`
// among the nodes in `tns`.
func (rt *reachabilityManager) findAncestorOfNode(stagingArea *model.StagingArea, tns orderedTreeNodeSet, node *externalapi.DomainHash) (*externalapi.DomainHash, bool) {
	ancestorIndex, ok, err := rt.findAncestorIndexOfNode(stagingArea, tns, node)
	if err != nil {
		return nil, false
	}

	if !ok {
		return nil, false
	}

	return tns[ancestorIndex], true
}

// findAncestorIndexOfNode finds the index of the reachability tree
// ancestor of `node` among the nodes in `tns`. It does so by finding
// the index of the block with the maximum start that is below the
// given block.
func (rt *reachabilityManager) findAncestorIndexOfNode(stagingArea *model.StagingArea, tns orderedTreeNodeSet,
	node *externalapi.DomainHash) (int, bool, error) {

	getInterval := func(n *externalapi.DomainHash) (*model.ReachabilityInterval, error) {
		if cached, ok := rt.intervalCache.Load(n); ok {
			return cached.(*model.ReachabilityInterval), nil
		}
		iv, err := rt.interval(stagingArea, n)
		if err != nil {
			return nil, err
		}
		rt.intervalCache.Store(n, iv)
		return iv, nil
	}

	blockInterval, err := getInterval(node)
	if err != nil {
		return 0, false, err
	}
	end := blockInterval.End

	low := 0
	high := len(tns)
	for low < high {
		middle := (low + high) / 2
		middleInterval, err := getInterval(tns[middle])
		if err != nil {
			return 0, false, err
		}
		if end < middleInterval.Start {
			high = middle
		} else {
			low = middle + 1
		}
	}

	if low == 0 {
		return 0, false, nil
	}
	return low - 1, true, nil
}
