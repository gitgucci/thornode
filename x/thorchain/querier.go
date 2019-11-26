package thorchain

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"gitlab.com/thorchain/thornode/common"

	q "gitlab.com/thorchain/thornode/x/thorchain/query"
	"gitlab.com/thorchain/thornode/x/thorchain/types"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper, poolAddressMgr *PoolAddressManager, validatorMgr *ValidatorManager) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		ctx.Logger().Info("query", "path", path[0])
		switch path[0] {
		case q.QueryChains.Key:
			return queryChains(ctx, req, keeper)
		case q.QueryPool.Key:
			return queryPool(ctx, path[1:], req, keeper, poolAddressMgr)
		case q.QueryPools.Key:
			return queryPools(ctx, req, keeper, poolAddressMgr)
		case q.QueryPoolStakers.Key:
			return queryPoolStakers(ctx, path[1:], req, keeper)
		case q.QueryStakerPools.Key:
			return queryStakerPool(ctx, path[1:], req, keeper)
		case q.QueryPoolIndex.Key:
			return queryPoolIndex(ctx, path[1:], req, keeper)
		case q.QueryTxIn.Key:
			return queryTxIn(ctx, path[1:], req, keeper)
		case q.QueryAdminConfig.Key, q.QueryAdminConfigBnb.Key:
			return queryAdminConfig(ctx, path[1:], req, keeper)
		case q.QueryTxOutArray.Key:
			return queryTxOutArray(ctx, path[1:], req, keeper)
		case q.QueryTxOutArrayPubkey.Key:
			return queryTxOutArray(ctx, path[1:], req, keeper)
		case q.QueryIncompleteEvents.Key:
			return queryInCompleteEvents(ctx, path[1:], req, keeper)
		case q.QueryCompleteEvents.Key:
			return queryCompleteEvents(ctx, path[1:], req, keeper)
		case q.QueryHeights.Key:
			return queryHeights(ctx, path[1:], req, keeper)
		case q.QueryChainHeights.Key:
			return queryHeights(ctx, path[1:], req, keeper)
		case q.QueryObservers.Key:
			return queryObservers(ctx, path[1:], req, keeper)
		case q.QueryObserver.Key:
			return queryObserver(ctx, path[1:], req, keeper)
		case q.QueryNodeAccount.Key:
			return queryNodeAccount(ctx, path[1:], req, keeper)
		case q.QueryNodeAccounts.Key:
			return queryNodeAccounts(ctx, path[1:], req, keeper)
		case q.QueryPoolAddresses.Key:
			return queryPoolAddresses(ctx, path[1:], req, keeper, poolAddressMgr)
		case q.QueryValidators.Key:
			return queryValidators(ctx, keeper, validatorMgr)
		default:
			return nil, sdk.ErrUnknownRequest(
				fmt.Sprintf("unknown thorchain query endpoint: %s", path[0]),
			)
		}
	}
}

func queryValidators(ctx sdk.Context, keeper Keeper, validatorMgr *ValidatorManager) ([]byte, sdk.Error) {
	activeAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if nil != err {
		ctx.Logger().Error("fail to get all active node accounts", err)
		return nil, sdk.ErrInternal("fail to get all active accounts")
	}

	resp := types.ValidatorsResp{
		ActiveNodes: activeAccounts,
	}
	if validatorMgr.Meta != nil {
		resp.RotateAt = uint64(validatorMgr.Meta.RotateAtBlockHeight)
		resp.RotateWindowOpenAt = uint64(validatorMgr.Meta.RotateWindowOpenAtBlockHeight)
		if !validatorMgr.Meta.Nominated.IsEmpty() {
			resp.Nominated = validatorMgr.Meta.Nominated
		}
		if !validatorMgr.Meta.Queued.IsEmpty() {
			resp.Queued = validatorMgr.Meta.Queued
		}
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), resp)
	if nil != err {
		ctx.Logger().Error("fail to marshal validator response to json", err)
		return nil, sdk.ErrInternal("fail to marshal validator response to json")
	}
	return res, nil
}

