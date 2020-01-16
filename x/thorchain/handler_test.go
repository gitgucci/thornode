package thorchain

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/supply"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	dbm "github.com/tendermint/tm-db"
	. "gopkg.in/check.v1"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/constants"

	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

type HandlerSuite struct{}

var _ = Suite(&HandlerSuite{})

func (s *HandlerSuite) SetUpSuite(*C) {
	SetupConfigForTest()
}

// nolint: deadcode unused
// create a codec used only for testing
func makeTestCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	RegisterCodec(cdc)
	supply.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}

var (
	multiPerm    = "multiple permissions account"
	randomPerm   = "random permission"
	holder       = "holder"
	keyThorchain = sdk.NewKVStoreKey(StoreKey)
)

func setupKeeperForTest(c *C) (sdk.Context, Keeper) {
	keyAcc := sdk.NewKVStoreKey(auth.StoreKey)
	keyParams := sdk.NewKVStoreKey(params.StoreKey)
	tkeyParams := sdk.NewTransientStoreKey(params.TStoreKey)
	keySupply := sdk.NewKVStoreKey(supply.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySupply, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyThorchain, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tkeyParams, sdk.StoreTypeTransient, db)
	err := ms.LoadLatestVersion()
	c.Assert(err, IsNil)

	ctx := sdk.NewContext(ms, abci.Header{ChainID: "thorchain"}, false, log.NewNopLogger())
	cdc := makeTestCodec()

	pk := params.NewKeeper(cdc, keyParams, tkeyParams, params.DefaultCodespace)
	ak := auth.NewAccountKeeper(cdc, keyAcc, pk.Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount)
	bk := bank.NewBaseKeeper(ak, pk.Subspace(bank.DefaultParamspace), bank.DefaultCodespace, nil)

	maccPerms := map[string][]string{
		auth.FeeCollectorName: nil,
		holder:                nil,
		supply.Minter:         {supply.Minter},
		supply.Burner:         {supply.Burner},
		multiPerm:             {supply.Minter, supply.Burner, supply.Staking},
		randomPerm:            {"random"},
		ModuleName:            {supply.Minter},
	}
	supplyKeeper := supply.NewKeeper(cdc, keySupply, ak, bk, maccPerms)
	totalSupply := sdk.NewCoins(sdk.NewCoin("bep", sdk.NewInt(1000*common.One)))
	supplyKeeper.SetSupply(ctx, supply.NewSupply(totalSupply))
	k := NewKVStore(bk, supplyKeeper, keyThorchain, cdc)
	return ctx, k
}

type handlerTestWrapper struct {
	ctx                  sdk.Context
	keeper               Keeper
	validatorMgr         VersionedValidatorManager
	versionedTxOutStore  VersionedTxOutStore
	activeNodeAccount    NodeAccount
	notActiveNodeAccount NodeAccount
}

func getHandlerTestWrapper(c *C, height int64, withActiveNode, withActieBNBPool bool) handlerTestWrapper {
	ctx, k := setupKeeperForTest(c)
	ctx = ctx.WithBlockHeight(height)
	acc1 := GetRandomNodeAccount(NodeActive)
	if withActiveNode {
		c.Assert(k.SetNodeAccount(ctx, acc1), IsNil)
	}
	if withActieBNBPool {
		p, err := k.GetPool(ctx, common.BNBAsset)
		c.Assert(err, IsNil)
		p.Asset = common.BNBAsset
		p.Status = PoolEnabled
		p.BalanceRune = sdk.NewUint(100 * common.One)
		p.BalanceAsset = sdk.NewUint(100 * common.One)
		c.Assert(k.SetPool(ctx, p), IsNil)
	}
	ver := semver.MustParse("0.1.0")
	constAccessor := constants.GetConstantValues(ver)
	vaultMgr := NewVaultMgrDummy()
	versionedTxOutStore := NewVersionedTxOutStore()
	txOutStore, err := versionedTxOutStore.GetTxOutStore(k, ver)
	c.Assert(err, IsNil)

	txOutStore.NewBlock(uint64(height), constAccessor)
	validatorMgr := NewVersionedValidatorMgr(k, versionedTxOutStore, vaultMgr)
	c.Assert(validatorMgr.BeginBlock(ctx, ver, constAccessor), IsNil)

	return handlerTestWrapper{
		ctx:                  ctx,
		keeper:               k,
		validatorMgr:         validatorMgr,
		versionedTxOutStore:  versionedTxOutStore,
		activeNodeAccount:    acc1,
		notActiveNodeAccount: GetRandomNodeAccount(NodeDisabled),
	}
}

