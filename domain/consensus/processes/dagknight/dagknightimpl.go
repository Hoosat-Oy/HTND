package dagknight

import (
	"math"
	"sort"

	"github.com/Hoosat-Oy/HTND/util/difficulty"

	"math/big"

	"github.com/Hoosat-Oy/HTND/domain/consensus/database"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/pkg/errors"
)

type dagknighthelper struct {
	k                  []externalapi.KType
	dataStore          model.GHOSTDAGDataStore
	dbAccess           model.DBReader
	dagTopologyManager model.DAGTopologyManager
	headerStore        model.BlockHeaderStore
	genesis            *externalapi.DomainHash
}

// New creates a new instance of this alternative ghostdag impl
func New(
	databaseContext model.DBReader,
	dagTopologyManager model.DAGTopologyManager,
	ghostdagDataStore model.GHOSTDAGDataStore,
	headerStore model.BlockHeaderStore,
	k []externalapi.KType,
	genesisHash *externalapi.DomainHash) model.GHOSTDAGManager {

	return &dagknighthelper{
		dbAccess:           databaseContext,
		dagTopologyManager: dagTopologyManager,
		dataStore:          ghostdagDataStore,
		headerStore:        headerStore,
		k:                  k,
		genesis:            genesisHash,
	}
}

// GHOSTDAG implements model.GHOSTDAGManager.
func (dk *dagknighthelper) GHOSTDAG(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) error {
	return dk.DAGKNIGHT(stagingArea, blockHash)
}

/* --------------------------------------------- */

func (dk *dagknighthelper) DAGKNIGHT(stagingArea *model.StagingArea, blockCandidate *externalapi.DomainHash) error {
	myWork := new(big.Int)
	maxWork := new(big.Int)
	var myScore uint64
	var spScore uint64
	/* find the selectedParent (prefer higher blue work, tie by hash) */
	blockParents, err := dk.dagTopologyManager.Parents(stagingArea, blockCandidate)
	if err != nil {
		return err
	}
	var selectedParent *externalapi.DomainHash
	for _, parent := range blockParents {
		blockData, err := dk.dataStore.Get(dk.dbAccess, stagingArea, parent, false)
		if database.IsNotFoundError(err) {
			log.Infof("GHOSTDAG failed to retrieve with %s\n", parent)
			return err
		}
		if err != nil {
			return err
		}
		blockWork := blockData.BlueWork()
		blockScore := blockData.BlueScore()
		if selectedParent == nil || blockWork.Cmp(maxWork) == 1 || (blockWork.Cmp(maxWork) == 0 && ismoreHash(parent, selectedParent)) {
			selectedParent = parent
			maxWork = blockWork
			spScore = blockScore
		}
	}
	myWork.Set(maxWork)
	myScore = spScore

	/* Goal: iterate blockCandidate's mergeSet and divide it to : blue, blues, reds. */
	var mergeSetBlues = make([]*externalapi.DomainHash, 0)
	var mergeSetReds = make([]*externalapi.DomainHash, 0)
	var blueSet = make([]*externalapi.DomainHash, 0)

	mergeSetBlues = append(mergeSetBlues, selectedParent)

	mergeSetArr, err := dk.findMergeSet(stagingArea, blockParents, selectedParent)
	if err != nil {
		return err
	}

	err = dk.sortByBlueWork(stagingArea, mergeSetArr)
	if err != nil {
		return err
	}
	err = dk.findBlueSet(stagingArea, &blueSet, selectedParent)
	if err != nil {
		return err
	}

	// Compute a local k based on DAGKnight rank; do not use params directly
	kLocal := 18
	if rank, err := dk.CalculateRank(stagingArea, mergeSetArr); err == nil {
		if rank > 0 && rank < 1024 {
			kLocal = rank
		}
	}

	// Update the shared consensus K slice for current block version
	if dk.k != nil {
		idx := int(constants.GetBlockVersion()) - 1
		if idx >= 0 && idx < len(dk.k) {
			dk.k[idx] = externalapi.KType(kLocal)
		}
	}

	for _, mergeSetBlock := range mergeSetArr {
		if mergeSetBlock.Equal(selectedParent) {
			if !contains(selectedParent, mergeSetBlues) {
				mergeSetBlues = append(mergeSetBlues, selectedParent)
				blueSet = append(blueSet, selectedParent)
			}
			continue
		}
		err := dk.divideBlueRed(stagingArea, selectedParent, mergeSetBlock, &mergeSetBlues, &mergeSetReds, &blueSet, kLocal)
		if err != nil {
			return err
		}
	}
	myScore += uint64(len(mergeSetBlues))

	// We add up all the *work*(not blueWork) that all our blues and selected parent did
	for _, blue := range mergeSetBlues {
		// Virtual genesis has no header; skip adding its work.
		if blue.Equal(model.VirtualGenesisBlockHash) {
			continue
		}
		header, err := dk.headerStore.BlockHeader(dk.dbAccess, stagingArea, blue)
		if err != nil {
			return err
		}
		myWork.Add(myWork, difficulty.CalcWork(header.Bits()))
	}

	e := externalapi.NewBlockGHOSTDAGData(myScore, myWork, selectedParent, mergeSetBlues, mergeSetReds, nil)
	dk.dataStore.Stage(stagingArea, blockCandidate, e, false)
	return nil
}