func queryChains(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	chains := keeper.GetChains(ctx)
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), chains)
	if nil != err {
		ctx.Logger().Error("fail to marshal current chains to json", err)
		return nil, sdk.ErrInternal("fail to marshal chains to json")
	}

	return res, nil
}

func queryPoolAddresses(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper, manager *PoolAddressManager) ([]byte, sdk.Error) {
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), manager.GetCurrentPoolAddresses())
	if nil != err {
		ctx.Logger().Error("fail to marshal current pool address to json", err)
		return nil, sdk.ErrInternal("fail to marshal current pool address to json")
	}

	return res, nil
}

func queryNodeAccount(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	nodeAddress := path[0]
	addr, err := sdk.AccAddressFromBech32(nodeAddress)
	if nil != err {
		return nil, sdk.ErrUnknownRequest("invalid account address")
	}

	nodeAcc, err := keeper.GetNodeAccount(ctx, addr)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get node accounts")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), nodeAcc)
	if nil != err {
		ctx.Logger().Error("fail to marshal node account to json", err)
		return nil, sdk.ErrInternal("fail to marshal node account to json")
	}

	return res, nil
}
func queryNodeAccounts(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	nodeAccounts, err := keeper.ListNodeAccounts(ctx)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get node accounts")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), nodeAccounts)
	if nil != err {
		ctx.Logger().Error("fail to marshal observers to json", err)
		return nil, sdk.ErrInternal("fail to marshal observers to json")
	}

	return res, nil
}

// queryObservers will only return all the active accounts
func queryObservers(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	activeAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get node account iterator")
	}
	var result []string
	for _, item := range activeAccounts {
		result = append(result, item.NodeAddress.String())
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), result)
	if nil != err {
		ctx.Logger().Error("fail to marshal observers to json", err)
		return nil, sdk.ErrInternal("fail to marshal observers to json")
	}

	return res, nil
}

func queryObserver(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	observerAddr := path[0]
	addr, err := sdk.AccAddressFromBech32(observerAddr)
	if nil != err {
		return nil, sdk.ErrUnknownRequest("invalid account address")
	}

	nodeAcc, err := keeper.GetNodeAccount(ctx, addr)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get node account")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), nodeAcc)
	if nil != err {
		ctx.Logger().Error("fail to marshal node account to json", err)
		return nil, sdk.ErrInternal("fail to marshal node account to json")
	}

	return res, nil
}

// queryPoolIndex
func queryPoolIndex(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	ps, err := keeper.GetPoolIndex(ctx)
	if nil != err {
		ctx.Logger().Error("fail to get pool index", err)
		return nil, sdk.ErrInternal("fail to get pool index")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal pool index to json", err)
		return nil, sdk.ErrInternal("fail to marshal pool index to json")
	}
	return res, nil
}

// queryPoolStakers
func queryPoolStakers(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	asset, err := common.NewAsset(path[0])
	if nil != err {
		ctx.Logger().Error("fail to get parse asset", err)
		return nil, sdk.ErrInternal("fail to parse asset")
	}
	ps, err := keeper.GetPoolStaker(ctx, asset)
	if nil != err {
		ctx.Logger().Error("fail to get pool staker", err)
		return nil, sdk.ErrInternal("fail to get pool staker")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal pool staker to json", err)
		return nil, sdk.ErrInternal("fail to marshal pool staker to json")
	}
	return res, nil
}

// queryStakerPool
func queryStakerPool(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	addr, err := common.NewAddress(path[0])
	if nil != err {
		ctx.Logger().Error("fail to parse bnb address", err)
		return nil, sdk.ErrInternal("fail to parse bnb address")
	}

	ps, err := keeper.GetStakerPool(ctx, addr)
	if nil != err {
		ctx.Logger().Error("fail to get staker pool", err)
		return nil, sdk.ErrInternal("fail to get staker pool")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal staker pool to json", err)
		return nil, sdk.ErrInternal("fail to marshal staker pool to json")
	}
	return res, nil
}

