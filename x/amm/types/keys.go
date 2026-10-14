package types

const (
	// ModuleName defines the AMM module name.
	ModuleName = "amm"

	// StoreKey is the default store key for the AMM module.
	StoreKey = ModuleName

	// RouterKey is the message route for AMM messages.
	RouterKey = ModuleName

	// QuerierRoute is the querier route for AMM module queries.
	QuerierRoute = ModuleName

	// LPDenomPrefix is the prefix for liquidity-provider token denoms.
	// Each pool's LP token denom is "<prefix>/<pool_id>".
	LPDenomPrefix = "amm-lp"
)

// KV-store prefixes — kept stable across upgrades.
var (
	// PoolPrefix stores Pool by ID.
	// key: PoolPrefix + uint64(pool_id) → Pool
	PoolPrefix = []byte{0x01}

	// PoolByDenomPrefix indexes pools by sorted (denomA, denomB) tuple.
	// key: PoolByDenomPrefix + sortedDenomTuple(a,b) → varint(pool_id)
	PoolByDenomPrefix = []byte{0x02}

	// PoolsByCreatorPrefix indexes pools by creator address.
	// key: PoolsByCreatorPrefix + addr_bytes + uint64(pool_id) → empty
	PoolsByCreatorPrefix = []byte{0x03}

	// LPBalancePrefix stores LP balances by (pool_id, holder_addr).
	// key: LPBalancePrefix + uint64(pool_id) + addr_bytes → math.Int
	LPBalancePrefix = []byte{0x04}

	// NextPoolIDKey holds the next pool ID to be assigned.
	NextPoolIDKey = []byte{0x05}

	// ParamsKey holds the singleton Params value.
	ParamsKey = []byte{0x06}

	// PausedPoolPrefix stores governance-paused pool IDs.
	// key: PausedPoolPrefix + uint64(pool_id) → empty
	PausedPoolPrefix = []byte{0x07}

	// PoolByLPDenomPrefix indexes pools by LP denom for fast lookup.
	// key: PoolByLPDenomPrefix + lp_denom → varint(pool_id)
	PoolByLPDenomPrefix = []byte{0x08}

	// BlockRefPricePrefix stores, per pool, the reserves the pool had before
	// its first swap in the current block (v3.2.2, amm-swap-impact-cap). A swap
	// with no slippage bound is measured against that price so a trade placed
	// in front of it in the same block cannot fill it at any price.
	// key: BlockRefPricePrefix + uint64(pool_id) -> {height, reserve_a, reserve_b}
	BlockRefPricePrefix = []byte{0x09}

	// EndBlockCursorKey holds the pool key EndBlock resumes from when there are
	// more pools than it refreshes in one block (v3.2.3, amm-endblock-bound).
	// Absent while every pool fits in one block. Not exported in genesis: an
	// imported chain starts the round at the first pool.
	// key: EndBlockCursorKey -> PoolPrefix + uint64(pool_id)
	EndBlockCursorKey = []byte{0x20}
)
