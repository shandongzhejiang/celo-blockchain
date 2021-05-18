package ethapi

import (
	"context"
	"math/big"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/contract_comm/currency"
	"github.com/celo-org/celo-blockchain/core/types"
	"github.com/celo-org/celo-blockchain/params"
	"github.com/celo-org/celo-blockchain/rpc"
)

func checkCeloTxFeeTx(ctx context.Context, b Backend, tx *types.Transaction) error {
	currencyManager, err := newCurrencyManager(ctx, b)
	if err != nil {
		return err
	}
	return checkTxFee(currencyManager, tx.FeeCurrency(), tx.Fee(), b.RPCTxFeeCap())
}

func checkCeloTxArgsFee(ctx context.Context, b Backend, args SendTxArgs) error {
	return checkCeloTxFee(ctx, b, args.FeeCurrency, (*big.Int)(args.GasPrice), uint64(*args.Gas), (*big.Int)(args.GatewayFee))
}

func checkCeloTxFee(ctx context.Context, b Backend, feeCurrency *common.Address, gasPrice *big.Int, gas uint64, gatewayFee *big.Int) error {
	currencyManager, err := newCurrencyManager(ctx, b)
	if err != nil {
		return err
	}
	gFee := gatewayFee
	if gFee == nil {
		gFee = big.NewInt(0)
	}
	fee := types.Fee(gasPrice, gas, gFee)
	return checkTxFee(currencyManager, feeCurrency, fee, b.RPCTxFeeCap())
}

// newCurrencyManager creates and returns a currencyManager pointing to the latest block
// from the underlying chain from the Backend.
func newCurrencyManager(ctx context.Context, b Backend) (*currency.CurrencyManager, error) {
	stateDb, header, err := b.StateAndHeaderByNumber(ctx, rpc.LatestBlockNumber)
	if err != nil {
		return nil, err
	}
	return currency.NewManager(
		header,
		stateDb), nil
}

// getWei converts a celo float to a big.Int Wei representation
func getWei(celo float64) *big.Int {
	floatWei := new(big.Float).Mul(big.NewFloat(params.Ether), big.NewFloat(celo))
	wei, _ := floatWei.Int(nil)
	return wei
}
