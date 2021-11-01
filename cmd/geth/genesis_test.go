// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	registryAddress = "'0x000000000000000000000000000000000000ce10'"
	registryCode    = "0x60806040526004361061004a5760003560e01c806303386ba3146101e757806342404e0714610280578063bb913f41146102d7578063d29d44ee14610328578063f7e6af8014610379575b6000600160405180807f656970313936372e70726f78792e696d706c656d656e746174696f6e00000000815250601c019050604051809103902060001c0360001b9050600081549050600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610136576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260158152602001807f4e6f20496d706c656d656e746174696f6e20736574000000000000000000000081525060200191505060405180910390fd5b61013f816103d0565b6101b1576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f496e76616c696420636f6e74726163742061646472657373000000000000000081525060200191505060405180910390fd5b60405136810160405236600082376000803683855af43d604051818101604052816000823e82600081146101e3578282f35b8282fd5b61027e600480360360408110156101fd57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291908035906020019064010000000081111561023a57600080fd5b82018360208201111561024c57600080fd5b8035906020019184600183028401116401000000008311171561026e57600080fd5b909192939192939050505061041b565b005b34801561028c57600080fd5b506102956105c1565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b3480156102e357600080fd5b50610326600480360360208110156102fa57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff16906020019092919050505061060d565b005b34801561033457600080fd5b506103776004803603602081101561034b57600080fd5b81019080803573ffffffffffffffffffffffffffffffffffffffff1690602001909291905050506107bd565b005b34801561038557600080fd5b5061038e610871565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b60008060007fc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a47060001b9050833f915080821415801561041257506000801b8214155b92505050919050565b610423610871565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146104c3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f73656e64657220776173206e6f74206f776e657200000000000000000000000081525060200191505060405180910390fd5b6104cc8361060d565b600060608473ffffffffffffffffffffffffffffffffffffffff168484604051808383808284378083019250505092505050600060405180830381855af49150503d8060008114610539576040519150601f19603f3d011682016040523d82523d6000602084013e61053e565b606091505b508092508193505050816105ba576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040180806020018281038252601e8152602001807f696e697469616c697a6174696f6e2063616c6c6261636b206661696c6564000081525060200191505060405180910390fd5b5050505050565b600080600160405180807f656970313936372e70726f78792e696d706c656d656e746174696f6e00000000815250601c019050604051809103902060001c0360001b9050805491505090565b610615610871565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16146106b5576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f73656e64657220776173206e6f74206f776e657200000000000000000000000081525060200191505060405180910390fd5b6000600160405180807f656970313936372e70726f78792e696d706c656d656e746174696f6e00000000815250601c019050604051809103902060001c0360001b9050610701826103d0565b610773576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260188152602001807f496e76616c696420636f6e74726163742061646472657373000000000000000081525060200191505060405180910390fd5b8181558173ffffffffffffffffffffffffffffffffffffffff167fab64f92ab780ecbf4f3866f57cee465ff36c89450dcce20237ca7a8d81fb7d1360405160405180910390a25050565b6107c5610871565b73ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614610865576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260148152602001807f73656e64657220776173206e6f74206f776e657200000000000000000000000081525060200191505060405180910390fd5b61086e816108bd565b50565b600080600160405180807f656970313936372e70726f78792e61646d696e000000000000000000000000008152506013019050604051809103902060001c0360001b9050805491505090565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff161415610960576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004018080602001828103825260118152602001807f6f776e65722063616e6e6f74206265203000000000000000000000000000000081525060200191505060405180910390fd5b6000600160405180807f656970313936372e70726f78792e61646d696e000000000000000000000000008152506013019050604051809103902060001c0360001b90508181558173ffffffffffffffffffffffffffffffffffffffff167f50146d0e3c60aa1d17a70635b05494f864e86144a2201275021014fbf08bafe260405160405180910390a2505056fea165627a7a723058206808dd43e7d765afca53fe439122bc5eac16d708ce7d463451be5042426f101f0029"
)