func (HandlerSuite) TestIsSignedByActiveObserver(c *C) {
	ctx, k := setupKeeperForTest(c)
	nodeAddr := GetRandomBech32Addr()
	c.Check(isSignedByActiveObserver(ctx, k, []sdk.AccAddress{nodeAddr}), Equals, false)
	c.Check(isSignedByActiveObserver(ctx, k, []sdk.AccAddress{}), Equals, false)
}

func (HandlerSuite) TestIsSignedByActiveNodeAccounts(c *C) {
	ctx, k := setupKeeperForTest(c)
	nodeAddr := GetRandomBech32Addr()
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []sdk.AccAddress{}), Equals, false)
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []sdk.AccAddress{nodeAddr}), Equals, false)
	nodeAccount1 := GetRandomNodeAccount(NodeWhiteListed)
	c.Assert(k.SetNodeAccount(ctx, nodeAccount1), IsNil)
	c.Check(isSignedByActiveNodeAccounts(ctx, k, []sdk.AccAddress{nodeAccount1.NodeAddress}), Equals, false)
}

func (HandlerSuite) TestHandleTxInCreateMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	vault := GetRandomVault()
	w.keeper.SetVault(w.ctx, vault)
	addr, err := vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	txIn := types.NewObservedTx(
		common.Tx{
			ID:          GetRandomTxHash(),
			Chain:       common.BNBChain,
			Coins:       common.Coins{common.NewCoin(common.RuneAsset(), sdk.NewUint(1*common.One))},
			Memo:        "create:BNB",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   addr,
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		vault.PubKey,
	)

	msg := types.NewMsgObservedTxIn(
		ObservedTxs{
			txIn,
		},
		w.activeNodeAccount.NodeAddress,
	)

	vaultMgr := NewVaultMgrDummy()
	handler := NewHandler(w.keeper, w.versionedTxOutStore, w.validatorMgr, vaultMgr)
	result := handler(w.ctx, msg)
	c.Assert(result.Code, Equals, sdk.CodeOK, Commentf("%s\n", result.Log))

	pool, err := w.keeper.GetPool(w.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.Empty(), Equals, false)
	c.Assert(pool.Status, Equals, PoolEnabled)
	c.Assert(pool.PoolUnits.Uint64(), Equals, uint64(0))
	c.Assert(pool.BalanceRune.Uint64(), Equals, uint64(0))
	c.Assert(pool.BalanceAsset.Uint64(), Equals, uint64(0))
}

