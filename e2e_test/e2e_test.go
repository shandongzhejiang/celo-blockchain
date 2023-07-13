package e2e_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/common/hexutil"
	"github.com/celo-org/celo-blockchain/core/types"
	"github.com/celo-org/celo-blockchain/eth/tracers"
	"github.com/celo-org/celo-blockchain/log"
	"github.com/celo-org/celo-blockchain/mycelo/env"
	"github.com/celo-org/celo-blockchain/mycelo/genesis"
	"github.com/celo-org/celo-blockchain/node"
	"github.com/celo-org/celo-blockchain/rpc"
	"github.com/celo-org/celo-blockchain/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// Force-load native and js packages, to trigger registration
	_ "github.com/celo-org/celo-blockchain/eth/tracers/js"
	_ "github.com/celo-org/celo-blockchain/eth/tracers/native"
)

func init() {
	// This statement is commented out but left here since its very useful for
	// debugging problems and its non trivial to construct.
	//
	// log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	// This disables all logging which in general we want, because there is a lot
	log.Root().SetHandler(log.DiscardHandler())
}

// This test starts a network submits a transaction and waits for the whole
// network to process the transaction.
func TestSendCelo(t *testing.T) {
	ac := test.AccountConfig(3, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)
}

// This test starts a network, submits a GoldToken.transfer tx, waits for the whole
// network to process the transaction, and traces that tx.
// This tests that CELO transfers made via the transfer precompile work e2e
// and can be useful for debugging these traces.
func TestTraceSendCeloViaGoldToken(t *testing.T) {
	ac := test.AccountConfig(3, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())
	// Send 1 wei of CELO from accounts[0] to accounts[1] by calling GoldToken.transfer
	tx, err := accounts[0].SendCeloViaGoldToken(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)
	c, err := rpc.DialContext(ctx, network[0].WSEndpoint())
	require.NoError(t, err)

	var result map[string]interface{}
	tracerStr := "callTracer"
	err = c.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash().String(), tracers.TraceConfig{Tracer: &tracerStr})

	require.NoError(t, err)
	// Check top level gas values
	require.Equal(t, "0x45d6", result["gasUsed"])
	require.Equal(t, "0x4653", result["gas"])
	// TODO add more specific trace-checking as part of
	// this issue: https://github.com/celo-org/celo-blockchain/issues/2078
}

// Moved from API tests because registering the callTracer (necessary after the
// go native tracer refactor) causes a circular import.
// Use the callTracer to trace a native CELO transfer.
func TestCallTraceTransactionNativeTransfer(t *testing.T) {
	ac := test.AccountConfig(1, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)
	c, err := rpc.DialContext(ctx, network[0].WSEndpoint())
	require.NoError(t, err)

	var result map[string]interface{}
	tracerStr := "callTracer"
	err = c.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash().String(), tracers.TraceConfig{Tracer: &tracerStr})
	require.NoError(t, err)
	res_json, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	expectedTraceStr := fmt.Sprintf(`{"from":"0x%x","gas":"0x0","gasUsed":"0x0","input":"0x","output":"0x","to":"0x%x","type":"CALL","value":"0x1"}`, accounts[0].Address, accounts[1].Address)
	require.JSONEq(t, expectedTraceStr, string(res_json))
}

// Moved from API tests because registering the prestateTracer (necessary after the
// go native tracer refactor) causes a circular import.
// Use the prestateTracer to trace a native CELO transfer.
func TestPrestateTransactionNativeTransfer(t *testing.T) {
	ac := test.AccountConfig(1, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)
	c, err := rpc.DialContext(ctx, network[0].WSEndpoint())
	require.NoError(t, err)

	var result map[string]interface{}
	tracerStr := "prestateTracer"
	err = c.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash().String(), tracers.TraceConfig{Tracer: &tracerStr})
	require.NoError(t, err)

	toAddrLowercase := strings.ToLower(accounts[1].Address.String())
	if _, has := result[toAddrLowercase]; !has {
		t.Fatalf("Expected %s in result", toAddrLowercase)
	}
}

