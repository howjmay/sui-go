package client_test

import (
	"context"
	"testing"

	"github.com/howjmay/go-sui-sdk/account"
	"github.com/howjmay/go-sui-sdk/sui_types"
	"github.com/howjmay/go-sui-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestAccountSignAndSend(t *testing.T) {
	account, err := account.NewAccountWithMnemonic(account.TEST_MNEMONIC)
	require.NoError(t, err)
	t.Log(account.Address)

	cli := TestnetClient(t)
	signer := AddressFromStrMust(account.Address)
	coins, err := cli.GetSuiCoinsOwnedByAddress(context.Background(), *signer)
	require.NoError(t, err)
	require.Greater(t, coins.TotalBalance().Int64(), sui_types.SUI(0.01).Int64(), "insufficient balance")

	coinIDs := make([]sui_types.ObjectID, len(coins))
	for i, c := range coins {
		coinIDs[i] = c.CoinObjectID
	}
	gasBudget := types.NewSafeSuiBigInt(uint64(10000000))
	txn, err := cli.PayAllSui(context.Background(), *signer, *signer, coinIDs, gasBudget)
	require.NoError(t, err)

	resp := executeTxn(t, cli, txn.TxBytes, account)
	t.Log("txn digest: ", resp.Digest)
}
