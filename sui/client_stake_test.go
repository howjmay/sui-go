package sui_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/howjmay/sui-go/models"
	"github.com/howjmay/sui-go/sui"
	"github.com/howjmay/sui-go/sui/conn"
	"github.com/howjmay/sui-go/sui_types"
	"github.com/stretchr/testify/require"
)

const (
	ComingChatValidatorAddress = "0x520289e77c838bae8501ae92b151b99a54407288fdd20dee6e5416bfe943eb7a"
)

func TestClient_GetLatestSuiSystemState(t *testing.T) {
	api := sui.NewSuiClient(conn.MainnetEndpointUrl)
	state, err := api.GetLatestSuiSystemState(context.Background())
	require.NoError(t, err)
	t.Logf("system state: %v", state)
}

func TestClient_GetValidatorsApy(t *testing.T) {
	api := sui.NewSuiClient(conn.DevnetEndpointUrl)
	apys, err := api.GetValidatorsApy(context.Background())
	require.NoError(t, err)
	t.Logf("current epoch %v", apys.Epoch)
	apyMap := apys.ApyMap()
	for _, apy := range apys.Apys {
		key := apy.Address
		t.Logf("%v apy: %v", key, apyMap[key])
	}
}

func TestGetDelegatedStakes(t *testing.T) {
	api := sui.NewSuiClient(conn.DevnetEndpointUrl)

	address, err := sui_types.NewAddressFromHex("0xd77955e670f42c1bc5e94b9e68e5fe9bdbed9134d784f2a14dfe5fc1b24b5d9f")
	require.NoError(t, err)
	stakes, err := api.GetStakes(context.Background(), address)
	require.NoError(t, err)

	for _, validator := range stakes {
		for _, stake := range validator.Stakes {
			if stake.Data.StakeStatus.Data.Active != nil {
				t.Logf(
					"earned amount %10v at %v",
					stake.Data.StakeStatus.Data.Active.EstimatedReward.Uint64(),
					validator.ValidatorAddress,
				)
			}
		}
	}
}

func TestGetStakesByIds(t *testing.T) {
	api := sui.NewSuiClient(conn.TestnetEndpointUrl)
	owner, err := sui_types.NewAddressFromHex("0xd77955e670f42c1bc5e94b9e68e5fe9bdbed9134d784f2a14dfe5fc1b24b5d9f")
	require.NoError(t, err)
	stakes, err := api.GetStakes(context.Background(), owner)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(stakes), 1)

	stake1 := stakes[0].Stakes[0].Data
	stakeId := stake1.StakedSuiId
	stakesFromId, err := api.GetStakesByIds(context.Background(), []sui_types.ObjectID{stakeId})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(stakesFromId), 1)

	queriedStake := stakesFromId[0].Stakes[0].Data
	require.Equal(t, stake1, queriedStake)
	t.Log(stakesFromId)
}

func TestRequestAddDelegation(t *testing.T) {
	api := sui.NewSuiClient(conn.TestnetEndpointUrl)
	signer := sui_types.TEST_ADDRESS

	coins, err := api.GetCoins(context.Background(), signer, nil, nil, 10)
	require.NoError(t, err)

	amount := sui_types.SUI(1).Uint64()
	gasBudget := sui_types.SUI(0.01).Uint64()
	pickedCoins, err := models.PickupCoins(coins, *big.NewInt(0).SetUint64(amount), 0, 0, 0)
	require.NoError(t, err)

	validatorAddress := ComingChatValidatorAddress
	validator, err := sui_types.NewAddressFromHex(validatorAddress)
	require.NoError(t, err)

	txBytes, err := sui.BCS_RequestAddStake(
		signer,
		pickedCoins.CoinRefs(),
		models.NewSafeSuiBigInt(amount),
		validator,
		gasBudget,
		1000,
	)
	require.NoError(t, err)

	dryRunTxn(t, api, txBytes, true)
}

func TestRequestWithdrawDelegation(t *testing.T) {
	api := sui.NewSuiClient(conn.TestnetEndpointUrl)
	gasBudget := sui_types.SUI(1).Uint64()

	signer, err := sui_types.NewAddressFromHex("0xd77955e670f42c1bc5e94b9e68e5fe9bdbed9134d784f2a14dfe5fc1b24b5d9f")
	require.NoError(t, err)
	stakes, err := api.GetStakes(context.Background(), signer)
	require.NoError(t, err)
	require.True(t, len(stakes) > 0)
	require.True(t, len(stakes[0].Stakes) > 0)

	coins, err := api.GetCoins(context.Background(), signer, nil, nil, 10)
	require.NoError(t, err)
	pickedCoins, err := models.PickupCoins(coins, *big.NewInt(0), gasBudget, 0, 0)
	require.NoError(t, err)

	stakeId := stakes[0].Stakes[0].Data.StakedSuiId
	detail, err := api.GetObject(context.Background(), &stakeId, nil)
	require.NoError(t, err)
	txBytes, err := sui.BCS_RequestWithdrawStake(signer, detail.Data.Reference(), pickedCoins.CoinRefs(), gasBudget, 1000)
	require.NoError(t, err)

	dryRunTxn(t, api, txBytes, true)
}