// This test verifies correct behavior in a network of size one, in the case that
// this fails we know that the problem does not lie with our network code.
func TestSingleNodeNetworkManyTxs(t *testing.T) {
	iterations := 5
	txsPerIteration := 5
	ac := test.AccountConfig(1, 1)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	gc.Istanbul.Epoch = uint64(iterations) * 50 // avoid the epoch for this test
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())
	for r := 0; r < iterations; r++ {
		txs := make([]*types.Transaction, 0, txsPerIteration)
		for j := 0; j < txsPerIteration; j++ {
			tx, err := accounts[0].SendCelo(ctx, common.Address{}, 1, network[0])
			require.NoError(t, err)
			require.NotNil(t, tx)
		}
		err = network.AwaitTransactions(ctx, txs...)
		require.NoError(t, err)
	}
}

// This test is intended to ensure that epoch blocks can be correctly marshalled.
// We previously had an open bug for this https://github.com/celo-org/celo-blockchain/issues/1574
func TestEpochBlockMarshaling(t *testing.T) {
	accounts := test.AccountConfig(1, 0)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(accounts, gingerbreadBlock)
	require.NoError(t, err)

	// Configure the shortest possible epoch, uptimeLookbackWindow minimum is 3
	// and it needs to be < (epoch -2).
	ec.Istanbul.Epoch = 6
	ec.Istanbul.DefaultLookbackWindow = 3
	network, shutdown, err := test.NewNetwork(accounts, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	// Wait for the whole network to process the transaction.
	err = network.AwaitBlock(ctx, 6)
	require.NoError(t, err)
	b := network[0].Tracker.GetProcessedBlock(6)

	// Check that epoch snark data was actually unmarshalled, I.E there was
	// something there.
	assert.True(t, len(b.EpochSnarkData().Signature) > 0)
	assert.True(t, b.EpochSnarkData().Bitmap.Uint64() > 0)
}

// This test checks that a network can have validators shut down mid operation
// and that it can continue to function, it also checks that if more than f
// validators are shut down, when they restart the network is able to continue.
func TestStartStopValidators(t *testing.T) {
	ac := test.AccountConfig(4, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, _, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)

	// We define our own shutdown function because we don't want to print
	// errors about already stopped nodes. Since this test can fail while we
	// have stopped nodes.
	defer func() {
		for _, err := range network.Shutdown() {
			if !errors.Is(err, test.ErrTrackerAlreadyStopped) && !errors.Is(err, node.ErrNodeStopped) {
				fmt.Println(err.Error())
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*40)
	defer cancel()

	var txs []*types.Transaction

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	txs = append(txs, tx)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

	// Stop one node, the rest of the network should still be able to progress
	err = network[3].Close()
	require.NoError(t, err)

	// Send one celo from external account 0 to 1 via node 0.
	tx, err = accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	txs = append(txs, tx)

	// Check that the remaining network can still process this transction.
	err = network[:3].AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

	// Stop another node, the network should now be stuck
	err = network[2].Close()
	require.NoError(t, err)

	// Now we will check that the network does not process transactions in this
	// state, by waiting for a reasonable amount of time for it to process a
	// transaction and assuming it is not processing transactions if we time out.
	shortCtx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	// Send one celo from external account 0 to 1 via node 0.
	tx, err = accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	txs = append(txs, tx)

	err = network[:2].AwaitTransactions(shortCtx, txs...)
	// Expect DeadlineExceeded error
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expecting %q, instead got: %v ", context.DeadlineExceeded.Error(), err)
	}

	// Start the last stopped node
	err = network[2].Start()
	require.NoError(t, err)

	// We need to wait here to allow the call to "Backend.RefreshValPeers" to
	// complete before adding peers. This is because "Backend.RefreshValPeers"
	// deletes all peers and then re-adds any peers from the cached
	// connections, but in the case that peers were recently added there may
	// not have been enough time to connect to them and populate the connection
	// cache, and in that case "Backend.RefreshValPeers" simply removes all the
	// peers.
	time.Sleep(250 * time.Millisecond)
	// Connect last stopped node to running nodes
	network[2].AddPeers(network[:2]...)
	time.Sleep(25 * time.Millisecond)
	for _, n := range network[:3] {
		err = n.GossipEnodeCertificatge()
		require.NoError(t, err)
	}

	// Check that the  network now processes the previous transaction.
	err = network[:3].AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

	// Check that the network now quickly processes incoming transactions.
	// Send one celo from external account 0 to 1 via node 0.
	tx, err = accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	txs = append(txs, tx)

	err = network[:3].AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

	// Start the first stopped node
	err = network[3].Start()
	require.NoError(t, err)

	// We need to wait here to allow the call to "Backend.RefreshValPeers" to
	// complete before adding peers. This is because "Backend.RefreshValPeers"
	// deletes all peers and then re-adds any peers from the cached
	// connections, but in the case that peers were recently added there may
	// not have been enough time to connect to them and populate the connection
	// cache, and in that case "Backend.RefreshValPeers" simply removes all the
	// peers.
	time.Sleep(250 * time.Millisecond)
	// Connect final node to rest of network
	network[3].AddPeers(network[:3]...)
	time.Sleep(25 * time.Millisecond)
	for _, n := range network {
		err = n.GossipEnodeCertificatge()
		require.NoError(t, err)
	}

	// Check that the network continues to quickly processes incoming transactions.
	// Send one celo from external account 0 to 1 via node 0.
	tx, err = accounts[0].SendCelo(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	txs = append(txs, tx)

	err = network.AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

}

// This test was created to reproduce the concurrent map access error in
// https://github.com/celo-org/celo-blockchain/issues/1799
//
// It does this by calling debug_traceBlockByNumber a number of times since the
// trace block code was the source of the concurrent map access.
func TestBlockTracingConcurrentMapAccess(t *testing.T) {
	ac := test.AccountConfig(1, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	n := network[0]

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	var txs []*types.Transaction
	// Send one celo from external account 0 to 1 via node 0.
	for i := 0; i < 10; i++ {
		tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, n)
		require.NoError(t, err)
		txs = append(txs, tx)
	}

	// Wait for the whole network to process the transactions.
	err = network.AwaitTransactions(ctx, txs...)
	require.NoError(t, err)

	lastTx := txs[len(txs)-1]

	b := n.Tracker.GetProcessedBlockForTx(lastTx.Hash())

	var wg sync.WaitGroup
	for i := 1; i < +int(b.NumberU64()); i++ {
		wg.Add(1)
		num := i
		go func() {
			defer wg.Done()
			c, err := rpc.DialContext(ctx, n.WSEndpoint())
			require.NoError(t, err)

			var result []interface{}
			err = c.CallContext(ctx, &result, "debug_traceBlockByNumber", hexutil.EncodeUint64(uint64(num)))
			require.NoError(t, err)
		}()

	}
	wg.Wait()
}

// Sends and traces a single native transfer.
// Helpful for debugging issues in TestBlockTracingConcurrentMapAccess.
func TestBlockTracingSequentialAccess(t *testing.T) {
	ac := test.AccountConfig(1, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()
	n := network[0]
	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())
	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCelo(ctx, accounts[1].Address, 1, n)
	require.NoError(t, err)
	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)
	b := n.Tracker.GetProcessedBlockForTx(tx.Hash())
	c, err := rpc.DialContext(ctx, n.WSEndpoint())
	require.NoError(t, err)
	var result []interface{}
	err = c.CallContext(ctx, &result, "debug_traceBlockByNumber", hexutil.EncodeUint64(uint64(int(b.NumberU64()))))
	require.NoError(t, err)
}

type rpcCustomTransaction struct {
	BlockNumber *hexutil.Big `json:"blockNumber"`
	BlockHash   *common.Hash `json:"blockHash"`
	GasPrice    *hexutil.Big `json:"gasPrice"`
}

// TestRPCDynamicTxGasPriceWithBigFeeCap test that after a dynamic tx
// was added to a block, the rpc sends in the gasPrice the actual consumed
// price by the tx, which could be less than the feeCap (as in this example)
func TestRPCDynamicTxGasPriceWithBigFeeCap(t *testing.T) {
	ac := test.AccountConfig(3, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	suggestedGasPrice, err := network[0].WsClient.SuggestGasPrice(ctx)
	require.NoError(t, err)
	gasFeeCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(90))
	gasTipCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(2))

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCeloWithDynamicFee(ctx, accounts[1].Address, 1, gasFeeCap, gasTipCap, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)

	var json *rpcCustomTransaction
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &json, "eth_getTransactionByHash", tx.Hash())
	require.NoError(t, err)
	require.NotNil(t, json.BlockNumber)
	gasPrice := json.GasPrice.ToInt()
	require.NotNil(t, json.GasPrice)
	require.Greater(t, gasPrice.Int64(), gasTipCap.Int64())
	require.Less(t, gasPrice.Int64(), gasFeeCap.Int64())
}

