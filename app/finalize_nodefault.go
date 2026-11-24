package app

import (
	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

// FinalizeBlock is BaseApp's, plus the SVM node-fault latch: if a NodeFault
// was raised while the block ran (svm-execution-thread-node-fault,
// svm-deploy-thread-node-fault), the block fails even when a caller in
// between recovered the panic (x/gov proposal execution, for one). A failed
// FinalizeBlock stops consensus on this node before Commit, so nothing it
// computed differently is committed; on restart the block runs again. The
// latch is set only when a NodeFault is raised, which happens only with those
// switches on and only during block execution.
func (app *QoreChainApp) FinalizeBlock(req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	return finalizeWithNodeFaultLatch(app.Logger(), req.Height, func() (*abci.ResponseFinalizeBlock, error) {
		return app.App.FinalizeBlock(req)
	})
}

// finalizeWithNodeFaultLatch runs one block and fails it if the SVM
// node-fault latch was set while it ran.
func finalizeWithNodeFaultLatch(logger log.Logger, height int64,
	run func() (*abci.ResponseFinalizeBlock, error)) (*abci.ResponseFinalizeBlock, error) {
	svmtypes.ClearNodeFault()
	res, err := run()
	if f := svmtypes.PendingNodeFault(); f != nil {
		logger.Error("SVM NODE FAULT: block not finalized on this node; halting before commit",
			"height", height, "fault", f.Error())
		return nil, *f
	}
	return res, err
}