// nolint: unparam
func queryPool(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper, poolAddrMgr *PoolAddressManager) ([]byte, sdk.Error) {
	asset, err := common.NewAsset(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse asset", err)
		return nil, sdk.ErrInternal("Could not parse asset")
	}
	currentPoolAddr := poolAddrMgr.GetCurrentPoolAddresses()
	pool := keeper.GetPool(ctx, asset)
	if pool.Empty() {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("pool: %s doesn't exist", path[0]))
	}
	bnbPoolPubKey := currentPoolAddr.Current.GetByChain(common.BNBChain)
	if bnbPoolPubKey == nil || bnbPoolPubKey.IsEmpty() {
		return nil, sdk.ErrInternal("fail to get current address")
	}
	addr, err := bnbPoolPubKey.GetAddress()
	if nil != err {
		return nil, sdk.ErrInternal("fail to get bnb chain pool address")
	}
	pool.PoolAddress = addr
	pool.ExpiryInBlockHeight = currentPoolAddr.RotateAt - req.Height
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), pool)
	if err != nil {
		return nil, sdk.ErrInternal("could not marshal result to JSON")
	}
	return res, nil
}

func queryPools(ctx sdk.Context, req abci.RequestQuery, keeper Keeper, poolAddrMgr *PoolAddressManager) ([]byte, sdk.Error) {
	pools := QueryResPools{}
	iterator := keeper.GetPoolDataIterator(ctx)
	currentPoolAddr := poolAddrMgr.GetCurrentPoolAddresses()
	bnbPoolPubKey := currentPoolAddr.Current.GetByChain(common.BNBChain)
	if bnbPoolPubKey == nil || bnbPoolPubKey.IsEmpty() {
		return nil, sdk.ErrInternal("fail to get current address")
	}
	addr, err := bnbPoolPubKey.GetAddress()
	if nil != err {
		return nil, sdk.ErrInternal("fail to get bnb chain pool address")
	}
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		keeper.Cdc().MustUnmarshalBinaryBare(iterator.Value(), &pool)
		pool.PoolAddress = addr
		pool.ExpiryInBlockHeight = currentPoolAddr.RotateAt - req.Height
		pools = append(pools, pool)
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), pools)
	if err != nil {
		return nil, sdk.ErrInternal("could not marshal pools result to json")
	}
	return res, nil
}

func queryTxIn(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	hash, err := common.NewTxID(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse tx id", err)
		return nil, sdk.ErrInternal("fail to parse tx id")
	}
	voter := keeper.GetTxInVoter(ctx, hash)
	trustAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get trust account")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), voter.GetTx(trustAccounts))
	if nil != err {
		ctx.Logger().Error("fail to marshal tx hash to json", err)
		return nil, sdk.ErrInternal("fail to marshal tx hash to json")
	}
	return res, nil
}

