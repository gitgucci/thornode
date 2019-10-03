package swapservice

import (
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"

	"gitlab.com/thorchain/bepswap/common"

	q "gitlab.com/thorchain/bepswap/statechain/x/swapservice/query"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper, poolAddressMgr *PoolAddressManager) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		ctx.Logger().Info("query", "path", path[0])
		switch path[0] {
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
		case q.QueryIncompleteEvents.Key:
			return queryInCompleteEvents(ctx, path[1:], req, keeper)
		case q.QueryCompleteEvents.Key:
			return queryCompleteEvents(ctx, path[1:], req, keeper)
		case q.QueryHeights.Key:
			return queryHeights(ctx, path[1:], req, keeper)
		case q.QueryObservers.Key:
			return queryObservers(ctx, path[1:], req, keeper)
		case q.QueryNodeAccount.Key:
			return queryNodeAccount(ctx, path[1:], req, keeper)
		case q.QueryNodeAccounts.Key:
			return queryNodeAccounts(ctx, path[1:], req, keeper)
		case q.QueryPoolAddresses.Key:
			return queryPoolAddresses(ctx, path[1:], req, keeper, poolAddressMgr)
		default:
			return nil, sdk.ErrUnknownRequest(
				fmt.Sprintf("unknown swapservice query endpoint: %s", path[0]),
			)
		}
	}
}
func queryPoolAddresses(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper, manager *PoolAddressManager) ([]byte, sdk.Error) {
	res, err := codec.MarshalJSONIndent(keeper.cdc, manager.GetCurrentPoolAddresses())
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
	res, err := codec.MarshalJSONIndent(keeper.cdc, nodeAcc)
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
	fmt.Println(nodeAccounts)
	res, err := codec.MarshalJSONIndent(keeper.cdc, nodeAccounts)
	if nil != err {
		ctx.Logger().Error("fail to marshal observers to json", err)
		return nil, sdk.ErrInternal("fail to marshal observers to json")
	}

	return res, nil
}

func queryObservers(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	activeAccounts, err := keeper.ListActiveNodeAccounts(ctx)
	if nil != err {
		return nil, sdk.ErrInternal("fail to get node account iterator")
	}
	var result []string
	for _, item := range activeAccounts {
		result = append(result, item.Accounts.ObserverBEPAddress.String())
	}
	res, err := codec.MarshalJSONIndent(keeper.cdc, result)
	if nil != err {
		ctx.Logger().Error("fail to marshal observers to json", err)
		return nil, sdk.ErrInternal("fail to marshal observers to json")
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
	res, err := codec.MarshalJSONIndent(keeper.cdc, ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal pool index to json", err)
		return nil, sdk.ErrInternal("fail to marshal pool index to json")
	}
	return res, nil
}

// queryPoolStakers
func queryPoolStakers(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	ticker, err := common.NewTicker(path[0])
	if nil != err {
		ctx.Logger().Error("fail to get parse ticker", err)
		return nil, sdk.ErrInternal("fail to parse ticker")
	}
	ps, err := keeper.GetPoolStaker(ctx, ticker)
	if nil != err {
		ctx.Logger().Error("fail to get pool staker", err)
		return nil, sdk.ErrInternal("fail to get pool staker")
	}
	res, err := codec.MarshalJSONIndent(keeper.cdc, ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal pool staker to json", err)
		return nil, sdk.ErrInternal("fail to marshal pool staker to json")
	}
	return res, nil
}

// queryStakerPool
func queryStakerPool(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	addr, err := common.NewBnbAddress(path[0])
	if nil != err {
		ctx.Logger().Error("fail to parse bnb address", err)
		return nil, sdk.ErrInternal("fail to parse bnb address")
	}

	ps, err := keeper.GetStakerPool(ctx, addr)
	if nil != err {
		ctx.Logger().Error("fail to get staker pool", err)
		return nil, sdk.ErrInternal("fail to get staker pool")
	}
	res, err := codec.MarshalJSONIndent(keeper.cdc, ps)
	if nil != err {
		ctx.Logger().Error("fail to marshal staker pool to json", err)
		return nil, sdk.ErrInternal("fail to marshal staker pool to json")
	}
	return res, nil
}

// nolint: unparam
func queryPool(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper, poolAddrMgr *PoolAddressManager) ([]byte, sdk.Error) {
	ticker, err := common.NewTicker(path[0])
	if err != nil {
		ctx.Logger().Error("fail to parse ticker", err)
		return nil, sdk.ErrInternal("Could not parse ticker")
	}
	currentPoolAddr := poolAddrMgr.GetCurrentPoolAddresses()
	pool := keeper.GetPool(ctx, ticker)
	if pool.Empty() {
		return nil, sdk.ErrUnknownRequest(fmt.Sprintf("pool: %s doesn't exist", path[0]))
	}
	pool.PoolAddress = currentPoolAddr.Current
	pool.ExpiryInBlockHeight = currentPoolAddr.RotateAt - req.Height
	res, err := codec.MarshalJSONIndent(keeper.cdc, pool)
	if err != nil {
		return nil, sdk.ErrInternal("could not marshal result to JSON")
	}
	return res, nil
}

func queryPools(ctx sdk.Context, req abci.RequestQuery, keeper Keeper, poolAddrMgr *PoolAddressManager) ([]byte, sdk.Error) {
	pools := QueryResPools{}
	iterator := keeper.GetPoolDataIterator(ctx)
	currentPoolAddr := poolAddrMgr.GetCurrentPoolAddresses()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		keeper.cdc.MustUnmarshalBinaryBare(iterator.Value(), &pool)
		pool.PoolAddress = currentPoolAddr.Current
		pool.ExpiryInBlockHeight = currentPoolAddr.RotateAt - req.Height
		pools = append(pools, pool)
	}
	res, err := codec.MarshalJSONIndent(keeper.cdc, pools)
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
	res, err := codec.MarshalJSONIndent(keeper.cdc, voter.GetTx(trustAccounts))
	if nil != err {
		ctx.Logger().Error("fail to marshal tx hash to json", err)
		return nil, sdk.ErrInternal("fail to marshal tx hash to json")
	}
	return res, nil
}

func queryTxOutArray(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	height, err := strconv.ParseUint(path[0], 0, 64)
	if nil != err {
		ctx.Logger().Error("fail to parse block height", err)
		return nil, sdk.ErrInternal("fail to parse block height")
	}
	tx, err := keeper.GetTxOut(ctx, height)
	if nil != err {
		ctx.Logger().Error("fail to get tx out array from key value store", err)
		return nil, sdk.ErrInternal("fail to get tx out array from key value store")
	}
	res, err := codec.MarshalJSONIndent(keeper.cdc, tx)
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
	res, err := codec.MarshalJSONIndent(keeper.cdc, config)
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
	res, err := codec.MarshalJSONIndent(keeper.cdc, events)
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

	res, err := codec.MarshalJSONIndent(keeper.cdc, events)
	if nil != err {
		ctx.Logger().Error("fail to marshal events to json", err)
		return nil, sdk.ErrInternal("fail to marshal events to json")
	}
	return res, nil
}

func queryHeights(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	binance := keeper.GetLastBinanceHeight(ctx)
	signed := keeper.GetLastSignedHeight(ctx)

	res, err := codec.MarshalJSONIndent(keeper.cdc, QueryResHeights{
		LastBinanceHeight: binance,
		LastSignedHeight:  signed,
		Statechain:        ctx.BlockHeight(),
	})
	if nil != err {
		ctx.Logger().Error("fail to marshal events to json", err)
		return nil, sdk.ErrInternal("fail to marshal events to json")
	}
	return res, nil
}
