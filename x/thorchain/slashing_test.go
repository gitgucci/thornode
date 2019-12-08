package thorchain

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"
)

type SlashingSuite struct{}

var _ = Suite(&SlashingSuite{})

func (s *SlashingSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

type TestSlashObservingKeeper struct {
	KVStoreDummy
	addrs []sdk.AccAddress
	nas   NodeAccounts
}

func (k *TestSlashObservingKeeper) GetObservingAddresses(_ sdk.Context) ([]sdk.AccAddress, error) {
	return k.addrs, nil
}

func (k *TestSlashObservingKeeper) ClearObservingAddresses(_ sdk.Context) {
	k.addrs = nil
}

func (k *TestSlashObservingKeeper) ListActiveNodeAccounts(_ sdk.Context) (NodeAccounts, error) {
	return k.nas, nil
}

func (k *TestSlashObservingKeeper) SetNodeAccount(_ sdk.Context, na NodeAccount) error {
	for i := range k.nas {
		if k.nas[i].NodeAddress.Equals(na.NodeAddress) {
			k.nas[i] = na
			return nil
		}
	}
	return fmt.Errorf("Node account not found")
}

func (s *SlashingSuite) TestObservingSlashing(c *C) {
	var err error
	ctx, _ := setupKeeperForTest(c)

	nas := NodeAccounts{
		GetRandomNodeAccount(NodeActive),
		GetRandomNodeAccount(NodeActive),
	}
	keeper := &TestSlashObservingKeeper{
		nas:   nas,
		addrs: []sdk.AccAddress{nas[0].NodeAddress},
	}
	txOutStore := NewTxStoreDummy()
	poolAddrMgr := NewPoolAddressDummyMgr()

	slasher := NewSlasher(keeper, txOutStore, poolAddrMgr)
	// should slash na2 only
	err = slasher.LackObserving(ctx)
	c.Assert(err, IsNil)
	c.Assert(keeper.nas[0].SlashPoints, Equals, int64(0))
	c.Assert(keeper.nas[1].SlashPoints, Equals, int64(constants.LackOfObservationPenalty))

	// since THORNode have cleared all node addresses in slashForObservingAddresses,
	// running it a second time should result in slashing nobody.
	err = slasher.LackObserving(ctx)
	c.Assert(err, IsNil)
	c.Assert(keeper.nas[0].SlashPoints, Equals, int64(0))
	c.Assert(keeper.nas[1].SlashPoints, Equals, int64(constants.LackOfObservationPenalty))
}

type TestSlashingLackKeeper struct {
	KVStoreDummy
	evts  Events
	txOut *TxOut
	na    NodeAccount
}

func (k *TestSlashingLackKeeper) GetIncompleteEvents(_ sdk.Context) (Events, error) {
	return k.evts, nil
}

func (k *TestSlashingLackKeeper) GetTxOut(_ sdk.Context, _ uint64) (*TxOut, error) {
	return k.txOut, nil
}

func (k *TestSlashingLackKeeper) SetTxOut(_ sdk.Context, tx *TxOut) error {
	k.txOut = tx
	return nil
}

func (k *TestSlashingLackKeeper) GetNodeAccountByPubKey(_ sdk.Context, _ common.PubKey) (NodeAccount, error) {
	return k.na, nil
}

func (k *TestSlashingLackKeeper) SetNodeAccount(_ sdk.Context, na NodeAccount) error {
	k.na = na
	return nil
}

func (s *SlashingSuite) TestNotSigningSlash(c *C) {
	ctx, _ := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(201) // set blockheight
	poolAddrMgr := NewPoolAddressDummyMgr()
	txOutStore := NewTxStoreDummy()
	txOutStore.asgard = poolAddrMgr.GetCurrentPoolAddresses().Current

	na := GetRandomNodeAccount(NodeActive)

	swapEvt := NewEventSwap(
		common.BNBAsset,
		sdk.NewUint(5),
		sdk.NewUint(5),
		sdk.NewDec(5),
	)
	swapBytes, _ := json.Marshal(swapEvt)
	evt := NewEvent(
		swapEvt.Type(),
		3,
		common.NewTx(
			GetRandomTxHash(),
			GetRandomBNBAddress(),
			GetRandomBNBAddress(),
			common.Coins{
				common.NewCoin(common.BNBAsset, sdk.NewUint(320000000)),
				common.NewCoin(common.RuneAsset(), sdk.NewUint(420000000)),
			},
			nil,
			"SWAP:BNB.BNB",
		),
		swapBytes,
		EventSuccess,
	)

	txOutItem := &TxOutItem{
		Chain:       common.BNBChain,
		InHash:      evt.InTx.ID,
		VaultPubKey: na.NodePubKey.Secp256k1,
		ToAddress:   GetRandomBNBAddress(),
		Coin: common.NewCoin(
			common.BNBAsset, sdk.NewUint(3980500*common.One),
		),
	}
	txOut := NewTxOut(uint64(evt.Height))
	txOut.TxArray = append(txOut.TxArray, txOutItem)

	keeper := &TestSlashingLackKeeper{
		txOut: txOut,
		evts:  Events{evt},
		na:    na,
	}

	ctx = ctx.WithBlockHeight(evt.Height + constants.SigningTransactionPeriod + 5)

	slasher := NewSlasher(keeper, txOutStore, poolAddrMgr)
	c.Assert(slasher.LackSigning(ctx), IsNil)

	c.Check(keeper.na.SlashPoints, Equals, int64(200), Commentf("%+v\n", na))

	outItems := txOutStore.GetOutboundItems()
	c.Assert(outItems, HasLen, 1)
	poolPubKey := poolAddrMgr.GetAsgardPoolPubKey(evt.InTx.Chain).PubKey
	c.Assert(outItems[0].VaultPubKey.Equals(poolPubKey), Equals, true)
}
