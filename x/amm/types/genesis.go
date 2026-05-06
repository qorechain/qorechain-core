package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// GenesisState is the AMM module's exportable/importable state.
type GenesisState struct {
	Params      Params      `json:"params"`
	Pools       []Pool      `json:"pools"`
	LPBalances  []LPBalance `json:"lp_balances"`
	NextPoolID  uint64      `json:"next_pool_id"`
	PausedPools []uint64    `json:"paused_pools"`
}

// DefaultGenesisState returns a fresh chain's AMM state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Pools:       []Pool{},
		LPBalances:  []LPBalance{},
		NextPoolID:  1,
		PausedPools: []uint64{},
	}
}

// Validate enforces:
//   - Params are valid
//   - Every pool is internally valid
//   - Pool IDs are unique
//   - NextPoolID > max(pool.ID) (so the next mint doesn't collide)
//   - Sum of LP balances per pool equals pool.LPSupply (no minted but
//     unaccounted LP tokens, no orphan balances)
//   - Each LPBalance.PoolID corresponds to an existing pool
//   - PausedPools contains only known pool IDs
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("params: %w", err)
	}

	poolByID := make(map[uint64]*Pool, len(gs.Pools))
	maxID := uint64(0)
	for i := range gs.Pools {
		p := &gs.Pools[i]
		if err := p.Validate(); err != nil {
			return fmt.Errorf("pool[%d] (id=%d): %w", i, p.ID, err)
		}
		if _, dup := poolByID[p.ID]; dup {
			return fmt.Errorf("duplicate pool id %d", p.ID)
		}
		poolByID[p.ID] = p
		if p.ID > maxID {
			maxID = p.ID
		}
	}
	if gs.NextPoolID <= maxID {
		return fmt.Errorf("next_pool_id (%d) must be > max(pool.ID) (%d)", gs.NextPoolID, maxID)
	}

	// Sum LP balances per pool.
	lpSumByPool := make(map[uint64]math.Int, len(poolByID))
	for i, b := range gs.LPBalances {
		if err := b.Validate(); err != nil {
			return fmt.Errorf("lp_balance[%d]: %w", i, err)
		}
		if _, ok := poolByID[b.PoolID]; !ok {
			return fmt.Errorf("lp_balance[%d] references unknown pool id %d", i, b.PoolID)
		}
		cur, ok := lpSumByPool[b.PoolID]
		if !ok {
			cur = math.ZeroInt()
		}
		lpSumByPool[b.PoolID] = cur.Add(b.Amount)
	}
	for id, p := range poolByID {
		sum, ok := lpSumByPool[id]
		if !ok {
			sum = math.ZeroInt()
		}
		if !sum.Equal(p.LPSupply) {
			return fmt.Errorf("pool %d: sum(lp_balances) = %s != pool.LPSupply = %s", id, sum, p.LPSupply)
		}
	}

	// Paused pool IDs must reference known pools.
	for _, id := range gs.PausedPools {
		if _, ok := poolByID[id]; !ok {
			return fmt.Errorf("paused_pools references unknown pool id %d", id)
		}
	}

	return nil
}
