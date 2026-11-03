package types

const (
	// FeatureEVMWindowUsedEvent makes every EVM transaction that the ante
	// handler admits against an account's post-quantum EVM window emit a
	// pqc_evm_window_used event (EventTypeEVMWindowUsed), carrying the
	// account, the height the window was opened at, the window's used
	// transactions and used value after this transaction, and what this
	// transaction took from it. It writes no state and charges no gas. Before
	// it the consumption was recorded in the stored window only, and a wallet
	// had to query the window before and after to learn what a transaction
	// used.
	FeatureEVMWindowUsedEvent = "evm-window-used-event"
)

// V327EVMWindowFeatures are the v3.2.7 switches for the EVM window usage event.
var V327EVMWindowFeatures = []string{
	FeatureEVMWindowUsedEvent,
}