// TestCustomGenesis tests that initializing Geth with a custom genesis block and chain definitions
// work properly.
func TestCustomGenesis(t *testing.T) {
	customGenesisTests := []struct {
		genesis string
		query   string
		result  string
	}{
		// Genesis file with an empty chain configuration (ensure missing fields work)
		// Note: We add Registry to genesis because it's required for initializing a eth node.
		{
			genesis: fmt.Sprintf(`{
			"alloc"      : {
				"000000000000000000000000000000000000ce10" : {
					"code":
						"%s",
					"storage":{"0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103":"47e172F6CfB6c7D01C1574fa3E2Be7CC73269D95"},
					"balance":"0"
				}
			},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"extraData"  : "",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0xabcdef",
			"config"     : {
				"istanbul": {}
			}
		}`, registryCode),
			query:  "eth.getBlock(0).timestamp",
			result: "11259375",
		},
		// Genesis file with specific chain configurations
		{
			genesis: fmt.Sprintf(`{
			"alloc"      : {
				"000000000000000000000000000000000000ce10" : {
					"code":
						"%s",
					"storage":{"0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103":"47e172F6CfB6c7D01C1574fa3E2Be7CC73269D95"},
					"balance":"0"
				}
			},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"extraData"  : "",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0xabcdf0",
			"config"     : {
				"homesteadBlock" : 42,
				"daoForkBlock"   : 141,
				"daoForkSupport" : true,
				"istanbul": {}
			}
		}`, registryCode),
			query:  "eth.getBlock(0).timestamp",
			result: "11259376",
		},
		// Genesis file with an empty chain configuration, and a deployed registry
		{
			genesis: fmt.Sprintf(`{
			"alloc"      : {
				"000000000000000000000000000000000000ce10" : {
					"code":
						"%s",
					"storage":{"0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103":"47e172F6CfB6c7D01C1574fa3E2Be7CC73269D95"},
					"balance":"0"
				}
			},
			"coinbase"   : "0x0000000000000000000000000000000000000000",
			"extraData"  : "",
			"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
			"timestamp"  : "0xabcdf0",
			"config"     : {
				"istanbul": {}
			}
		}`, registryCode),
			query:  "eth.getCode(" + registryAddress + ")",
			result: registryCode,
		},
		// Genesis file without Registry deployed
		{
			// nolint:gosimple
			genesis: fmt.Sprintf(`{
				"alloc"      : {},
				"coinbase"   : "0x0000000000000000000000000000000000000000",
				"extraData"  : "",
				"parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
				"timestamp"  : "0xabcdf0",
				"config"     : {
					"homesteadBlock" : 42,
					"daoForkBlock"   : 141,
					"daoForkSupport" : true,
					"istanbul": {}
				}
			}`),
			query:  "", // No need, initializing node is supposed to fail
			result: "no Registry Smart Contract deployed in genesis",
		},
	}

	for i, tt := range customGenesisTests {
		// Create a temporary data directory to use and inspect later
		datadir := tmpdir(t)
		defer os.RemoveAll(datadir)

		// Initialize the data directory with the custom genesis block
		json := filepath.Join(datadir, "genesis.json")
		if err := ioutil.WriteFile(json, []byte(tt.genesis), 0600); err != nil {
			t.Fatalf("test %d: failed to write genesis file: %v", i, err)
		}
		runGeth(t, "--datadir", datadir, "init", json).WaitExit()

		// Query the custom genesis block
		// geth := runGeth(t, "--networkid", "1337", "--syncmode=full", "--cache", "16", // 1.10.7
		// 	"--datadir", datadir, "--maxpeers", "0", "--port", "0",
		geth := runGeth(t, "--nousb", "--networkid", "1337", "--syncmode=full",
			"--datadir", datadir, "--maxpeers", "0", "--port", "0", "--light.maxpeers", "0",

			"--nodiscover", "--nat", "none", "--ipcdisable",
			"--exec", tt.query, "console")
		geth.ExpectRegexp(tt.result)
		geth.ExpectExit()
	}
}

// TestRegistryInGenesis tests that initializing Geth with a default genesis block(mainnet genesis)
// Expects registry contract is deployed.
func TestRegistryInGenesis(t *testing.T) {
	query := fmt.Sprintf("eth.getCode(%s)", registryAddress)

	// Query the custom genesis block
	geth := runGeth(t, "--maxpeers", "0", "--port", "0", "--light.maxpeers", "0",
		"--nodiscover", "--nat", "none", "--ipcdisable",
		"--exec", query, "console")
	defer geth.Cleanup()
	geth.ExpectRegexp(registryCode)
	geth.ExpectExit()
}