func (HandlerSuite) TestHandleTxInWithdrawMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	vault := GetRandomVault()
	w.keeper.SetVault(w.ctx, vault)
	addr, err := vault.PubKey.GetAddress(common.BNBChain)
	c.Assert(err, IsNil)

	staker := GetRandomBNBAddress()
	// lets do a stake first, otherwise nothing to withdraw
	txStake := types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, sdk.NewUint(100*common.One)),
				common.NewCoin(common.RuneAsset(), sdk.NewUint(100*common.One)),
			},
			Memo:        "stake:BNB",
			FromAddress: staker,
			ToAddress:   addr,
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		vault.PubKey,
	)

	msg := types.NewMsgObservedTxIn(
		ObservedTxs{
			txStake,
		},
		w.activeNodeAccount.NodeAddress,
	)

	vaultMgr := NewVaultMgrDummy()
	handler := NewHandler(w.keeper, w.versionedTxOutStore, w.validatorMgr, vaultMgr)
	result := handler(w.ctx, msg)
	c.Assert(result.Code, Equals, sdk.CodeOK, Commentf("%s\n", result.Log))

	txStake = types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.RuneAsset(), sdk.NewUint(1*common.One)),
			},
			Memo:        "withdraw:BNB",
			FromAddress: staker,
			ToAddress:   addr,
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		vault.PubKey,
	)

	msg = types.NewMsgObservedTxIn(
		ObservedTxs{
			txStake,
		},
		w.activeNodeAccount.NodeAddress,
	)
	ver := semver.MustParse("0.1.0")
	constAccessor := constants.GetConstantValues(ver)
	txOutStore, err := w.versionedTxOutStore.GetTxOutStore(w.keeper, ver)
	c.Assert(err, IsNil)
	txOutStore.NewBlock(2, constAccessor)
	result = handler(w.ctx, msg)
	c.Assert(result.Code, Equals, sdk.CodeOK, Commentf("%s\n", result.Log))

	pool, err := w.keeper.GetPool(w.ctx, common.BNBAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.Empty(), Equals, false)
	c.Assert(pool.Status, Equals, PoolBootstrap)
	c.Assert(pool.PoolUnits.Uint64(), Equals, uint64(0))
	c.Assert(pool.BalanceRune.Uint64(), Equals, uint64(0))
	c.Assert(pool.BalanceAsset.Uint64(), Equals, uint64(0))

}

func (HandlerSuite) TestRefund(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)

	pool := Pool{
		Asset:        common.BNBAsset,
		BalanceRune:  sdk.NewUint(100 * common.One),
		BalanceAsset: sdk.NewUint(100 * common.One),
	}
	c.Assert(w.keeper.SetPool(w.ctx, pool), IsNil)

	txin := NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(common.BNBAsset, sdk.NewUint(100*common.One)),
			},
			Memo:        "withdraw:BNB",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		GetRandomPubKey(),
	)
	ver := semver.MustParse("0.1.0")
	txOutStore, err := w.versionedTxOutStore.GetTxOutStore(w.keeper, ver)
	c.Assert(err, IsNil)
	c.Assert(refundTx(w.ctx, txin, txOutStore, w.keeper, sdk.CodeInternal, "refund"), IsNil)
	c.Assert(txOutStore.GetOutboundItems(), HasLen, 1)

	// check THORNode DONT create a refund transaction when THORNode don't have a pool for
	// the asset sent.
	lokiAsset, _ := common.NewAsset(fmt.Sprintf("BNB.LOKI"))
	txin.Tx.Coins = common.Coins{
		common.NewCoin(lokiAsset, sdk.NewUint(100*common.One)),
	}

	c.Assert(refundTx(w.ctx, txin, txOutStore, w.keeper, sdk.CodeInternal, "refund"), IsNil)
	c.Assert(txOutStore.GetOutboundItems(), HasLen, 1)

	pool, err = w.keeper.GetPool(w.ctx, lokiAsset)
	c.Assert(err, IsNil)
	// pool should be zero since we drop coins we don't recognize on the floor
	c.Assert(pool.BalanceAsset.Equal(sdk.ZeroUint()), Equals, true, Commentf("%d", pool.BalanceAsset.Uint64()))

	// doing it a second time should keep it at zero
	c.Assert(refundTx(w.ctx, txin, txOutStore, w.keeper, sdk.CodeInternal, "refund"), IsNil)
	c.Assert(txOutStore.GetOutboundItems(), HasLen, 1)
	pool, err = w.keeper.GetPool(w.ctx, lokiAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceAsset.Equal(sdk.ZeroUint()), Equals, true)
}

func (HandlerSuite) TestGetMsgSwapFromMemo(c *C) {
	m, err := ParseMemo("swap:BNB")
	swapMemo, ok := m.(SwapMemo)
	c.Assert(ok, Equals, true)
	c.Assert(err, IsNil)

	txin := types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(
					common.BNBAsset,
					sdk.NewUint(100*common.One),
				),
				common.NewCoin(
					common.RuneAsset(),
					sdk.NewUint(100*common.One),
				),
			},
			Memo:        "withdraw:BNB",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		common.EmptyPubKey,
	)

	// more than one coin
	resultMsg, err := getMsgSwapFromMemo(swapMemo, txin, GetRandomBech32Addr())
	c.Assert(err, NotNil)
	c.Assert(resultMsg, IsNil)

	txin.Tx.Coins = common.Coins{
		common.NewCoin(
			common.BNBAsset,
			sdk.NewUint(100*common.One),
		),
	}

	// coin and the ticker is the same, thus no point to swap
	resultMsg1, err := getMsgSwapFromMemo(swapMemo, txin, GetRandomBech32Addr())
	c.Assert(resultMsg1, IsNil)
	c.Assert(err, NotNil)
}

