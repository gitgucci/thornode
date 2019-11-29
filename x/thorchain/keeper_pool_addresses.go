package thorchain

import sdk "github.com/cosmos/cosmos-sdk/types"

type KeeperPoolAddresses interface {
	SetPoolAddresses(ctx sdk.Context, addresses *PoolAddresses)
	GetPoolAddresses(ctx sdk.Context) PoolAddresses
}

// SetPoolAddresses save the pool address to key value store
func (k KVStore) SetPoolAddresses(ctx sdk.Context, addresses *PoolAddresses) {
	key := k.GetKey(ctx, prefixPoolAddresses, "")
	store := ctx.KVStore(k.storeKey)
	store.Set([]byte(key), k.cdc.MustMarshalBinaryBare(*addresses))
}

// GetPoolAddresses get current pool addresses
func (k KVStore) GetPoolAddresses(ctx sdk.Context) PoolAddresses {
	var addr PoolAddresses
	key := k.GetKey(ctx, prefixPoolAddresses, "")
	store := ctx.KVStore(k.storeKey)
	if store.Has([]byte(key)) {
		buf := store.Get([]byte(key))
		_ = k.cdc.UnmarshalBinaryBare(buf, &addr)
	}
	return addr
}