// TestRPCDynamicTxGasPriceWithState aims to test the scenario where a
// an old dynamic tx is requested via rpc, to an archive node.
// As right now on Celo, we are not storing the baseFee in the header (as ethereum does),
// to know the exactly gasPrice expent in a dynamic tx, depends on consuming the
// GasPriceMinimum contract
func TestRPCDynamicTxGasPriceWithState(t *testing.T) {
	ac := test.AccountConfig(3, 2)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	ec.TxLookupLimit = 0
	ec.NoPruning = true
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	suggestedGasPrice, err := network[0].WsClient.SuggestGasPrice(ctx)
	require.NoError(t, err)
	gasFeeCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(90))
	gasTipCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(2))

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendCeloWithDynamicFee(ctx, accounts[1].Address, 1, gasFeeCap, gasTipCap, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)

	var json *rpcCustomTransaction
	// Check that the transaction can be retrieved via the rpc api
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &json, "eth_getTransactionByHash", tx.Hash())
	require.NoError(t, err)
	// Blocknumber != nil it means that it eas already processed
	require.NotNil(t, json.BlockNumber)

	// Create a block containing a transaction, we will prune the state of this block.
	_, err = accounts[0].SendCeloTracked(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)

	// Prune state
	// As the gasPrice of the block N, is the one from the state of the block N-1, we need to prune the parent block
	err = pruneStateOfBlock(ctx, network[0], new(big.Int).Sub(json.BlockNumber.ToInt(), common.Big1))
	require.NoError(t, err)

	var json2 *rpcCustomTransaction
	// Check that the transaction can still be retrieved via the rpc api
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &json2, "eth_getTransactionByHash", tx.Hash())
	require.NoError(t, err)
	// if the object is nil, it means that was not found
	require.NotNil(t, json2)
	// Blocknumber != nil it means that it eas already processed
	require.NotNil(t, json2.BlockNumber)
	require.Equal(t, json.GasPrice, json2.GasPrice)
}

