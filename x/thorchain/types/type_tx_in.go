package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pkg/errors"

	"gitlab.com/thorchain/bepswap/thornode/common"
)

type status string
type TxInIndex common.TxIDs

const (
	Incomplete status = "incomplete"
	Done       status = "done"
	Reverted   status = "reverted"
)

// Meant to track if we have processed a specific tx
type TxIn struct {
	Status             status           `json:"status"`
	NumOuts            int64            `json:"num_outs"`
	OutHashes          common.TxIDs     `json:"out_hashes"` // completed chain tx hash. This is a slice to track if we've "double spent" an input
	Memo               string           `json:"memo"`       // memo
	Coins              common.Coins     `json:"coins"`      // coins sent in tx
	Sender             common.Address   `json:"sender"`
	To                 common.Address   `json:"to"` // to address
	BlockHeight        sdk.Uint         `json:"block_height"`
	Signers            []sdk.AccAddress `json:"signers"` // trust accounts saw this tx
	ObservePoolAddress common.PubKey    `json:"pool_address"`
}

func NewTxIn(coins common.Coins, memo string, sender, to common.Address, height sdk.Uint, observePoolAddress common.PubKey) TxIn {
	return TxIn{
		Coins:              coins,
		Memo:               memo,
		Sender:             sender,
		To:                 to,
		Status:             Incomplete,
		BlockHeight:        height,
		ObservePoolAddress: observePoolAddress,
	}
}

func (tx TxIn) Valid() error {
	if tx.Sender.IsEmpty() {
		return errors.New("sender cannot be empty")
	}
	if tx.To.IsEmpty() {
		return errors.New("to address cannot be empty")
	}
	if len(tx.Coins) == 0 {
		return errors.New("coins cannot be empty")
	}
	for _, coin := range tx.Coins {
		if err := coin.IsValid(); err != nil {
			return err
		}
	}
	// ideally memo should not be empty, we check it here, but if we check it
	// empty here, then the tx will be rejected by thorchain given that , we
	// are not going to refund the transaction, thus we will allow TxIn has
	// empty to get into thorchain. and let thorchain to refund customer
	if tx.BlockHeight.IsZero() {
		return errors.New("block height can't be zero")
	}
	if tx.ObservePoolAddress.IsEmpty() {
		return errors.New("observed pool address is empty")
	}
	return nil
}

func (tx TxIn) Empty() bool {
	return tx.Sender.IsEmpty()
}

func (tx TxIn) Equals(tx2 TxIn) bool {
	if tx.Memo != tx2.Memo {
		return false
	}
	if !tx.Sender.Equals(tx2.Sender) {
		return false
	}
	if !tx.To.Equals(tx2.To) {
		return false
	}
	if !tx.ObservePoolAddress.Equals(tx2.ObservePoolAddress) {
		return false
	}
	if len(tx.Coins) != len(tx2.Coins) {
		return false
	}
	for i := range tx.Coins {
		if !tx.Coins[i].Asset.Equals(tx2.Coins[i].Asset) {
			return false
		}
		if !tx.Coins[i].Amount.Equal(tx2.Coins[i].Amount) {
			return false
		}
	}

	return true
}

func (tx TxIn) String() string {
	hashes := make([]string, len(tx.OutHashes))
	for i, h := range tx.OutHashes {
		hashes[i] = h.String()
	}
	return strings.Join(hashes, ", ")
}

func (tx *TxIn) Sign(signer sdk.AccAddress) {
	for _, sign := range tx.Signers {
		if sign.Equals(signer) {
			return // do nothing
		}
	}
	tx.Signers = append(tx.Signers, signer)
}

func (tx *TxIn) SetOutHash(s status, hash common.TxID) {
	for _, done := range tx.OutHashes {
		if done.Equals(hash) {
			return
		}
	}
	tx.OutHashes = append(tx.OutHashes, hash)
	if int64(len(tx.OutHashes)) >= tx.NumOuts {
		tx.Status = s
	}
}

func (tx *TxIn) SetDone(hash common.TxID) {
	tx.SetOutHash(Done, hash)
}

func (tx *TxIn) SetReverted(hash common.TxID) {
	tx.SetOutHash(Reverted, hash)
}

func (tx *TxIn) GetCommonTx(txid common.TxID) common.Tx {
	return common.NewTx(
		txid,
		tx.Sender,
		tx.To,
		tx.Coins,
		tx.Memo,
	)
}

type TxInVoter struct {
	TxID   common.TxID `json:"tx_id"`
	Txs    []TxIn      `json:"txs"`
	Height int64       `json:"height"`
}

func NewTxInVoter(txID common.TxID, txs []TxIn) TxInVoter {
	return TxInVoter{
		TxID: txID,
		Txs:  txs,
	}
}

func (tx TxInVoter) Valid() error {
	if tx.TxID.IsEmpty() {
		return errors.New("Cannot have an empty tx id")
	}

	for _, in := range tx.Txs {
		if err := in.Valid(); err != nil {
			return err
		}
	}

	return nil
}

func (tx TxInVoter) Key() common.TxID {
	return tx.TxID
}

func (tx TxInVoter) String() string {
	return tx.TxID.String()
}

func (tx *TxInVoter) SetDone(hash common.TxID) {
	for i := range tx.Txs {
		tx.Txs[i].SetDone(hash)
	}
}

func (tx *TxInVoter) Add(txIn TxIn, signer sdk.AccAddress) {
	// check if this signer has already signed, no take backs allowed
	for _, transaction := range tx.Txs {
		for _, siggy := range transaction.Signers {
			if siggy.Equals(signer) {
				return
			}
		}
	}

	for i := range tx.Txs {
		if tx.Txs[i].Equals(txIn) {
			tx.Txs[i].Sign(signer)
			return
		}
	}

	txIn.Sign(signer)
	tx.Txs = append(tx.Txs, txIn)
}

func (tx *TxInVoter) Adds(txs []TxIn, signer sdk.AccAddress) {
	for _, txIn := range txs {
		tx.Add(txIn, signer)
	}
}

func (tx TxInVoter) HasConensus(nodeAccounts NodeAccounts) bool {
	for _, txIn := range tx.Txs {
		var count int
		for _, signer := range txIn.Signers {
			if nodeAccounts.IsTrustAccount(signer) {
				count += 1
			}
		}
		if HasMajority(count, len(nodeAccounts)) {
			return true
		}
	}

	return false
}

func (tx TxInVoter) GetTx(nodeAccounts NodeAccounts) TxIn {
	for _, txIn := range tx.Txs {
		var count int
		for _, signer := range txIn.Signers {
			if nodeAccounts.IsTrustAccount(signer) {
				count += 1
			}
		}
		if HasMajority(count, len(nodeAccounts)) {
			return txIn
		}
	}

	return TxIn{}
}
