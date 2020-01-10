package constants

import (
	"github.com/blang/semver"
)

var int64Overrides = map[ConstantName]int64{}
var boolOverrides = map[ConstantName]bool{}

// The version of this software
var SWVersion, _ = semver.Make("0.1.0")

// ConstantValue010 implement ConstantValues interface for version 0.1.0
type ConstantValue010 struct {
	int64values map[ConstantName]int64
	boolValues  map[ConstantName]bool
}

// NewConstantValue010 get new instance of ConstantValue010
func NewConstantValue010() *ConstantValue010 {
	return &ConstantValue010{
		int64values: map[ConstantName]int64{
			EmissionCurve:                   6,
			BlocksPerYear:                   6311390,
			TransactionFee:                  100_000_000,         // A 1.0 Rune fee on all swaps and withdrawals
			NewPoolCycle:                    50000,               // Enable a pool every 50,000 blocks (~3 days)
			MinimumNodesForYggdrasil:        6,                   // No yggdrasil pools if THORNode have less than 6 active nodes
			MinimumNodesForBFT:              4,                   // Minimum node count to keep network running. Below this, Ragnarök is performed.
			ValidatorRotateInNumBeforeFull:  2,                   // How many validators should THORNode nominate before THORNode reach the desire validator set
			ValidatorRotateOutNumBeforeFull: 1,                   // How many validators should THORNode queued to be rotate out before THORNode reach the desire validator set)
			ValidatorRotateNumAfterFull:     1,                   // How many validators should THORNode nominate after THORNode reach the desire validator set
			DesireValidatorSet:              33,                  // desire validator set
			FundMigrationInterval:           360,                 // number of blocks THORNode will attempt to move funds from a retiring vault to an active one
			RotatePerBlockHeight:            51840,               // How many blocks THORNode try to rotate validators
			BadValidatorRate:                51840,               // rate to mark a validator to be rotated out for bad behavior
			OldValidatorRate:                51840,               // rate to mark a validator to be rotated out for age
			LackOfObservationPenalty:        2,                   // add two slash point for each block where a node does not observe
			SigningTransactionPeriod:        100,                 // how many blocks before a request to sign a tx by yggdrasil pool, is counted as delinquent.
			MinimumBondInRune:               100_000_000_000_000, // 1 million rune
			WhiteListGasAsset:               1000,                // thor coins we will be given to the validator
		},
		boolValues: map[ConstantName]bool{
			StrictBondStakeRatio: false,
		}}
}

// GetInt64Value get value in int64 type, if it doesn't exist then it will return the default value of int64, which is 0
func (cv *ConstantValue010) GetInt64Value(name ConstantName) int64 {
	// check overrides first
	v, ok := int64Overrides[name]
	if ok {
		return v
	}

	v, ok = cv.int64values[name]
	if !ok {
		return 0
	}
	return v
}

// GetBoolValue retrieve a bool constant value from the map
func (cv *ConstantValue010) GetBoolValue(name ConstantName) bool {
	v, ok := boolOverrides[name]
	if ok {
		return v
	}
	v, ok = cv.boolValues[name]
	if !ok {
		return false
	}
	return v
}