// TestRPCDynamicTxGasPriceWithoutState aims to test the scenario where a
// an old dynamic tx is requested via rpc, to a full node that does not have
// the state anymore.
func TestRPCDynamicTxGasPriceWithoutStateBeforeGingerbread(t *testing.T) {
	testRPCDynamicTxGasPriceWithoutState(t, false, false)
}

func TestRPCDynamicTxGasPriceWithoutStateAfterGingerbread(t *testing.T) {
	testRPCDynamicTxGasPriceWithoutState(t, true, false)
}

func TestRPCDynamicTxGasPriceWithoutStateForAlternativeCurrencyBeforeGingerbread(t *testing.T) {
	testRPCDynamicTxGasPriceWithoutState(t, false, true)
}

func TestRPCDynamicTxGasPriceWithoutStateForAlternativeCurrencyAfterGingerbread(t *testing.T) {
	testRPCDynamicTxGasPriceWithoutState(t, true, true)
}

func testRPCDynamicTxGasPriceWithoutState(t *testing.T, afterGingerbread, alternativeCurrency bool) {
	var gingerbreadBlock *big.Int
	if afterGingerbread {
		gingerbreadBlock = common.Big1
	}
	cusdAddress := common.HexToAddress("0xd008")
	ac := test.AccountConfig(3, 2)
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	ec.TrieDirtyCache = 5
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*90)
	defer cancel()

	accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())

	var feeCurrency *common.Address
	if alternativeCurrency {
		feeCurrency = &cusdAddress
	}
	suggestedGasPrice, err := network[0].WsClient.SuggestGasPriceInCurrency(ctx, feeCurrency)
	require.NoError(t, err)
	gasFeeCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(90))
	gasTipCap := big.NewInt(0).Mul(suggestedGasPrice, big.NewInt(2))

	// Send one celo from external account 0 to 1 via node 0.
	tx, err := accounts[0].SendValueWithDynamicFee(ctx, accounts[1].Address, 1, feeCurrency, gasFeeCap, gasTipCap, network[0])
	require.NoError(t, err)

	// Wait for the whole network to process the transaction.
	err = network.AwaitTransactions(ctx, tx)
	require.NoError(t, err)

	var json *rpcCustomTransaction
	// Check that the transaction can be retrieved via the rpc api
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &json, "eth_getTransactionByHash", tx.Hash())
	require.NoError(t, err)
	// Blocknumber != nil it means that it eas already processed
	require.NotNil(t, json.BlockNumber)

	// Create one block to be able to prune the last state
	_, err = accounts[0].SendCeloTracked(ctx, accounts[1].Address, 1, network[0])
	require.NoError(t, err)
	// Prune state
	// As the gasPrice of the block N, is the one from the state of the block N-1, we need to prune the parent block
	err = pruneStateOfBlock(ctx, network[0], new(big.Int).Sub(json.BlockNumber.ToInt(), common.Big1))
	require.NoError(t, err)

	var json2 *rpcCustomTransaction
	// Check that the transaction can still be retrieved via the rpc api
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &json2, "eth_getTransactionByHash", tx.Hash())
	require.NoError(t, err)
	// if the object is nil, it means that was not found
	require.NotNil(t, json2)
	// Blocknumber != nil it means that it eas already processed
	require.NotNil(t, json2.BlockNumber)

	if afterGingerbread && !alternativeCurrency {
		require.Equal(t, json.GasPrice, json2.GasPrice)
	} else {
		require.Nil(t, json2.GasPrice)
	}
}