func queryTxOutArray(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var err error
	height, err := strconv.ParseUint(path[0], 0, 64)
	if nil != err {
		ctx.Logger().Error("fail to parse block height", err)
		return nil, sdk.ErrInternal("fail to parse block height")
	}

	pk := common.EmptyPubKey
	if len(path) > 1 {
		pk, err = common.NewPubKey(path[1])
		if nil != err {
			ctx.Logger().Error("fail to parse pubkey", err)
			return nil, sdk.ErrInternal("fail to parse pubkey")
		}
	}

	txs, err := keeper.GetTxOut(ctx, height)
	if nil != err {
		ctx.Logger().Error("fail to get tx out array from key value store", err)
		return nil, sdk.ErrInternal("fail to get tx out array from key value store")
	}

	if !pk.IsEmpty() {
		newTxs := &TxOut{
			Height: txs.Height,
		}
		for _, tx := range txs.TxArray {
			if pk.Equals(tx.PoolAddress) {
				newTxs.TxArray = append(newTxs.TxArray, tx)
			}
		}
		txs = newTxs
	}

	out := make(map[common.Chain]ResTxOut, 0)
	for _, item := range txs.TxArray {
		if item.Coin.IsEmpty() {
			continue
		}
		res, ok := out[item.Chain]
		if !ok {
			res = ResTxOut{
				Height:  txs.Height,
				Chain:   item.Coin.Asset.Chain,
				TxArray: make([]TxOutItem, 0),
			}
		}
		res.TxArray = append(res.TxArray, *item)
		out[item.Chain] = res
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), QueryResTxOut{
		Chains: out,
	})
	if nil != err {
		ctx.Logger().Error("fail to marshal tx hash to json", err)
		return nil, sdk.ErrInternal("fail to marshal tx hash to json")
	}
	return res, nil
}

func queryAdminConfig(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var err error
	key := GetAdminConfigKey(path[0])
	addr := EmptyAccAddress
	if len(path) > 1 {
		addr, err = sdk.AccAddressFromBech32(path[1])
		if err != nil {
			ctx.Logger().Error("fail to parse bep address", err)
			return nil, sdk.ErrInternal("fail to parse bep address")
		}
	}
	config := NewAdminConfig(key, "", addr)
	config.Value, err = keeper.GetAdminConfigValue(ctx, key, addr)
	if nil != err {
		ctx.Logger().Error("fail to get admin config", err)
		return nil, sdk.ErrInternal("fail to get admin config")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), config)
	if nil != err {
		ctx.Logger().Error("fail to marshal config to json", err)
		return nil, sdk.ErrInternal("fail to marshal config to json")
	}
	return res, nil
}

func queryInCompleteEvents(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	events, err := keeper.GetIncompleteEvents(ctx)
	if nil != err {
		ctx.Logger().Error("fail to get incomplete events", err)
		return nil, sdk.ErrInternal("fail to get incomplete events")
	}
	res, err := codec.MarshalJSONIndent(keeper.Cdc(), events)
	if nil != err {
		ctx.Logger().Error("fail to marshal events to json", err)
		return nil, sdk.ErrInternal("fail to marshal events to json")
	}
	return res, nil
}

func queryCompleteEvents(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	id, err := strconv.ParseInt(path[0], 10, 64)
	if err != nil {
		ctx.Logger().Error("fail to discover id number", err)
		return nil, sdk.ErrInternal("fail to discover id number")
	}

	limit := int64(100) // limit the number of events, aka pagination
	events := make(Events, 0)
	for i := id; i <= id+limit; i++ {
		event, _ := keeper.GetCompletedEvent(ctx, i)
		if !event.Empty() {
			events = append(events, event)
		} else {
			break
		}
	}

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), events)
	if nil != err {
		ctx.Logger().Error("fail to marshal events to json", err)
		return nil, sdk.ErrInternal("fail to marshal events to json")
	}
	return res, nil
}

func queryHeights(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var chain common.Chain
	if path[0] == "" {
		chain = common.BNBChain
	} else {
		var err error
		chain, err = common.NewChain(path[0])
		if err != nil {
			ctx.Logger().Error("fail to retrieve chain", err)
			return nil, sdk.ErrInternal("fail to retrieve chain")
		}
	}
	chainHeight := keeper.GetLastChainHeight(ctx, chain)
	signed := keeper.GetLastSignedHeight(ctx)

	res, err := codec.MarshalJSONIndent(keeper.Cdc(), QueryResHeights{
		Chain:            chain,
		LastChainHeight:  chainHeight,
		LastSignedHeight: signed,
		Statechain:       ctx.BlockHeight(),
	})
	if nil != err {
		ctx.Logger().Error("fail to marshal events to json", err)
		return nil, sdk.ErrInternal("fail to marshal events to json")
	}
	return res, nil
}
