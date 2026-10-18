package types

import (
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GenesisState is the stakingtier module's genesis state.
//
// The genuine genesis of a network that takes this module through the v3.2.4
// upgrade has no "stakingtier" entry at all, so InitGenesis does not run for
// it at height 0 and nothing is written; the upgrade handler initialises the
// module at the plan height instead.
type GenesisState struct {
	Params     Params      `json:"params"`
	Pool       Pool        `json:"pool"`
	NextLockID uint64      `json:"next_lock_id"`
	Locks      []Lock      `json:"locks"`
	Claimable  []Claimable `json:"claimable"`
}

// DefaultGenesisState returns the default genesis state.
func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Pool:       DefaultPool(),
		NextLockID: 1,
		Locks:      []Lock{},
		Claimable:  []Claimable{},
	}
}

// Validate checks the genesis state, including that the pool's outstanding
// amount is exactly what the locks and claimable entries are owed.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if err := gs.Pool.Validate(); err != nil {
		return fmt.Errorf("invalid pool: %w", err)
	}
	if gs.NextLockID == 0 {
		return fmt.Errorf("next_lock_id must be at least 1")
	}
	owed := math.ZeroInt()
	seen := map[uint64]bool{}
	open := map[string]int{}
	var openCount uint64
	for i, l := range gs.Locks {
		if err := l.Validate(); err != nil {
			return fmt.Errorf("locks[%d]: %w", i, err)
		}
		if seen[l.ID] {
			return fmt.Errorf("locks[%d]: duplicate id %d", i, l.ID)
		}
		seen[l.ID] = true
		if l.ID >= gs.NextLockID {
			return fmt.Errorf("locks[%d]: id %d is not below next_lock_id %d", i, l.ID, gs.NextLockID)
		}
		if _, err := sdk.AccAddressFromBech32(l.Delegator); err != nil {
			return fmt.Errorf("locks[%d]: invalid delegator: %w", i, err)
		}
		if _, err := sdk.ValAddressFromBech32(l.Validator); err != nil {
			return fmt.Errorf("locks[%d]: invalid validator: %w", i, err)
		}
		if !l.Released {
			open[l.Delegator]++
			openCount++
			if open[l.Delegator] > MaxLocksPerDelegator {
				return fmt.Errorf("locks[%d]: delegator %s holds more than %d open locks", i, l.Delegator, MaxLocksPerDelegator)
			}
		}
		owed = owed.Add(l.Normalized().Accrued)
	}
	claimants := map[string]bool{}
	for i, c := range gs.Claimable {
		if _, err := sdk.AccAddressFromBech32(c.Address); err != nil {
			return fmt.Errorf("claimable[%d]: invalid address: %w", i, err)
		}
		if claimants[c.Address] {
			return fmt.Errorf("claimable[%d]: duplicate address %s", i, c.Address)
		}
		claimants[c.Address] = true
		if c.Amount.IsNil() || !c.Amount.IsPositive() {
			return fmt.Errorf("claimable[%d]: amount must be positive", i)
		}
		owed = owed.Add(c.Amount)
	}
	pool := gs.Pool.Normalized()
	if !pool.Outstanding.Equal(owed) {
		return fmt.Errorf("pool outstanding %s does not equal what locks and claimable entries are owed, %s",
			pool.Outstanding, owed)
	}
	if pool.LockCount != openCount {
		return fmt.Errorf("pool lock_count %d does not equal the %d open locks", pool.LockCount, openCount)
	}
	return nil
}