func pruneStateOfBlock(ctx context.Context, node *test.Node, blockNumber *big.Int) error {
	var block *types.Block
	block, err := node.WsClient.BlockByNumber(ctx, blockNumber)
	if err != nil {
		return err
	}
	root := block.Root()
	node.Eth.BlockChain().StateCache().TrieDB().Dereference(root)

	return nil
}

func runMochaTest(t *testing.T, add_args func(*env.AccountsConfig, *genesis.Config, test.Network) []string) {
	ac := test.AccountConfig(1, 1)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()

	// Execute typescript tests to check ethers.js compatibility.
	//
	// The '--networkaddr' and '--blocknum' flags are npm config variables, the
	// values become available under 'process.env.npm_config_networkaddr' and
	// 'process.env.npm_config_blocknum' in typescript test. Everything after
	// '--' are flags that are passed to mocha and these flags are controlling
	// which tests to run.

	// The tests don't seem to work on CI with IPV6 addresses so we convert to IPV4 here
	addr := strings.Replace(network[0].Node.HTTPEndpoint(), "[::]", "127.0.0.1", 1)

	common_args := []string{"run", "--networkaddr=" + addr}
	custom_args := add_args(ac, gc, network)

	cmd := exec.Command("npm", append(common_args, custom_args...)...)

	cmd.Dir = "./ethersjs-api-check/"
	println("executing mocha test with", cmd.String())
	output, err := cmd.CombinedOutput()
	println(string(output))
	require.NoError(t, err)
}

