package app

import storetypes "cosmossdk.io/store/types"

// nodeLocalABCIListeners are ABCI listeners that feed node-local services
// (never consensus). The full build adds the x/ai node insights listener while
// wiring its gates; the list is installed after the streaming services.
var nodeLocalABCIListeners []storetypes.ABCIListener
