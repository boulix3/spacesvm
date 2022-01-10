// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"github.com/ava-labs/avalanchego/database"
)

var _ UnsignedTransaction = &LifelineTx{}

type LifelineTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`
}

func addLife(db database.KeyValueReaderWriter, prefix []byte) error {
	i, has, err := GetPrefixInfo(db, prefix)
	if err != nil {
		return err
	}
	// Cannot add time to missing prefix
	if !has {
		return ErrPrefixMissing
	}
	// Lifeline spread across all units
	lastExpiry := i.Expiry
	prefixPenalty := prefixUnits(prefix) / PrefixRenewalDiscount
	if prefixPenalty < 1 { // avoid division by 0
		prefixPenalty = 1
	}

	i.Expiry += ExpiryTime / i.Units / prefixPenalty
	return PutPrefixInfo(db, prefix, i, lastExpiry)
}

func (l *LifelineTx) Execute(db database.Database, blockTime uint64) error {
	return addLife(db, l.Prefix)
}

func (l *LifelineTx) Copy() UnsignedTransaction {
	return &LifelineTx{
		BaseTx: l.BaseTx.Copy(),
	}
}