func TestPrecompileWrappers(t *testing.T) {
	add_args := func(ac *env.AccountsConfig, gc *genesis.Config, network test.Network) []string {
		accounts := test.Accounts(ac.DeveloperAccounts(), gc.ChainConfig())
		privateKeyHex := fmt.Sprintf("0x%064x", accounts[0].Key.D)
		return []string{"test-precompile-wrappers", "--signerkey=" + privateKeyHex}
	}
	runMochaTest(t, add_args)
}

func TestEthersJSCompatibility(t *testing.T) {
	add_args := func(ac *env.AccountsConfig, gc *genesis.Config, network test.Network) []string {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()

		num, err := network[0].WsClient.BlockNumber(ctx)
		require.NoError(t, err)
		return []string{"test", "--blocknum=" + hexutil.Uint64(num).String(), "--", "--grep", "ethers.js compatibility tests with state"}
	}
	runMochaTest(t, add_args)
}

// This test checks the functionality of the configuration to enable/disable
// returning the 'gasLimit' and 'baseFeePerGas' fields on RPC blocks.
func TestEthersJSCompatibilityDisableAfterGingerbread(t *testing.T) {
	ac := test.AccountConfig(1, 1)
	gingerbreadBlock := common.Big1
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)

	// Check fields present (compatibility set by default)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	result := make(map[string]interface{})
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &result, "eth_getBlockByNumber", "latest", true)
	require.NoError(t, err)

	for _, field := range []string{"gasLimit", "baseFeePerGas", "sha3Uncles", "uncles", "nonce", "mixHash", "difficulty"} {
		_, ok := result[field]
		assert.Truef(t, ok, "%s field should be present on RPC block after Gingerbread", field)
	}
	require.Equal(t, result["sha3Uncles"], "0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")

	// Turn off compatibility and check fields are not present
	ec.RPCEthCompatibility = false
	network, shutdown, err = test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	result = make(map[string]interface{})
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &result, "eth_getBlockByNumber", "latest", true)
	require.NoError(t, err)

	// After Gingerbread, gasLimit should be returned directly from the header, even if
	// RPCEthCompatibility is off, since it is now part of the header hash.
	_, ok := result["gasLimit"]
	assert.True(t, ok, "gasLimit field must be present on RPC block after Gingerbread")
	_, ok = result["baseFeePerGas"]
	assert.True(t, ok, "baseFeePerGas field must be present on RPC block")
}

// This test checks the functionality of the configuration to enable/disable
// returning the 'gasLimit' and 'baseFeePerGas' fields on RPC blocks before the Gingerbread happened.
// Gingerbread is relevant because it added the gasLimit to the header.
func TestEthersJSCompatibilityDisableBeforeGingerbread(t *testing.T) {
	ac := test.AccountConfig(1, 1)
	var gingerbreadBlock *big.Int = nil
	gc, ec, err := test.BuildConfig(ac, gingerbreadBlock)
	require.NoError(t, err)

	// Check fields present (compatibility set by default)
	network, shutdown, err := test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	result := make(map[string]interface{})
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &result, "eth_getBlockByNumber", "latest", true)
	require.NoError(t, err)

	for _, field := range []string{"gasLimit", "baseFeePerGas", "difficulty"} {
		_, ok := result[field]
		assert.Truef(t, ok, "%s field should be present on RPC block before Gingerbread", field)
	}
	for _, field := range []string{"sha3Uncles", "uncles", "nonce", "mixHash"} {
		_, ok := result[field]
		assert.Falsef(t, ok, "%s field should not be present on RPC block before Gingerbread", field)
	}

	// Turn off compatibility and check fields are not present
	ec.RPCEthCompatibility = false
	network, shutdown, err = test.NewNetwork(ac, gc, ec)
	require.NoError(t, err)
	defer shutdown()

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	result = make(map[string]interface{})
	err = network[0].WsClient.GetRPCClient().CallContext(ctx, &result, "eth_getBlockByNumber", "latest", true)
	require.NoError(t, err)

	for _, field := range []string{"gasLimit", "baseFeePerGas", "sha3Uncles", "uncles", "nonce", "mixHash", "difficulty"} {
		_, ok := result[field]
		assert.Falsef(t, ok, "%s field should not be present on RPC block before Gingerbread", field)
	}
}
