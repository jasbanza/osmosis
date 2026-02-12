package blockprocessor

import (
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"

	commondomain "github.com/osmosis-labs/osmosis/v31/ingest/common/domain"
	"github.com/osmosis-labs/osmosis/v31/ingest/sqs/domain"
)

type fullSQSBlockProcessStrategy struct {
	sqsGRPCClient domain.SQSGRPClient

	poolExtractor    commondomain.PoolExtractor
	poolsTransformer domain.PoolsTransformer

	nodeStatusChecker domain.NodeStatusChecker

	transformAndLoadFunc transformAndLoadFunc
}

// IsFullBlockProcessor implements commondomain.BlockProcessor.
func (f *fullSQSBlockProcessStrategy) IsFullBlockProcessor() bool {
	return true
}

var _ commondomain.BlockProcessor = &fullSQSBlockProcessStrategy{}

// ProcessBlock implements commondomain.BlockProcessStrategy.
// MODIFIED: Sync check removed. Halts after successful SQS push.
func (f *fullSQSBlockProcessStrategy) ProcessBlock(ctx sdk.Context) (err error) {
	ctx.Logger().Info("SQS snapshot ingestor: skipping sync check, processing block", "height", ctx.BlockHeight())

	pools, _, err := f.poolExtractor.ExtractAll(ctx)
	if err != nil {
		return err
	}

	numPools := len(pools.ConcentratedPools) + len(pools.CFMMPools) + len(pools.CosmWasmPools)
	ctx.Logger().Info("SQS snapshot ingestor: extracted pools",
		"concentrated", len(pools.ConcentratedPools),
		"cfmm", len(pools.CFMMPools),
		"cosmwasm", len(pools.CosmWasmPools),
		"total", numPools,
		"height", ctx.BlockHeight(),
	)

	// Publish the pools
	err = f.transformAndLoadFunc(ctx, f.poolsTransformer, f.sqsGRPCClient, pools)
	if err != nil {
		return err
	}

	ctx.Logger().Info("========================================")
	ctx.Logger().Info("SQS INGEST COMPLETE")
	ctx.Logger().Info("All pool data pushed successfully. Halting.", "height", ctx.BlockHeight(), "total_pools", numPools)
	ctx.Logger().Info("========================================")
	os.Exit(0)

	return nil
}