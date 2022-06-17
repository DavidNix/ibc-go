package setup

import (
	"context"
	"fmt"
	"testing"

	"github.com/cosmos/ibc-go/v3/e2e/testconfig"
	"github.com/strangelove-ventures/ibctest"
	"github.com/strangelove-ventures/ibctest/chain/cosmos"
	"github.com/strangelove-ventures/ibctest/ibc"
	"github.com/strangelove-ventures/ibctest/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// StandardTwoChainEnvironment creates two default simapp containers as well as a go relayer container.
// the relayer that is returned is not yet started.
func StandardTwoChainEnvironment(t *testing.T, req *require.Assertions, eRep *testreporter.RelayerExecReporter, optFuncs ...ConfigurationFunc) (*cosmos.CosmosChain, *cosmos.CosmosChain, ibc.Relayer) {
	opts := defaultSetupOpts()
	for _, fn := range optFuncs {
		fn(opts)
	}

	ctx := context.Background()
	pool, network := ibctest.DockerSetup(t)
	home := t.TempDir() // Must be before chain cleanup to avoid test error during cleanup.

	logger := zaptest.NewLogger(t)
	chain1 := cosmos.NewCosmosChain(t.Name(), *opts.ChainAConfig, 1, 1, logger)
	chain2 := cosmos.NewCosmosChain(t.Name(), *opts.ChainBConfig, 1, 1, logger)

	t.Cleanup(func() {
		for _, c := range []*cosmos.CosmosChain{chain1, chain2} {
			if err := c.Cleanup(ctx); err != nil {
				t.Logf("Chain cleanup for %s failed: %v", c.Config().ChainID, err)
			}
		}
	})

	r := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, logger).Build(
		t, pool, network, home,
	)

	ic := ibctest.NewInterchain().
		AddChain(chain1).
		AddChain(chain2).
		AddRelayer(r, "r").
		AddLink(ibctest.InterchainLink{
			Chain1:  chain1,
			Chain2:  chain2,
			Relayer: r,
			Path:    testconfig.TestPath,
		})

	req.NoError(ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:  t.Name(),
		HomeDir:   home,
		Pool:      pool,
		NetworkID: network,
	}))

	return chain1, chain2, r
}

// ConfigurationFunc allows for arbitrary configuration of the setup Options.
type ConfigurationFunc func(opts *Options)

// Options holds values that allow for configuring setup functions.
type Options struct {
	ChainAConfig *ibc.ChainConfig
	ChainBConfig *ibc.ChainConfig
}

func defaultSetupOpts() *Options {
	chainAConfig := NewSimappConfig("simapp-a", "chain-a", "atoma")
	chainBConfig := NewSimappConfig("simapp-b", "chain-b", "atomb")
	return &Options{
		ChainAConfig: &chainAConfig,
		ChainBConfig: &chainBConfig,
	}
}

// NewSimappConfig creates an ibc configuration for simd.
func NewSimappConfig(name, chainId, denom string) ibc.ChainConfig {
	tc := testconfig.FromEnv()
	return ibc.ChainConfig{
		Type:    "cosmos",
		Name:    name,
		ChainID: chainId,
		Images: []ibc.DockerImage{
			{
				Repository: tc.SimdImage,
				Version:    tc.SimdTag,
			},
		},
		Bin:            "simd",
		Bech32Prefix:   "cosmos",
		Denom:          denom,
		GasPrices:      fmt.Sprintf("0.01%s", denom),
		GasAdjustment:  1.3,
		TrustingPeriod: "508h",
		NoHostMount:    false,
	}
}
