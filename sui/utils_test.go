package sui_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/howjmay/sui-go/lib"
	"github.com/howjmay/sui-go/models"
	"github.com/howjmay/sui-go/sui"
	"github.com/howjmay/sui-go/sui_types"

	"github.com/stretchr/testify/require"
)

func AddressFromStrMust(str string) *sui_types.SuiAddress {
	s, _ := sui_types.NewAddressFromHex(str)
	return s
}

// @return types.DryRunTransactionBlockResponse
func dryRunTxn(
	t *testing.T,
	api *sui.ImplSuiAPI,
	txBytes lib.Base64Data,
	showJson bool,
) *models.DryRunTransactionBlockResponse {
	simulate, err := api.DryRunTransaction(context.Background(), txBytes)
	require.NoError(t, err)
	require.Equal(t, "", simulate.Effects.Data.V1.Status.Error)
	require.True(t, simulate.Effects.Data.IsSuccess())
	if showJson {
		data, err := json.Marshal(simulate)
		require.NoError(t, err)
		t.Log(string(data))
		t.Log("gasFee: ", simulate.Effects.Data.GasFee())
	}
	return simulate
}

func executeTxn(
	t *testing.T,
	api *sui.ImplSuiAPI,
	txBytes lib.Base64Data,
	acc *sui_types.Account,
) *models.SuiTransactionBlockResponse {
	// First of all, make sure that there are no problems with simulated trading.
	simulate, err := api.DryRunTransaction(context.Background(), txBytes)
	require.NoError(t, err)
	require.True(t, simulate.Effects.Data.IsSuccess())

	// sign and send
	signature, err := acc.SignTransactionBlock(txBytes, sui_types.DefaultIntent())
	require.NoError(t, err)
	options := models.SuiTransactionBlockResponseOptions{
		ShowEffects: true,
	}
	resp, err := api.ExecuteTransactionBlock(
		context.TODO(), txBytes, []any{signature}, &options,
		models.TxnRequestTypeWaitForLocalExecution,
	)
	require.NoError(t, err)
	t.Log(resp)
	return resp
}