// OrderDAG implements Algorithm 2: KNIGHT DAG ordering algorithm
func (dk *dagknighthelper) OrderDAG(stagingArea *model.StagingArea, tips []*externalapi.DomainHash) (*externalapi.DomainHash, []*externalapi.DomainHash, error) {
	if len(tips) == 0 {
		return nil, nil, errors.New("no tips")
	}
	if len(tips) == 1 && tips[0].Equal(dk.genesis) {
		return dk.genesis, []*externalapi.DomainHash{dk.genesis}, nil
	}

	// Recursive calls on past of each tip
	chainParentMap := make(map[*externalapi.DomainHash]*externalapi.DomainHash)
	orderMap := make(map[*externalapi.DomainHash][]*externalapi.DomainHash)
	for _, b := range tips {
		pastTips, err := dk.parentsAsTips(stagingArea, b) // Approximation: use parents as 'tips' of past
		if err != nil {
			return nil, nil, err
		}
		chainParent, order, err := dk.OrderDAG(stagingArea, pastTips)
		if err != nil {
			return nil, nil, err
		}
		chainParentMap[b] = chainParent
		orderMap[b] = order
	}

	// P = tips
	p := tips

	for len(p) > 1 {
		// g ← latest common chain ancestor of all B ∈ P
		g, err := dk.latestCommonChainAncestor(stagingArea, p)
		if err != nil {
			return nil, nil, err
		}
		// Partition P into maximal disjoint sets P1, …, Pn
		partitions, err := dk.partitionTips(stagingArea, p, g)
		if err != nil {
			return nil, nil, err
		}
		// Calculate ranks
		ranks := make([]int, len(partitions))
		minRank := math.MaxInt32
		for i, pi := range partitions {
			rank, err := dk.CalculateRank(stagingArea, pi)
			if err != nil {
				return nil, nil, err
			}
			ranks[i] = rank
			if rank < minRank {
				minRank = rank
			}
		}
		// Collect Pi with min rank
		minPartitions := make([][]*externalapi.DomainHash, 0)
		for i, rank := range ranks {
			if rank == minRank {
				minPartitions = append(minPartitions, partitions[i])
			}
		}
		// Tie-Breaking
		p, err = dk.TieBreaking(stagingArea, minPartitions)
		if err != nil {
			return nil, nil, err
		}
	}

	// p is the single element
	theP := p[0]

	// order = order_p || p || anticone(p) in hash topo order
	orderP := orderMap[theP]
	order := append(orderP, theP)
	anticone, err := dk.AnticoneSorted(stagingArea, theP) // Need to implement sorted anticone
	if err != nil {
		return nil, nil, err
	}
	order = append(order, anticone...)

	return theP, order, nil
}

// CalculateRank implements Algorithm 3: Rank calculation procedure
func (dk *dagknighthelper) CalculateRank(stagingArea *model.StagingArea, p []*externalapi.DomainHash) (int, error) {
	// Safety cap to avoid non-termination in adversarial graphs
	const maxK = 64
	for k := 0; k <= maxK; k++ {
		reps, err := dk.Reps(stagingArea, p)
		if err != nil {
			return 0, err
		}
		for _, r := range reps {
			ck, _, err := dk.KColouring(stagingArea, r, k, false)
			if err != nil {
				return 0, err
			}
			// future(g) approx as the DAG
			vote, err := dk.UMCVoting(stagingArea, ck, int(math.Sqrt(float64(k))))
			if err != nil {
				return 0, err
			}
			if vote > 0 {
				return k, nil
			}
		}
	}
	// Fallback if no k <= maxK satisfies the voting condition
	return maxK, nil
}

