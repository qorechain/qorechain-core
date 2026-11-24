package types

// V327SVMClusterFeatures are the v3.2.7 switches for the SVM genesis hash and cluster id.
//
// Empty on purpose: the item is query code only. The genesis hash is derived
// from the chain id (x/svm/types/cluster.go) and served by the ClusterInfo
// gRPC/REST query and the SVM JSON-RPC methods getGenesisHash/getVersion. No
// state is written, no gas is charged in a tx, no tx is accepted or refused
// differently and no event is emitted, so replay below the v3.2.7 height is
// unchanged without a switch.
var V327SVMClusterFeatures = []string{}
