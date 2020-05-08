package thorchain

import (
	"encoding/json"

	"github.com/blang/semver"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/constants"
)

// ReserveContributorHandler is handler to process MsgReserveContributor
type ReserveContributorHandler struct {
	keeper Keeper
}

// NewReserveContributorHandler create a new instance of ReserveContributorHandler
func NewReserveContributorHandler(keeper Keeper) ReserveContributorHandler {
	return ReserveContributorHandler{
		keeper: keeper,
	}
}

func (h ReserveContributorHandler) Run(ctx sdk.Context, m sdk.Msg, version semver.Version, _ constants.ConstantValues) sdk.Result {
	msg, ok := m.(MsgReserveContributor)
	if !ok {
		return errInvalidMessage.Result()
	}
	if err := h.Validate(ctx, msg, version); err != nil {
		ctx.Logger().Error("MsgReserveContributor failed validation", "error", err)
		return err.Result()
	}
	return h.Handle(ctx, msg, version)
}

func (h ReserveContributorHandler) Validate(ctx sdk.Context, msg MsgReserveContributor, version semver.Version) sdk.Error {
	if version.GTE(semver.MustParse("0.1.0")) {
		return h.ValidateV1(ctx, msg)
	}
	return errBadVersion
}

func (h ReserveContributorHandler) ValidateV1(ctx sdk.Context, msg MsgReserveContributor) sdk.Error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	return nil
}

func (h ReserveContributorHandler) Handle(ctx sdk.Context, msg MsgReserveContributor, version semver.Version) sdk.Result {
	ctx.Logger().Info("handleMsgReserveContributor request")
	if version.GTE(semver.MustParse("0.1.0")) {
		if err := h.HandleV1(ctx, msg); err != nil {
			ctx.Logger().Error("fail to process MsgReserveContributor", "error", err)
			return sdk.ErrInternal("fail to process reserve contributor").Result()
		}
		return sdk.Result{
			Code:      sdk.CodeOK,
			Codespace: DefaultCodespace,
		}
	}
	ctx.Logger().Error(errInvalidVersion.Error())
	return errBadVersion.Result()
}

// HandleV1  process MsgReserveContributor
func (h ReserveContributorHandler) HandleV1(ctx sdk.Context, msg MsgReserveContributor) error {
	reses, err := h.keeper.GetReservesContributors(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get reserve contributors", "error", err)
		return err
	}

	reses = reses.Add(msg.Contributor)
	if err := h.keeper.SetReserveContributors(ctx, reses); err != nil {
		ctx.Logger().Error("fail to save reserve contributors", "error", err)
		return err
	}

	vault, err := h.keeper.GetVaultData(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get vault data", "error", err)
		return err
	}

	vault.TotalReserve = vault.TotalReserve.Add(msg.Contributor.Amount)
	if err := h.keeper.SetVaultData(ctx, vault); err != nil {
		ctx.Logger().Error("fail to save vault data", "error", err)
		return err
	}
	reserveEvent := NewEventReserve(msg.Contributor)
	buf, err := json.Marshal(reserveEvent)
	if nil != err {
		return err
	}
	e := NewEvent(reserveEvent.Type(), ctx.BlockHeight(), msg.Tx, buf, EventSuccess)
	return h.keeper.UpsertEvent(ctx, e)
}
