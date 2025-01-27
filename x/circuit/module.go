package circuit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/registry"
	"cosmossdk.io/x/circuit/keeper"
	"cosmossdk.io/x/circuit/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// ConsensusVersion defines the current circuit module consensus version.
const ConsensusVersion = 1

var (
	_ module.HasName        = AppModule{}
	_ module.HasGRPCGateway = AppModule{}

	_ appmodule.AppModule             = AppModule{}
	_ appmodule.HasServices           = AppModule{}
	_ appmodule.HasGenesis            = AppModule{}
	_ appmodule.HasRegisterInterfaces = AppModule{}
)

// AppModule implements an application module for the circuit module.
type AppModule struct {
	cdc    codec.Codec
	keeper keeper.Keeper
}

// IsAppModule implements the appmodule.AppModule interface.
func (AppModule) IsAppModule() {}

// Name returns the circuit module's name.
func (AppModule) Name() string { return types.ModuleName }

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the circuit module.
func (AppModule) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *gwruntime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// RegisterInterfaces registers interfaces and implementations of the circuit module.
func (AppModule) RegisterInterfaces(registry registry.LegacyRegistry) {
	types.RegisterInterfaces(registry)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(registrar grpc.ServiceRegistrar) error {
	types.RegisterMsgServer(registrar, keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(registrar, keeper.NewQueryServer(am.keeper))

	return nil
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper) AppModule {
	return AppModule{
		cdc:    cdc,
		keeper: keeper,
	}
}

// ConsensusVersion implements HasConsensusVersion
func (AppModule) ConsensusVersion() uint64 { return ConsensusVersion }

// DefaultGenesis returns default genesis state as raw bytes for the circuit module.
func (am AppModule) DefaultGenesis() json.RawMessage {
	return am.cdc.MustMarshalJSON(types.DefaultGenesisState())
}

// ValidateGenesis performs genesis state validation for the circuit module.
func (am AppModule) ValidateGenesis(bz json.RawMessage) error {
	var data types.GenesisState
	if err := am.cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return data.Validate()
}

// InitGenesis performs genesis initialization for the circuit module.
func (am AppModule) InitGenesis(ctx context.Context, data json.RawMessage) error {
	start := time.Now()
	var genesisState types.GenesisState
	if err := am.cdc.UnmarshalJSON(data, &genesisState); err != nil {
		return err
	}
	telemetry.MeasureSince(start, "InitGenesis", "crisis", "unmarshal")

	return am.keeper.InitGenesis(ctx, &genesisState)
}

// ExportGenesis returns the exported genesis state as raw bytes for the circuit
// module.
func (am AppModule) ExportGenesis(ctx context.Context) (json.RawMessage, error) {
	gs, err := am.keeper.ExportGenesis(ctx)
	if err != nil {
		return nil, err
	}
	return am.cdc.MarshalJSON(gs)
}
