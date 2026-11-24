package types

// V327APRFeatures are the v3.2.7 switches for the staking APR query. None:
// Query.Apr and Query.DelegatorApr (x/stakingtier) only read state, and the
// settlement code they share (stakingtier keeper/rates.go) was moved, not
// changed, so blocks replay identically without a switch.
var V327APRFeatures = []string{}
