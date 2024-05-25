package isc_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/howjmay/sui-go/isc"
	"github.com/howjmay/sui-go/models"
	"github.com/howjmay/sui-go/sui"
	"github.com/howjmay/sui-go/sui/conn"
	"github.com/howjmay/sui-go/sui_signer"
	"github.com/howjmay/sui-go/utils"

	"github.com/stretchr/testify/require"
)

type Client struct {
	API sui.ImplSuiAPI
}

func TestStartNewChain(t *testing.T) {
	t.Skip("only for localnet")
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	modules, err := utils.MoveBuild(utils.GetGitRoot() + "/isc/contracts/isc/")
	require.NoError(t, err)

	txnBytes, err := client.Publish(context.Background(), signer.Address, modules.Modules, modules.Dependencies, nil, new(models.BigInt).SetUint64(uint64(100000000)))
	require.NoError(t, err)
	txnResponse, err := client.SignAndExecuteTransaction(context.Background(), signer, txnBytes.TxBytes, &models.SuiTransactionBlockResponseOptions{
		ShowEffects:       true,
		ShowObjectChanges: true,
	})
	require.NoError(t, err)
	require.True(t, txnResponse.Effects.Data.IsSuccess())

	packageID, err := txnResponse.GetPublishedPackageID()
	require.NoError(t, err)
	t.Log("packageID: ", packageID)

	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		packageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, startNewChainRes.Effects.Data.IsSuccess())
	t.Logf("StartNewChain response: %#v\n", startNewChainRes)
}

func TestSendCoin(t *testing.T) {
	t.Skip("only for localnet")
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	iscPackageID := sui.BuildDeployContract(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/isc/contracts/isc/",
	)
	tokenPackageID, _ := sui.BuildDeployMintCoin(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/contracts/testcoin/",
		100000,
		"testtoken",
	)

	// start a new chain
	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		iscPackageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, startNewChainRes.Effects.Data.IsSuccess())

	anchorObjID, _, err := sui.GetCreatedObjectIdAndType(startNewChainRes, "anchor", "Anchor")
	coinType := fmt.Sprintf("%s::testcoin::TESTCOIN", tokenPackageID.String())
	require.NoError(t, err)
	// the signer should have only one coin object which belongs to testcoin type
	coins, err := client.GetCoins(context.Background(), signer.Address, &coinType, nil, 10)
	require.NoError(t, err)

	sendCoinRes, err := client.SendCoin(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		coinType,
		coins.Data[0].CoinObjectID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		})
	require.NoError(t, err)
	require.True(t, sendCoinRes.Effects.Data.IsSuccess())

	getObjectRes, err := client.GetObject(context.Background(), coins.Data[0].CoinObjectID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, anchorObjID.String(), getObjectRes.Data.Owner.ObjectOwnerInternal.AddressOwner.String())
}

func TestReceiveCoin(t *testing.T) {
	t.Skip("only for localnet")
	var err error
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	iscPackageID := sui.BuildDeployContract(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/isc/contracts/isc/",
	)
	tokenPackageID, _ := sui.BuildDeployMintCoin(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/contracts/testcoin/",
		100000,
		"testtoken",
	)

	// start a new chain
	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		iscPackageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, startNewChainRes.Effects.Data.IsSuccess())

	var assets1, assets2 *isc.Assets
	for _, change := range startNewChainRes.ObjectChanges {
		if change.Data.Created != nil {
			assets1, err = client.GetAssets(context.Background(), iscPackageID, &change.Data.Created.ObjectID)
			require.NoError(t, err)
			require.Len(t, assets1.Coins, 0)
		}
	}

	anchorObjID, _, err := sui.GetCreatedObjectIdAndType(startNewChainRes, "anchor", "Anchor")
	coinType := fmt.Sprintf("%s::testcoin::TESTCOIN", tokenPackageID.String())
	require.NoError(t, err)
	// the signer should have only one coin object which belongs to testcoin type
	coins, err := client.GetCoins(context.Background(), signer.Address, &coinType, nil, 10)
	require.NoError(t, err)

	sendCoinRes, err := client.SendCoin(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		coinType,
		coins.Data[0].CoinObjectID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		})
	require.NoError(t, err)
	require.True(t, sendCoinRes.Effects.Data.IsSuccess())

	getObjectRes, err := client.GetObject(context.Background(), coins.Data[0].CoinObjectID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, anchorObjID.String(), getObjectRes.Data.Owner.ObjectOwnerInternal.AddressOwner.String())

	receiveCoinRes, err := client.ReceiveCoin(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		coinType,
		coins.Data[0].CoinObjectID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		})
	require.NoError(t, err)
	require.True(t, receiveCoinRes.Effects.Data.IsSuccess())
	assets2, err = client.GetAssets(context.Background(), iscPackageID, anchorObjID)
	require.NoError(t, err)
	require.Len(t, assets2.Coins, 1)
}

