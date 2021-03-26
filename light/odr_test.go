// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package light

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/celo-org/celo-blockchain/common"
	"github.com/celo-org/celo-blockchain/common/math"
	mockEngine "github.com/celo-org/celo-blockchain/consensus/consensustest"
	"github.com/celo-org/celo-blockchain/core"
	"github.com/celo-org/celo-blockchain/core/rawdb"
	"github.com/celo-org/celo-blockchain/core/state"
	"github.com/celo-org/celo-blockchain/core/types"
	"github.com/celo-org/celo-blockchain/core/vm"
	"github.com/celo-org/celo-blockchain/crypto"
	"github.com/celo-org/celo-blockchain/ethdb"
	"github.com/celo-org/celo-blockchain/params"
	"github.com/celo-org/celo-blockchain/rlp"
	"github.com/celo-org/celo-blockchain/trie"
)

var (
	testBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(100000000)

	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)

	testContractCode = common.Hex2Bytes("606060405260cc8060106000396000f360606040526000357c01000000000000000000000000000000000000000000000000000000009004806360cd2685146041578063c16431b914606b57603f565b005b6055600480803590602001909190505060a9565b6040518082815260200191505060405180910390f35b60886004808035906020019091908035906020019091905050608a565b005b80600060005083606481101560025790900160005b50819055505b5050565b6000600060005082606481101560025790900160005b5054905060c7565b91905056")
	testContractAddr common.Address
)

type testOdr struct {
	OdrBackend
	indexerConfig *IndexerConfig
	sdb, ldb      ethdb.Database
	disable       bool
}

func (odr *testOdr) ChtIndexer() *core.ChainIndexer {
	return nil
}

func (odr *testOdr) Database() ethdb.Database {
	return odr.ldb
}

var ErrOdrDisabled = errors.New("ODR disabled")

func (odr *testOdr) Retrieve(ctx context.Context, req OdrRequest) error {
	if odr.disable {
		return ErrOdrDisabled
	}
	switch req := req.(type) {
	case *BlockRequest:
		number := rawdb.ReadHeaderNumber(odr.sdb, req.Hash)
		if number != nil {
			req.Rlp = rawdb.ReadBodyRLP(odr.sdb, req.Hash, *number)
		}
	case *HeaderRequest:
		if req.Origin.Hash != (common.Hash{}) {
			number := rawdb.ReadHeaderNumber(odr.sdb, req.Origin.Hash)
			req.Header = rawdb.ReadHeader(odr.sdb, req.Origin.Hash, *number)
		} else {
			hash := rawdb.ReadCanonicalHash(odr.sdb, *req.Origin.Number)
			req.Header = rawdb.ReadHeader(odr.sdb, hash, *req.Origin.Number)
		}
		// Simulate `InsertHeaderChain` call that would be done in the real `odr`
		rawdb.WriteHeader(odr.ldb, req.Header)
		rawdb.WriteTd(odr.ldb, req.Header.Hash(), req.Header.Number.Uint64(), big.NewInt(int64(req.Header.Number.Uint64()+1)))
	case *ReceiptsRequest:
		number := rawdb.ReadHeaderNumber(odr.sdb, req.Hash)
		if number != nil {
			req.Receipts = rawdb.ReadRawReceipts(odr.sdb, req.Hash, *number)
		}
	case *TrieRequest:
		t, _ := trie.New(req.Id.Root, trie.NewDatabase(odr.sdb))
		nodes := NewNodeSet()
		t.Prove(req.Key, 0, nodes)
		req.Proof = nodes
	case *CodeRequest:
		req.Data, _ = odr.sdb.Get(req.Hash[:])
	}
	req.StoreResult(odr.ldb)
	return nil
}

func (odr *testOdr) IndexerConfig() *IndexerConfig {
	return odr.indexerConfig
}

type odrTestFn func(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) ([]byte, error)
type odrTestFnNum func(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, origin blockHashOrNumber) ([]byte, error)