func (HandlerSuite) TestGetMsgStakeFromMemo(c *C) {
	w := getHandlerTestWrapper(c, 1, true, false)
	// Stake BNB, however THORNode send T-CAN as coin , which is incorrect, should result in an error
	m, err := ParseMemo("stake:BNB")
	c.Assert(err, IsNil)
	stakeMemo, ok := m.(StakeMemo)
	c.Assert(ok, Equals, true)
	tcanAsset, err := common.NewAsset("BNB.TCAN-014")
	c.Assert(err, IsNil)
	runeAsset := common.RuneAsset()
	c.Assert(err, IsNil)

	txin := types.NewObservedTx(
		common.Tx{
			ID:    GetRandomTxHash(),
			Chain: common.BNBChain,
			Coins: common.Coins{
				common.NewCoin(tcanAsset,
					sdk.NewUint(100*common.One)),
				common.NewCoin(runeAsset,
					sdk.NewUint(100*common.One)),
			},
			Memo:        "withdraw:BNB",
			FromAddress: GetRandomBNBAddress(),
			ToAddress:   GetRandomBNBAddress(),
			Gas:         common.BNBGasFeeSingleton,
		},
		1024,
		common.EmptyPubKey,
	)

	msg, err := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg, IsNil)
	c.Assert(err, NotNil)

	// Asymentic stake should works fine, only RUNE
	txin.Tx.Coins = common.Coins{
		common.NewCoin(runeAsset,
			sdk.NewUint(100*common.One)),
	}

	// stake only rune should be fine
	msg1, err1 := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg1, NotNil)
	c.Assert(err1, IsNil)

	bnbAsset, err := common.NewAsset("BNB.BNB")
	c.Assert(err, IsNil)
	txin.Tx.Coins = common.Coins{
		common.NewCoin(bnbAsset,
			sdk.NewUint(100*common.One)),
	}

	// stake only token(BNB) should be fine
	msg2, err2 := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg2, NotNil)
	c.Assert(err2, IsNil)

	lokiAsset, _ := common.NewAsset(fmt.Sprintf("BNB.LOKI"))
	txin.Tx.Coins = common.Coins{
		common.NewCoin(tcanAsset,
			sdk.NewUint(100*common.One)),
		common.NewCoin(lokiAsset,
			sdk.NewUint(100*common.One)),
	}

	// stake only token should be fine
	msg3, err3 := getMsgStakeFromMemo(w.ctx, stakeMemo, txin, GetRandomBech32Addr())
	c.Assert(msg3, IsNil)
	c.Assert(err3, NotNil)

	// Make sure the RUNE Address and Asset Address set correctly
	txin.Tx.Coins = common.Coins{
		common.NewCoin(runeAsset,
			sdk.NewUint(100*common.One)),
		common.NewCoin(lokiAsset,
			sdk.NewUint(100*common.One)),
	}

	lokiStakeMemo, err := ParseMemo("stake:BNB.LOKI")
	c.Assert(err, IsNil)
	msg4, err4 := getMsgStakeFromMemo(w.ctx, lokiStakeMemo.(StakeMemo), txin, GetRandomBech32Addr())
	c.Assert(err4, IsNil)
	c.Assert(msg4, NotNil)
	msgStake := msg4.(MsgSetStakeData)
	c.Assert(msgStake, NotNil)
	c.Assert(msgStake.RuneAddress, Equals, txin.Tx.FromAddress)
	c.Assert(msgStake.AssetAddress, Equals, txin.Tx.FromAddress)
}