func TestCreateRequest(t *testing.T) {
	t.Skip("only for localnet")
	var err error
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	iscPackageID := sui.BuildDeployContract(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/isc/contracts/isc/",
	)

	// start a new chain
	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		iscPackageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)

	anchorObjID, _, err := sui.GetCreatedObjectIdAndType(startNewChainRes, "anchor", "Anchor")
	require.NoError(t, err)

	createReqRes, err := client.CreateRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		"isc_test_contract_name",
		"isc_test_func_name",
		[][]byte{}, // func input
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		})
	require.NoError(t, err)
	require.True(t, createReqRes.Effects.Data.IsSuccess())

	_, _, err = sui.GetCreatedObjectIdAndType(createReqRes, "request", "Request")
	require.NoError(t, err)
}

func TestSendRequest(t *testing.T) {
	t.Skip("only for localnet")
	var err error
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	iscPackageID := sui.BuildDeployContract(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/isc/contracts/isc/",
	)

	// start a new chain
	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		iscPackageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)

	anchorObjID, _, err := sui.GetCreatedObjectIdAndType(startNewChainRes, "anchor", "Anchor")
	require.NoError(t, err)

	createReqRes, err := client.CreateRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		"isc_test_contract_name",
		"isc_test_func_name",
		[][]byte{}, // func input
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, createReqRes.Effects.Data.IsSuccess())

	reqObjID, _, err := sui.GetCreatedObjectIdAndType(createReqRes, "request", "Request")
	require.NoError(t, err)
	getObjectRes, err := client.GetObject(context.Background(), reqObjID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, signer.Address, getObjectRes.Data.Owner.AddressOwner)

	sendReqRes, err := client.SendRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		reqObjID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, sendReqRes.Effects.Data.IsSuccess())

	getObjectRes, err = client.GetObject(context.Background(), reqObjID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, anchorObjID, getObjectRes.Data.Owner.AddressOwner)
}

func TestReceiveRequest(t *testing.T) {
	t.Skip("only for localnet")
	var err error
	suiClient, signer := sui.NewTestSuiClientWithSignerAndFund(conn.LocalnetEndpointUrl, sui_signer.TEST_MNEMONIC)
	client := isc.NewIscClient(suiClient)

	iscPackageID := sui.BuildDeployContract(
		t,
		client.ImplSuiAPI,
		signer,
		utils.GetGitRoot()+"/isc/contracts/isc/",
	)

	// start a new chain
	startNewChainRes, err := client.StartNewChain(
		context.Background(),
		signer,
		iscPackageID,
		sui.DefaultGasBudget,
		&models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)

	anchorObjID, _, err := sui.GetCreatedObjectIdAndType(startNewChainRes, "anchor", "Anchor")
	require.NoError(t, err)

	createReqRes, err := client.CreateRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		"isc_test_contract_name", // FIXME set up the proper ISC target contract name
		"isc_test_func_name",     // FIXME set up the proper ISC target func name
		[][]byte{},               // func input
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, createReqRes.Effects.Data.IsSuccess())

	reqObjID, _, err := sui.GetCreatedObjectIdAndType(createReqRes, "request", "Request")
	require.NoError(t, err)
	getObjectRes, err := client.GetObject(context.Background(), reqObjID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, signer.Address, getObjectRes.Data.Owner.AddressOwner)

	sendReqRes, err := client.SendRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		reqObjID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, sendReqRes.Effects.Data.IsSuccess())

	getObjectRes, err = client.GetObject(context.Background(), reqObjID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.Equal(t, anchorObjID, getObjectRes.Data.Owner.AddressOwner)

	receiveReqRes, err := client.ReceiveRequest(
		context.Background(),
		signer,
		iscPackageID,
		anchorObjID,
		reqObjID,
		sui.DefaultGasBudget, &models.SuiTransactionBlockResponseOptions{
			ShowEffects:       true,
			ShowObjectChanges: true,
		},
	)
	require.NoError(t, err)
	require.True(t, receiveReqRes.Effects.Data.IsSuccess())

	getObjectRes, err = client.GetObject(context.Background(), reqObjID, &models.SuiObjectDataOptions{ShowOwner: true})
	require.NoError(t, err)
	require.NotNil(t, getObjectRes.Error.Data.Deleted)
}