func TestOdrGetBlockLes2(t *testing.T) { testChainOdr(t, 1, odrGetBlock) }

func TestOdrGetBlockLightest(t *testing.T) { testLightestChainOdr(t, 1, odrGetBlockHashOrNumber) }

func odrGetBlock(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) ([]byte, error) {
	var block *types.Block
	var err error
	if bc != nil {
		block = bc.GetBlockByHash(bhash)
	} else {
		block, err = lc.GetBlockByHash(ctx, bhash)
		if err != nil {
			return nil, err
		}
	}
	if block == nil {
		return nil, nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp, nil
}

func odrGetBlockHashOrNumber(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, origin blockHashOrNumber) ([]byte, error) {
	var block *types.Block
	var err error
	if bc != nil {
		if origin.Hash != (common.Hash{}) {
			block = bc.GetBlockByHash(origin.Hash)
		} else {
			block = bc.GetBlockByNumber(*origin.Number)
		}
	} else {
		if origin.Hash != (common.Hash{}) {
			block, err = lc.GetBlockByHash(ctx, origin.Hash)
		} else {
			block, err = lc.GetBlockByNumber(ctx, *origin.Number)
		}
		if err != nil {
			return nil, err
		}
	}
	if block == nil {
		return nil, nil
	}
	rlp, _ := rlp.EncodeToBytes(block)
	return rlp, nil
}

func TestOdrGetReceiptsLes2(t *testing.T) { testChainOdr(t, 1, odrGetReceipts) }

func odrGetReceipts(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) ([]byte, error) {
	var receipts types.Receipts
	if bc != nil {
		number := rawdb.ReadHeaderNumber(db, bhash)
		if number != nil {
			receipts = rawdb.ReadReceipts(db, bhash, *number, bc.Config())
		}
	} else {
		number := rawdb.ReadHeaderNumber(db, bhash)
		if number != nil {
			receipts, _ = GetBlockReceipts(ctx, lc.Odr(), bhash, *number)
		}
	}
	if receipts == nil {
		return nil, nil
	}
	rlp, _ := rlp.EncodeToBytes(receipts)
	return rlp, nil
}

func TestOdrAccountsLes2(t *testing.T) { testChainOdr(t, 1, odrAccounts) }

func odrAccounts(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) ([]byte, error) {
	dummyAddr := common.HexToAddress("1234567812345678123456781234567812345678")
	acc := []common.Address{testBankAddress, acc1Addr, acc2Addr, dummyAddr}

	var st *state.StateDB
	if bc == nil {
		header := lc.GetHeaderByHash(bhash)
		st = NewState(ctx, header, lc.Odr())
	} else {
		header := bc.GetHeaderByHash(bhash)
		st, _ = state.New(header.Root, state.NewDatabase(db), nil)
	}

	var res []byte
	for _, addr := range acc {
		bal := st.GetBalance(addr)
		rlp, _ := rlp.EncodeToBytes(bal)
		res = append(res, rlp...)
	}
	return res, st.Error()
}

func TestOdrContractCallLes2(t *testing.T) { testChainOdr(t, 1, odrContractCall) }

type callmsg struct {
	types.Message
}

func (callmsg) CheckNonce() bool { return false }

func odrContractCall(ctx context.Context, db ethdb.Database, bc *core.BlockChain, lc *LightChain, bhash common.Hash) ([]byte, error) {
	data := common.Hex2Bytes("60CD26850000000000000000000000000000000000000000000000000000000000000000")
	config := params.IstanbulTestChainConfig

	var res []byte
	for i := 0; i < 3; i++ {
		data[35] = byte(i)

		var (
			st     *state.StateDB
			header *types.Header
			chain  vm.ChainContext
		)
		if bc == nil {
			chain = lc
			header = lc.GetHeaderByHash(bhash)
			st = NewState(ctx, header, lc.Odr())
		} else {
			chain = bc
			header = bc.GetHeaderByHash(bhash)
			st, _ = state.New(header.Root, state.NewDatabase(db), nil)
		}

		// Perform read-only call.
		st.SetBalance(testBankAddress, math.MaxBig256)
		msg := callmsg{types.NewMessage(testBankAddress, &testContractAddr, 0, new(big.Int), 1000000, new(big.Int), nil, nil, new(big.Int), data, false, false)}
		context := vm.NewEVMContext(msg, header, chain, nil)
		vmenv := vm.NewEVM(context, st, config, vm.Config{})
		gp := new(core.GasPool).AddGas(math.MaxUint64)
		result, _ := core.ApplyMessage(vmenv, msg, gp)
		res = append(res, result.Return()...)
		if st.Error() != nil {
			return res, st.Error()
		}
	}
	return res, nil
}

func testChainGen(i int, block *core.BlockGen) {
	signer := types.HomesteadSigner{}
	switch i {
	case 0:
		// In block 1, the test bank sends account #1 some ether.
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(10000), params.TxGas, nil, nil, nil, nil, nil), signer, testBankKey)
		block.AddTx(tx)
	case 1:
		// In block 2, the test bank sends some more ether to account #1.
		// acc1Addr passes it on to account #2.
		// acc1Addr creates a test contract.
		tx1, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), acc1Addr, big.NewInt(1000), params.TxGas, nil, nil, nil, nil, nil), signer, testBankKey)
		nonce := block.TxNonce(acc1Addr)
		tx2, _ := types.SignTx(types.NewTransaction(nonce, acc2Addr, big.NewInt(1000), params.TxGas, nil, nil, nil, nil, nil), signer, acc1Key)
		nonce++
		tx3, _ := types.SignTx(types.NewContractCreation(nonce, big.NewInt(0), 1000000, big.NewInt(0), nil, nil, nil, testContractCode), signer, acc1Key)
		testContractAddr = crypto.CreateAddress(acc1Addr, nonce)
		block.AddTx(tx1)
		block.AddTx(tx2)
		block.AddTx(tx3)
	case 2:
		// Block 3 is empty but was mined by account #2.
		block.SetCoinbase(acc2Addr)
		block.SetExtra([]byte("yeehaw"))
		data := common.Hex2Bytes("C16431B900000000000000000000000000000000000000000000000000000000000000010000000000000000000000000000000000000000000000000000000000000001")
		tx, _ := types.SignTx(types.NewTransaction(block.TxNonce(testBankAddress), testContractAddr, big.NewInt(0), 100000, nil, nil, nil, nil, data), signer, testBankKey)
		block.AddTx(tx)
	}
}

