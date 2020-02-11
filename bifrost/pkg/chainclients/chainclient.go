package chainclients

import (
	"github.com/binance-chain/go-sdk/common/types"

	"gitlab.com/thorchain/thornode/bifrost/config"
	"gitlab.com/thorchain/thornode/bifrost/metrics"
	pubkeymanager "gitlab.com/thorchain/thornode/bifrost/pubkeymanager"
	stypes "gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/common"
)

// ChainClient is the interface that wraps basic chain client methods
//
// SignTx       signs transactions
// BroadcastTx  broadcast transactions on the chain associated with the client
// GetChain     get chain id
// SignTx       sign transaction
// GetHeight    get chain height
// GetAddress   gets address for public key pool in chain
// GetAccount   gets account from thorclient in cain
// GetGasFee    calculates gas fee based on number of simple transfer sents
// ValidateMetadata  checks if given metadata is correct or not
// Start
// Stop
// InitBlockScanner

// Methods for block scanner and storage interface
// GetTxInForRetry
// GetMessages
// SetTxInStatus
// RemoveTxIn

type ChainClient interface {
	SignTx(tx stypes.TxOutItem, height int64) ([]byte, error)
	BroadcastTx(tx []byte) error
	CheckIsTestNet() (string, bool)
	GetHeight() (int64, error)
	GetAddress(poolPubKey common.PubKey) string
	GetAccount(addr types.AccAddress) (types.BaseAccount, error)
	GetChain() common.Chain
	GetGasFee(count uint64) common.Gas
	ValidateMetadata(_ interface{}) bool
	Start()
	Stop() error
	InitBlockScanner(cfg config.BlockScannerConfiguration, pubkeyMgr pubkeymanager.PubKeyValidator, m *metrics.Metrics) error
	GetMessages() <-chan stypes.TxIn
	GetTxInForRetry(failedOnly bool) ([]stypes.TxIn, error)
	SetTxInStatus(txIn stypes.TxIn, status stypes.TxInStatus) error
	RemoveTxIn(txin stypes.TxIn) error
}