// TieBreaking implements Algorithm 4: Rank tie-breaking procedure
func (dk *dagknighthelper) TieBreaking(stagingArea *model.StagingArea, partitions [][]*externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	// Mutual rank
	k, err := dk.CalculateRank(stagingArea, partitions[0]) // Assume same for all
	if err != nil {
		return nil, err
	}
	gk := int(math.Sqrt(float64(k)))
	// Virtual G
	virtual := dk.Virtual(stagingArea)
	f, _, err := dk.KColouring(stagingArea, virtual, gk, true)
	if err != nil {
		return nil, err
	}
	minMaxC := math.MaxInt32
	var bestJ int
	for i, pi := range partitions {
		var cMax int
		for kprime := k / 2; kprime <= k; kprime++ {
			_, chainIKprime, err := dk.KColouringConditioned(stagingArea, virtual, kprime, false, pi) // Conditioned on agreeing with Pi
			if err != nil {
				return nil, err
			}
			for _, b := range f {
				anticoneB, err := dk.Anticone(stagingArea, b)
				if err != nil {
					return nil, err
				}
				count := 0
				for _, a := range anticoneB {
					if contains(a, chainIKprime) {
						count++
					}
				}
				if count > kprime {
					cMax += count // Sum as per algo
				}
			}
		}
		maxC := cMax                                                                        // Max over k'
		if maxC < minMaxC || (maxC == minMaxC && ismoreHash(pi[0], partitions[bestJ][0])) { // Tie by hash of first
			minMaxC = maxC
			bestJ = i
		}
	}
	return partitions[bestJ], nil
}

// KColouring implements Algorithm 5: k-colouring algorithm
func (dk *dagknighthelper) KColouring(stagingArea *model.StagingArea, c *externalapi.DomainHash, k int, freeSearch bool) ([]*externalapi.DomainHash, []*externalapi.DomainHash, error) {
	parents, err := dk.dagTopologyManager.Parents(stagingArea, c)
	if err != nil {
		return nil, nil, err
	}
	if len(parents) == 0 {
		return make([]*externalapi.DomainHash, 0), make([]*externalapi.DomainHash, 0), nil
	}
	p := make([]*externalapi.DomainHash, 0)
	bluesMap := make(map[string][]*externalapi.DomainHash) // Use string for hash key
	chainMap := make(map[string][]*externalapi.DomainHash)
	for _, b := range parents {
		agrees, err := dk.Agrees(stagingArea, b, c)
		if err != nil {
			return nil, nil, err
		}
		var bluesB, chainB []*externalapi.DomainHash
		if agrees {
			bluesB, chainB, err = dk.KColouring(stagingArea, b, k, freeSearch)
			if err != nil {
				return nil, nil, err
			}
			p = append(p, b)
			bluesMap[b.String()] = bluesB
			chainMap[b.String()] = chainB
			continue
		}
		if freeSearch {
			bluesB, chainB, err = dk.KColouring(stagingArea, b, k, true)
			if err != nil {
				return nil, nil, err
			}
			p = append(p, b)
			bluesMap[b.String()] = bluesB
			chainMap[b.String()] = chainB
		}
	}
	// B_max arg max |bluesB|, tie hash
	var bMax *externalapi.DomainHash
	maxLen := -1
	for _, b := range p {
		l := len(bluesMap[b.String()])
		if l > maxLen || (l == maxLen && ismoreHash(b, bMax)) {
			maxLen = l
			bMax = b
		}
	}
	if bMax == nil {
		return make([]*externalapi.DomainHash, 0), make([]*externalapi.DomainHash, 0), nil
	}
	bluesG := append(bluesMap[bMax.String()], bMax)
	chainG := append(chainMap[bMax.String()], bMax)
	// anticone(bMax, G) in hash topo order
	anticone, err := dk.AnticoneSorted(stagingArea, bMax)
	if err != nil {
		return nil, nil, err
	}
	for _, b := range anticone {
		anticoneB, err := dk.Anticone(stagingArea, b)
		if err != nil {
			return nil, nil, err
		}
		countChain := 0
		countBlues := 0
		for _, a := range anticoneB {
			if contains(a, chainG) {
				countChain++
			}
			if contains(a, bluesG) {
				countBlues++
			}
		}
		if countChain <= k && countBlues < k {
			bluesG = append(bluesG, b)
		}
	}
	return bluesG, chainG, nil
}

