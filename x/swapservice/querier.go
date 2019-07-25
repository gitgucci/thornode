package swapservice

import (
	"log"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"

	sdk "github.com/cosmos/cosmos-sdk/types"
	abci "github.com/tendermint/tendermint/abci/types"
)

// query endpoints supported by the swapservice Querier
const (
	QueryPoolStruct = "poolstruct"
	QueryPoolDatas  = "pooldatas"
	QueryStakeDatas = "stakedatas"
)

// NewQuerier is the module level router for state queries
func NewQuerier(keeper Keeper) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) (res []byte, err sdk.Error) {
		switch path[0] {
		case QueryPoolStruct:
			return queryPoolStruct(ctx, path[1:], req, keeper)
		case QueryPoolDatas:
			return queryPoolDatas(ctx, req, keeper)
		case QueryStakeDatas:
			return queryStakeDatas(ctx, req, keeper)
		default:
			return nil, sdk.ErrUnknownRequest("unknown swapservice query endpoint")
		}
	}
}

// nolint: unparam
func queryPoolStruct(ctx sdk.Context, path []string, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	poolstruct := keeper.GetPoolStruct(ctx, path[0])

	res, err := codec.MarshalJSONIndent(keeper.cdc, poolstruct)
	if err != nil {
		panic("could not marshal result to JSON")
	}

	return res, nil
}
func queryPoolDatas(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var pooldatasList QueryResPoolDatas

	iterator := keeper.GetDatasIterator(ctx)

	for ; iterator.Valid(); iterator.Next() {
		key := string(iterator.Key())
		if strings.HasPrefix(key, "pool-") {
			poolstruct := keeper.GetPoolStruct(ctx, key)
			pooldatasList = append(pooldatasList, poolstruct)
		}
	}
	log.Printf("Pools: %+v", pooldatasList)

	res, err := codec.MarshalJSONIndent(keeper.cdc, pooldatasList)
	if err != nil {
		panic("could not marshal result to JSON")
	}

	return res, nil
}
func queryStakeDatas(ctx sdk.Context, req abci.RequestQuery, keeper Keeper) ([]byte, sdk.Error) {
	var accdatasList QueryResStakeDatas

	iterator := keeper.GetDatasIterator(ctx)

	for ; iterator.Valid(); iterator.Next() {
		accdatasList = append(accdatasList, string(iterator.Key()))
	}

	res, err := codec.MarshalJSONIndent(keeper.cdc, accdatasList)
	if err != nil {
		panic("could not marshal result to JSON")
	}

	return res, nil
}
