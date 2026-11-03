package types

const (
	// FeatureEVMWindowUsedEvent makes every charge of an EVM transaction to
	// an account's post-quantum EVM window emit a pqc_evm_window_used event
	// (EventTypeEVMWindowUsed): the ante handler's admission of the sender
	// (source "ante") and each charge of the delegated-actor settlement after
	// execution (source "settlement"). It carries the account, the height the
	// window was opened at, the window's used transactions and used value
	// after the charge, and what the charge added. It writes no state and
	// charges no gas. Before it the consumption was recorded in the stored
	// window only, and a wallet had to query the window before and after to
	// learn what a transaction used.
	FeatureEVMWindowUsedEvent = "evm-window-used-event"
)

// V327EVMWindowFeatures are the v3.2.7 switches for the EVM window usage event.
var V327EVMWindowFeatures = []string{
	FeatureEVMWindowUsedEvent,
}