// KColouringConditioned is a variant for conditioned coloring
func (dk *dagknighthelper) KColouringConditioned(stagingArea *model.StagingArea, c *externalapi.DomainHash, k int, freeSearch bool, conditioned []*externalapi.DomainHash) ([]*externalapi.DomainHash, []*externalapi.DomainHash, error) {
	// Similar to KColouring, but include only parents that agree with the conditioned set
	parents, err := dk.dagTopologyManager.Parents(stagingArea, c)
	if err != nil {
		return nil, nil, err
	}
	if len(parents) == 0 {
		return make([]*externalapi.DomainHash, 0), make([]*externalapi.DomainHash, 0), nil
	}
	p := make([]*externalapi.DomainHash, 0)
	bluesMap := make(map[string][]*externalapi.DomainHash)
	chainMap := make(map[string][]*externalapi.DomainHash)
	for _, b := range parents {
		// Must agree with all in conditioned
		agreesAll := true
		for _, cond := range conditioned {
			agrees, aerr := dk.Agrees(stagingArea, b, cond)
			if aerr != nil {
				return nil, nil, aerr
			}
			if !agrees {
				agreesAll = false
				break
			}
		}
		if !agreesAll {
			continue
		}
		bluesB, chainB, err := dk.KColouring(stagingArea, b, k, freeSearch)
		if err != nil {
			return nil, nil, err
		}
		p = append(p, b)
		bluesMap[b.String()] = bluesB
		chainMap[b.String()] = chainB
	}
	// Choose B_max
	var bMax *externalapi.DomainHash
	maxLen := -1
	for _, b := range p {
		l := len(bluesMap[b.String()])
		if l > maxLen || (l == maxLen && ismoreHash(b, bMax)) {
			maxLen = l
			bMax = b
		}
	}
	if bMax == nil {
		return make([]*externalapi.DomainHash, 0), make([]*externalapi.DomainHash, 0), nil
	}
	bluesG := append(bluesMap[bMax.String()], bMax)
	chainG := append(chainMap[bMax.String()], bMax)
	anticone, err := dk.AnticoneSorted(stagingArea, bMax)
	if err != nil {
		return nil, nil, err
	}
	for _, b := range anticone {
		anticoneB, err := dk.Anticone(stagingArea, b)
		if err != nil {
			return nil, nil, err
		}
		countChain := 0
		countBlues := 0
		for _, a := range anticoneB {
			if contains(a, chainG) {
				countChain++
			}
			if contains(a, bluesG) {
				countBlues++
			}
		}
		if countChain <= k && countBlues < k {
			bluesG = append(bluesG, b)
		}
	}
	return bluesG, chainG, nil
}

// UMCVoting implements Algorithm 6: UMC cascade voting procedure
func (dk *dagknighthelper) UMCVoting(stagingArea *model.StagingArea, u []*externalapi.DomainHash, e int) (int, error) {
	// Restrict voting context to future(Virtual()).
	g := dk.Virtual(stagingArea)
	futureG, err := dk.Future(stagingArea, g)
	if err != nil {
		return 0, err
	}

	// Recursive cascade over u intersect future(g)
	uInContext := intersection(u, futureG)
	v := 0
	for _, b := range uInContext {
		futureB, err := dk.Future(stagingArea, b)
		if err != nil {
			return 0, err
		}
		uFuture := intersection(uInContext, futureB)
		vote, err := dk.UMCVoting(stagingArea, uFuture, e)
		if err != nil {
			return 0, err
		}
		v += vote
	}

	// |future(g) \ U|
	gMinusU := len(futureG) - len(uInContext)
	if v-gMinusU+e < 0 {
		return -1, nil
	}
	return 1, nil
}

/* Stub for missing helpers */

func (dk *dagknighthelper) Agrees(stagingArea *model.StagingArea, b *externalapi.DomainHash, c *externalapi.DomainHash) (bool, error) {
	// Agreement relative to future(g): two blocks agree if they lie on the same branch
	// from their latest common chain ancestor.
	g, err := dk.latestCommonChainAncestor(stagingArea, []*externalapi.DomainHash{b, c})
	if err != nil {
		return false, err
	}
	// Child in selected-parent chain from g towards each block
	var childB *externalapi.DomainHash
	var childC *externalapi.DomainHash
	// Guard: if g == b or g == c, skip child lookup (strict ancestor required)
	if g != nil && !g.Equal(b) {
		inChainB, err := dk.dagTopologyManager.IsInSelectedParentChainOf(stagingArea, g, b)
		if err != nil {
			return false, err
		}
		if inChainB {
			childB, err = dk.dagTopologyManager.ChildInSelectedParentChainOf(stagingArea, g, b)
			if err != nil {
				return false, err
			}
		}
	}
	if g != nil && !g.Equal(c) {
		inChainC, err := dk.dagTopologyManager.IsInSelectedParentChainOf(stagingArea, g, c)
		if err != nil {
			return false, err
		}
		if inChainC {
			childC, err = dk.dagTopologyManager.ChildInSelectedParentChainOf(stagingArea, g, c)
			if err != nil {
				return false, err
			}
		}
	}
	// Both on the same branch (including the case g == b or g == c)
	if childB == nil && childC == nil {
		return true, nil
	}
	if childB == nil || childC == nil {
		// One is exactly g's chain start, the other continues; treat as agreeing.
		return true, nil
	}
	return childB.Equal(childC), nil
}