func testChainOdr(t *testing.T, protocol int, fn odrTestFn) {
	var (
		sdb     = rawdb.NewMemoryDatabase()
		ldb     = rawdb.NewMemoryDatabase()
		gspec   = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
		genesis = gspec.MustCommit(sdb)
	)
	gspec.MustCommit(ldb)
	// Assemble the test environment
	blockchain, _ := core.NewBlockChain(sdb, nil, params.IstanbulTestChainConfig, mockEngine.NewFullFaker(), vm.Config{}, nil, nil)
	gchain, _ := core.GenerateChain(params.IstanbulTestChainConfig, genesis, mockEngine.NewFaker(), sdb, 4, testChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		t.Fatal(err)
	}

	odr := &testOdr{sdb: sdb, ldb: ldb, indexerConfig: TestClientIndexerConfig}
	lightchain, err := NewLightChain(odr, params.IstanbulTestChainConfig, mockEngine.NewFullFaker(), nil)
	if err != nil {
		t.Fatal(err)
	}
	headers := make([]*types.Header, len(gchain))
	for i, block := range gchain {
		headers[i] = block.Header()
	}
	if _, err := lightchain.InsertHeaderChain(headers, 1, true); err != nil {
		t.Fatal(err)
	}

	test := func(expFail int) {
		for i := uint64(0); i <= blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(sdb, i)
			b1, err := fn(NoOdr, sdb, blockchain, nil, bhash)
			if err != nil {
				t.Fatalf("error in full-node test for block %d: %v", i, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			exp := i < uint64(expFail)
			b2, err := fn(ctx, ldb, nil, lightchain, bhash)
			if err != nil && exp {
				t.Errorf("error in ODR test for block %d: %v", i, err)
			}

			eq := bytes.Equal(b1, b2)
			if exp && !eq {
				t.Errorf("ODR test output for block %d doesn't match full node", i)
			}
		}
	}

	// expect retrievals to fail (except genesis block) without a les peer
	t.Log("checking without ODR")
	odr.disable = true
	test(1)

	// expect all retrievals to pass with ODR enabled
	t.Log("checking with ODR")
	odr.disable = false
	test(len(gchain))

	// still expect all retrievals to pass, now data should be cached locally
	t.Log("checking without ODR, should be cached")
	odr.disable = true
	test(len(gchain))
}

func testLightestChainOdr(t *testing.T, protocol int, fn odrTestFnNum) {
	var (
		sdb     = rawdb.NewMemoryDatabase()
		ldb     = rawdb.NewMemoryDatabase()
		gspec   = core.Genesis{Alloc: core.GenesisAlloc{testBankAddress: {Balance: testBankFunds}}}
		genesis = gspec.MustCommit(sdb)
	)
	gspec.MustCommit(ldb)
	// Assemble the test environment
	blockchain, _ := core.NewBlockChain(sdb, nil, params.IstanbulTestChainConfig, mockEngine.NewFullFaker(), vm.Config{}, nil)
	gchain, _ := core.GenerateChain(params.IstanbulTestChainConfig, genesis, mockEngine.NewFaker(), sdb, 4, testChainGen)
	if _, err := blockchain.InsertChain(gchain); err != nil {
		t.Fatal(err)
	}

	odr := &testOdr{sdb: sdb, ldb: ldb, indexerConfig: TestClientIndexerConfig}
	lightchain, err := NewLightChain(odr, params.IstanbulTestChainConfig, mockEngine.NewFullFaker(), nil)
	if err != nil {
		t.Fatal(err)
	}

	test := func(expFail int, tryHash bool) {
		for i := uint64(0); i <= blockchain.CurrentHeader().Number.Uint64(); i++ {
			bhash := rawdb.ReadCanonicalHash(sdb, i)
			var origin blockHashOrNumber
			if tryHash {
				origin = blockHashOrNumber{Hash: bhash}
			} else {
				origin = blockHashOrNumber{Number: &i}
			}
			b1, err := fn(NoOdr, sdb, blockchain, nil, origin)
			if err != nil {
				t.Fatalf("error in full-node test for block %d: %v", i, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			exp := i < uint64(expFail)
			b2, err := fn(ctx, ldb, nil, lightchain, origin)
			if err != nil && exp {
				t.Errorf("error in ODR test for block %d: %v", i, err)
			}

			eq := bytes.Equal(b1, b2)
			if exp && !eq {
				t.Errorf("ODR test output for tryHash %t block %d doesn't match full node", tryHash, i)
			}

			header := rawdb.ReadHeader(odr.ldb, bhash, i)
			if exp {
				if header == nil {
					t.Errorf("ODR test for tryHash %t block %d did not properly receive/store header", tryHash, i)
				}
				headers := []*types.Header{header}

				if _, err := lightchain.InsertHeaderChain(headers, 1, true); err != nil {
					t.Errorf("ODR test for tryHash %t block %d could not insert header to headerchain", tryHash, i)
				}
			}
		}
	}
	// expect retrievals to fail (except genesis block) without a les peer
	t.Log("checking without ODR")
	odr.disable = true
	test(1, true)
	test(1, false)

	// expect all retrievals to pass with ODR enabled
	t.Log("checking with ODR")
	odr.disable = false
	test(len(gchain), true)
	test(len(gchain), false)

	// still expect all retrievals to pass, now data should be cached locally
	t.Log("checking without ODR, should be cached")
	odr.disable = true
	test(len(gchain), true)
	test(len(gchain), false)
}