func (dk *dagknighthelper) Reps(stagingArea *model.StagingArea, p []*externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	if len(p) == 0 {
		return make([]*externalapi.DomainHash, 0), nil
	}
	g, err := dk.latestCommonChainAncestor(stagingArea, p)
	if err != nil {
		return nil, err
	}
	partitions, err := dk.partitionTips(stagingArea, p, g)
	if err != nil {
		return nil, err
	}
	reps := make([]*externalapi.DomainHash, 0, len(partitions))
	for _, group := range partitions {
		// Choose representative as max blue-work tip in the group
		var best *externalapi.DomainHash
		var bestWork *big.Int
		for _, tip := range group {
			data, derr := dk.dataStore.Get(dk.dbAccess, stagingArea, tip, false)
			if derr != nil {
				continue
			}
			work := data.BlueWork()
			if best == nil || work.Cmp(bestWork) > 0 || (work.Cmp(bestWork) == 0 && ismoreHash(tip, best)) {
				best = tip
				bestWork = work
			}
		}
		if best != nil {
			reps = append(reps, best)
		}
	}
	return reps, nil
}

func (dk *dagknighthelper) Virtual(stagingArea *model.StagingArea) *externalapi.DomainHash {
	// Virtual context approximated as the latest common chain ancestor of current tips.
	// This ties the virtual g to the branching structure used in DAGKnight ordering.
	tips, err := dk.currentTips(stagingArea)
	if err != nil || len(tips) == 0 {
		return dk.genesis
	}
	g, err := dk.latestCommonChainAncestor(stagingArea, tips)
	if err != nil || g == nil {
		return dk.genesis
	}
	return g
}

func (dk *dagknighthelper) Anticone(stagingArea *model.StagingArea, b *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	// Compute anticone as all reachable blocks which are neither ancestors nor descendants of b
	all, err := dk.AllBlocks(stagingArea)
	if err != nil {
		return nil, err
	}
	past, err := dk.Past(stagingArea, b)
	if err != nil {
		return nil, err
	}
	future, err := dk.Future(stagingArea, b)
	if err != nil {
		return nil, err
	}
	anticone := make([]*externalapi.DomainHash, 0)
	for _, h := range all {
		if !contains(h, past) && !contains(h, future) && !h.Equal(b) {
			anticone = append(anticone, h)
		}
	}
	return anticone, nil
}

func (dk *dagknighthelper) AnticoneSorted(stagingArea *model.StagingArea, b *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	anticone, err := dk.Anticone(stagingArea, b)
	if err != nil {
		return nil, err
	}
	// Sort by hash topo order, assume sort by blue work or hash
	err = dk.sortByBlueWork(stagingArea, anticone)
	return anticone, err
}

func (dk *dagknighthelper) Past(stagingArea *model.StagingArea, b *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	// BFS backward from b using Parents
	visited := make(map[string]bool)
	queue := []*externalapi.DomainHash{b}
	past := make([]*externalapi.DomainHash, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current.String()] {
			continue
		}
		visited[current.String()] = true
		past = append(past, current)
		parents, err := dk.dagTopologyManager.Parents(stagingArea, current)
		if err != nil {
			return nil, err
		}
		queue = append(queue, parents...)
	}
	return past, nil
}

func (dk *dagknighthelper) Future(stagingArea *model.StagingArea, b *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	// BFS forward using Children, excluding the start node b itself
	visited := make(map[string]bool)
	queue := []*externalapi.DomainHash{b}
	future := make([]*externalapi.DomainHash, 0)
	start := b
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current.String()] {
			continue
		}
		visited[current.String()] = true
		if !current.Equal(start) {
			future = append(future, current)
		}
		children, err := dk.dagTopologyManager.Children(stagingArea, current)
		if err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return future, nil
}

func (dk *dagknighthelper) AllBlocks(stagingArea *model.StagingArea) ([]*externalapi.DomainHash, error) {
	// Enumerate all reachable blocks from genesis via Children traversal.
	visited := make(map[string]bool)
	queue := []*externalapi.DomainHash{dk.genesis}
	all := make([]*externalapi.DomainHash, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil {
			continue
		}
		if visited[current.String()] {
			continue
		}
		visited[current.String()] = true
		all = append(all, current)
		children, err := dk.dagTopologyManager.Children(stagingArea, current)
		if err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return all, nil
}

// selectedParentOf returns the selected parent of a block, or nil if none.
func (dk *dagknighthelper) selectedParentOf(stagingArea *model.StagingArea, block *externalapi.DomainHash) (*externalapi.DomainHash, error) {
	if block == nil {
		return nil, nil
	}
	data, err := dk.dataStore.Get(dk.dbAccess, stagingArea, block, false)
	if err != nil {
		return nil, err
	}
	return data.SelectedParent(), nil
}

// currentTips returns all blocks with no children in the reachable DAG.
func (dk *dagknighthelper) currentTips(stagingArea *model.StagingArea) ([]*externalapi.DomainHash, error) {
	all, err := dk.AllBlocks(stagingArea)
	if err != nil {
		return nil, err
	}
	tips := make([]*externalapi.DomainHash, 0)
	for _, h := range all {
		children, cerr := dk.dagTopologyManager.Children(stagingArea, h)
		if cerr != nil {
			return nil, cerr
		}
		if len(children) == 0 {
			tips = append(tips, h)
		}
	}
	return tips, nil
}

func (dk *dagknighthelper) latestCommonChainAncestor(stagingArea *model.StagingArea, p []*externalapi.DomainHash) (*externalapi.DomainHash, error) {
	if len(p) == 0 {
		return dk.genesis, nil
	}
	// Walk down the selected-parent chain of the first tip;
	// pick the closest ancestor that's in the selected-parent chain of all tips.
	current := p[0]
	for current != nil {
		inAll := true
		for i := 1; i < len(p); i++ {
			ok, err := dk.dagTopologyManager.IsInSelectedParentChainOf(stagingArea, current, p[i])
			if err != nil {
				return nil, err
			}
			if !ok {
				inAll = false
				break
			}
		}
		if inAll {
			return current, nil
		}
		// Move to selected parent
		sp, err := dk.selectedParentOf(stagingArea, current)
		if err != nil {
			return nil, err
		}
		current = sp
	}
	return dk.genesis, nil
}

func (dk *dagknighthelper) partitionTips(stagingArea *model.StagingArea, p []*externalapi.DomainHash, g *externalapi.DomainHash) ([][]*externalapi.DomainHash, error) {
	// Partition tips by their first child on the selected-parent chain from g
	// towards the tip (i.e., branch under g).
	byChild := make(map[string][]*externalapi.DomainHash)
	// Use empty key for tips equal to g
	for _, tip := range p {
		// If tip equals g, don't query ChildInSelectedParentChainOf (it requires strict ancestor).
		if g != nil && tip.Equal(g) {
			byChild[tip.String()] = append(byChild[tip.String()], tip)
			continue
		}
		// Ensure g is in the selected-parent chain of tip before calling for the child.
		inChain, err := dk.dagTopologyManager.IsInSelectedParentChainOf(stagingArea, g, tip)
		if err != nil {
			return nil, err
		}
		if !inChain {
			// If not in chain (should be rare since g is LCCA), group by the tip itself.
			byChild[tip.String()] = append(byChild[tip.String()], tip)
			continue
		}
		child, err := dk.dagTopologyManager.ChildInSelectedParentChainOf(stagingArea, g, tip)
		if err != nil {
			return nil, err
		}
		key := tip.String()
		if child != nil {
			key = child.String()
		}
		byChild[key] = append(byChild[key], tip)
	}
	partitions := make([][]*externalapi.DomainHash, 0, len(byChild))
	for _, group := range byChild {
		partitions = append(partitions, group)
	}
	return partitions, nil
}

func (dk *dagknighthelper) parentsAsTips(stagingArea *model.StagingArea, b *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	return dk.dagTopologyManager.Parents(stagingArea, b)
}

// intersection helper
func intersection(a, b []*externalapi.DomainHash) []*externalapi.DomainHash {
	m := make(map[string]bool)
	for _, item := range a {
		m[item.String()] = true
	}
	res := make([]*externalapi.DomainHash, 0)
	for _, item := range b {
		if m[item.String()] {
			res = append(res, item)
		}
	}
	return res
}

/* Existing functions below... */

func ismoreHash(parent *externalapi.DomainHash, selectedParent *externalapi.DomainHash) bool {
	if selectedParent == nil {
		return true
	}
	parentByteArray := parent.ByteArray()
	selectedParentByteArray := selectedParent.ByteArray()
	for i := 0; i < len(parentByteArray); i++ {
		switch {
		case parentByteArray[i] < selectedParentByteArray[i]:
			return false
		case parentByteArray[i] > selectedParentByteArray[i]:
			return true
		}
	}
	return false
}

func (dk *dagknighthelper) divideBlueRed(stagingArea *model.StagingArea,
	selectedParent *externalapi.DomainHash, desiredBlock *externalapi.DomainHash,
	blues *[]*externalapi.DomainHash, reds *[]*externalapi.DomainHash, blueSet *[]*externalapi.DomainHash, k int) error {
	counter := 0

	var suspectsBlues = make([]*externalapi.DomainHash, 0)
	isMergeBlue := true
	for _, block := range *blueSet {
		isAnticone, err := dk.isAnticone(stagingArea, block, desiredBlock)
		if err != nil {
			return err
		}
		if isAnticone {
			counter++
			suspectsBlues = append(suspectsBlues, block)
		}
		if counter > k {
			isMergeBlue = false
			break
		}
	}
	if !isMergeBlue {
		if !contains(desiredBlock, *reds) {
			*reds = append(*reds, desiredBlock)
		}
		return nil
	}

	for _, blue := range suspectsBlues {
		isDestroyed, err := dk.checkIfDestroy(stagingArea, blue, blueSet, k)
		if err != nil {
			return err
		}
		if isDestroyed {
			isMergeBlue = false
			break
		}
	}
	if !isMergeBlue {
		if !contains(desiredBlock, *reds) {
			*reds = append(*reds, desiredBlock)
		}
		return nil
	}
	if !contains(desiredBlock, *blues) {
		*blues = append(*blues, desiredBlock)
	}
	if !contains(desiredBlock, *blueSet) {
		*blueSet = append(*blueSet, desiredBlock)
	}
	return nil
}

func (dk *dagknighthelper) isAnticone(stagingArea *model.StagingArea, blockA, blockB *externalapi.DomainHash) (bool, error) {
	isAAncestorOfB, err := dk.dagTopologyManager.IsAncestorOf(stagingArea, blockA, blockB)
	if err != nil {
		return false, err
	}
	if isAAncestorOfB {
		return false, nil
	}

	isBAncestorOfA, err := dk.dagTopologyManager.IsAncestorOf(stagingArea, blockB, blockA)
	if err != nil {
		return false, err
	}
	return !isBAncestorOfA, nil
}

func contains(item *externalapi.DomainHash, items []*externalapi.DomainHash) bool {
	for _, r := range items {
		if r.Equal(item) {
			return true
		}
	}
	return false
}

func (dk *dagknighthelper) checkIfDestroy(stagingArea *model.StagingArea, blockBlue *externalapi.DomainHash,
	blueSet *[]*externalapi.DomainHash, k int) (bool, error) {
	counter := 0
	for _, blue := range *blueSet {
		isAnticone, err := dk.isAnticone(stagingArea, blue, blockBlue)
		if err != nil {
			return true, err
		}
		if isAnticone {
			counter++
		}
		if counter > k {
			return true, nil
		}
	}
	return false, nil
}

func (dk *dagknighthelper) findMergeSet(stagingArea *model.StagingArea, parents []*externalapi.DomainHash,
	selectedParent *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {

	allMergeSet := make([]*externalapi.DomainHash, 0)
	blockQueue := make([]*externalapi.DomainHash, 0)
	for _, parent := range parents {
		if !contains(parent, blockQueue) {
			blockQueue = append(blockQueue, parent)
		}

	}
	for len(blockQueue) > 0 {
		block := blockQueue[0]
		blockQueue = blockQueue[1:]
		if selectedParent.Equal(block) {
			if !contains(block, allMergeSet) {
				allMergeSet = append(allMergeSet, block)
			}
			continue
		}
		isancestorOf, err := dk.dagTopologyManager.IsAncestorOf(stagingArea, block, selectedParent)
		if err != nil {
			return nil, err
		}
		if isancestorOf {
			continue
		}
		if !contains(block, allMergeSet) {
			allMergeSet = append(allMergeSet, block)
		}
		err = dk.insertParent(stagingArea, block, &blockQueue)
		if err != nil {
			return nil, err
		}

	}
	return allMergeSet, nil
}

func (dk *dagknighthelper) insertParent(stagingArea *model.StagingArea, child *externalapi.DomainHash,
	queue *[]*externalapi.DomainHash) error {

	parents, err := dk.dagTopologyManager.Parents(stagingArea, child)
	if err != nil {
		return err
	}
	for _, parent := range parents {
		if contains(parent, *queue) {
			continue
		}
		*queue = append(*queue, parent)
	}
	return nil
}

func (dk *dagknighthelper) findBlueSet(stagingArea *model.StagingArea, blueSet *[]*externalapi.DomainHash, selectedParent *externalapi.DomainHash) error {
	for selectedParent != nil {
		if !contains(selectedParent, *blueSet) {
			*blueSet = append(*blueSet, selectedParent)
		}
		blockData, err := dk.dataStore.Get(dk.dbAccess, stagingArea, selectedParent, false)
		if database.IsNotFoundError(err) {
			log.Infof("findBlueSet failed to retrieve with %s\n", selectedParent)
			return err
		}
		if err != nil {
			return err
		}
		mergeSetBlue := blockData.MergeSetBlues()
		for _, blue := range mergeSetBlue {
			if contains(blue, *blueSet) {
				continue
			}
			*blueSet = append(*blueSet, blue)
		}
		selectedParent = blockData.SelectedParent()
	}
	return nil
}

func (dk *dagknighthelper) sortByBlueWork(stagingArea *model.StagingArea, arr []*externalapi.DomainHash) error {

	var err error = nil
	sort.Slice(arr, func(i, j int) bool {

		blockLeft, eLeft := dk.dataStore.Get(dk.dbAccess, stagingArea, arr[i], false)
		if eLeft != nil {
			err = eLeft
			return false
		}

		blockRight, eRight := dk.dataStore.Get(dk.dbAccess, stagingArea, arr[j], false)
		if database.IsNotFoundError(eRight) {
			log.Infof("sortByBlueWork failed to retrieve with %s\n", arr[j])
			err = eRight
			return false
		}
		if eRight != nil {
			err = eRight
			return false
		}

		if blockLeft.BlueWork().Cmp(blockRight.BlueWork()) == 1 {
			return true
		}
		if blockLeft.BlueWork().Cmp(blockRight.BlueWork()) == 0 {
			return ismoreHash(arr[i], arr[j])
		}
		return false
	})
	return err
}

// dynamicK removed: rank is computed via CalculateRank per DAGKnight.

func (dk *dagknighthelper) BlockData(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) (*externalapi.BlockGHOSTDAGData, error) {
	return dk.dataStore.Get(dk.dbAccess, stagingArea, blockHash, false)
}

func (dk *dagknighthelper) ChooseSelectedParent(stagingArea *model.StagingArea, blockHashes ...*externalapi.DomainHash) (*externalapi.DomainHash, error) {
	if len(blockHashes) == 0 {
		return nil, nil
	}
	var best *externalapi.DomainHash
	var bestData *externalapi.BlockGHOSTDAGData
	for _, h := range blockHashes {
		data, err := dk.BlockData(stagingArea, h)
		if err != nil {
			return nil, err
		}
		if best == nil || dk.Less(best, bestData, h, data) {
			best = h
			bestData = data
		}
	}
	return best, nil
}

func (dk *dagknighthelper) Less(blockHashA *externalapi.DomainHash, ghostdagDataA *externalapi.BlockGHOSTDAGData, blockHashB *externalapi.DomainHash, ghostdagDataB *externalapi.BlockGHOSTDAGData) bool {
	if ghostdagDataA.BlueScore() != ghostdagDataB.BlueScore() {
		return ghostdagDataA.BlueScore() < ghostdagDataB.BlueScore()
	}

	blueWorkCmp := ghostdagDataA.BlueWork().Cmp(ghostdagDataB.BlueWork())
	if blueWorkCmp != 0 {
		return blueWorkCmp < 0
	}

	return !ismoreHash(blockHashA, blockHashB)
}

func (dk *dagknighthelper) GetSortedMergeSet(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {
	ghostdagData, err := dk.BlockData(stagingArea, blockHash)
	if err != nil {
		return nil, err
	}

	mergeSet := append([]*externalapi.DomainHash{}, ghostdagData.MergeSetBlues()...)
	mergeSet = append(mergeSet, ghostdagData.MergeSetReds()...)

	err = dk.sortByBlueWork(stagingArea, mergeSet)
	if err != nil {
		return nil, err
	}

	return mergeSet, nil
}
