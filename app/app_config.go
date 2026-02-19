package app

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	runtimev1alpha1 "cosmossdk.io/api/cosmos/app/runtime/v1alpha1"
	appv1alpha1 "cosmossdk.io/api/cosmos/app/v1alpha1"
	authmodulev1 "cosmossdk.io/api/cosmos/auth/module/v1"
	authzmodulev1 "cosmossdk.io/api/cosmos/authz/module/v1"
	bankmodulev1 "cosmossdk.io/api/cosmos/bank/module/v1"
	circuitmodulev1 "cosmossdk.io/api/cosmos/circuit/module/v1"
	consensusmodulev1 "cosmossdk.io/api/cosmos/consensus/module/v1"
	distrmodulev1 "cosmossdk.io/api/cosmos/distribution/module/v1"
	epochsmodulev1 "cosmossdk.io/api/cosmos/epochs/module/v1"
	evidencemodulev1 "cosmossdk.io/api/cosmos/evidence/module/v1"
	feegrantmodulev1 "cosmossdk.io/api/cosmos/feegrant/module/v1"
	genutilmodulev1 "cosmossdk.io/api/cosmos/genutil/module/v1"
	govmodulev1 "cosmossdk.io/api/cosmos/gov/module/v1"
	groupmodulev1 "cosmossdk.io/api/cosmos/group/module/v1"
	mintmodulev1 "cosmossdk.io/api/cosmos/mint/module/v1"
	nftmodulev1 "cosmossdk.io/api/cosmos/nft/module/v1"
	protocolpoolmodulev1 "cosmossdk.io/api/cosmos/protocolpool/module/v1"
	slashingmodulev1 "cosmossdk.io/api/cosmos/slashing/module/v1"
	stakingmodulev1 "cosmossdk.io/api/cosmos/staking/module/v1"
	txconfigv1 "cosmossdk.io/api/cosmos/tx/config/v1"
	upgrademodulev1 "cosmossdk.io/api/cosmos/upgrade/module/v1"
	vestingmodulev1 "cosmossdk.io/api/cosmos/vesting/module/v1"
	"cosmossdk.io/core/appconfig"
	"cosmossdk.io/depinject"
	_ "cosmossdk.io/x/circuit"
	circuittypes "cosmossdk.io/x/circuit/types"
	_ "cosmossdk.io/x/evidence"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	_ "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/nft"
	_ "cosmossdk.io/x/nft/module"
	_ "cosmossdk.io/x/upgrade"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"
	_ "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	_ "github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	_ "github.com/cosmos/cosmos-sdk/x/authz/module"
	_ "github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	_ "github.com/cosmos/cosmos-sdk/x/consensus"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	_ "github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	_ "github.com/cosmos/cosmos-sdk/x/epochs"
	epochstypes "github.com/cosmos/cosmos-sdk/x/epochs/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	_ "github.com/cosmos/cosmos-sdk/x/group/module"
	_ "github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	_ "github.com/cosmos/cosmos-sdk/x/protocolpool"
	protocolpooltypes "github.com/cosmos/cosmos-sdk/x/protocolpool/types"
	_ "github.com/cosmos/cosmos-sdk/x/slashing"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	_ "github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var (
	// moduleAccPerms defines module account permissions for QoreChain.
	moduleAccPerms = []*authmodulev1.ModuleAccountPermission{
		{Account: authtypes.FeeCollectorName},
		{Account: distrtypes.ModuleName},
		{Account: minttypes.ModuleName, Permissions: []string{authtypes.Minter}},
		{Account: stakingtypes.BondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: stakingtypes.NotBondedPoolName, Permissions: []string{authtypes.Burner, stakingtypes.ModuleName}},
		{Account: govtypes.ModuleName, Permissions: []string{authtypes.Burner}},
		{Account: nft.ModuleName},
		{Account: protocolpooltypes.ModuleName},
		{Account: protocolpooltypes.ProtocolPoolEscrowAccount},
	}

	// blockAccAddrs defines blocked module account addresses.
	blockAccAddrs = []string{
		authtypes.FeeCollectorName,
		distrtypes.ModuleName,
		minttypes.ModuleName,
		stakingtypes.BondedPoolName,
		stakingtypes.NotBondedPoolName,
		nft.ModuleName,
	}

	// ModuleConfig defines the standard QoreChain SDK module configuration for QoreChain.
	// Custom modules (x/pqc, x/ai, x/reputation, x/qca) will be registered manually
	// after the depinject phase since they require special initialization.
	ModuleConfig = []*appv1alpha1.ModuleConfig{
		{
			Name: runtime.ModuleName,
			Config: appconfig.WrapAny(&runtimev1alpha1.Module{
				AppName: "QoreChain",
				PreBlockers: []string{
					upgradetypes.ModuleName,
					authtypes.ModuleName,
				},
				BeginBlockers: []string{
					minttypes.ModuleName,
					distrtypes.ModuleName,
					protocolpooltypes.ModuleName,
					slashingtypes.ModuleName,
					evidencetypes.ModuleName,
					stakingtypes.ModuleName,
					authz.ModuleName,
					epochstypes.ModuleName,
				},
				EndBlockers: []string{
					govtypes.ModuleName,
					stakingtypes.ModuleName,
					feegrant.ModuleName,
					group.ModuleName,
					protocolpooltypes.ModuleName,
				},
				OverrideStoreKeys: []*runtimev1alpha1.StoreKeyConfig{
					{
						ModuleName: authtypes.ModuleName,
						KvStoreKey: "acc",
					},
				},
				SkipStoreKeys: []string{
					"tx",
				},
				InitGenesis: []string{
					authtypes.ModuleName,
					banktypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					slashingtypes.ModuleName,
					govtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					evidencetypes.ModuleName,
					authz.ModuleName,
					feegrant.ModuleName,
					nft.ModuleName,
					group.ModuleName,
					upgradetypes.ModuleName,
					vestingtypes.ModuleName,
					circuittypes.ModuleName,
					epochstypes.ModuleName,
					protocolpooltypes.ModuleName,
					// QoreChain custom modules
					"pqc",
					"ai",
					"reputation",
					"qca",
				},
				ExportGenesis: []string{
					consensustypes.ModuleName,
					authtypes.ModuleName,
					protocolpooltypes.ModuleName,
					banktypes.ModuleName,
					distrtypes.ModuleName,
					stakingtypes.ModuleName,
					slashingtypes.ModuleName,
					govtypes.ModuleName,
					minttypes.ModuleName,
					genutiltypes.ModuleName,
					evidencetypes.ModuleName,
					authz.ModuleName,
					feegrant.ModuleName,
					nft.ModuleName,
					group.ModuleName,
					upgradetypes.ModuleName,
					vestingtypes.ModuleName,
					circuittypes.ModuleName,
					epochstypes.ModuleName,
					// QoreChain custom modules
					"pqc",
					"ai",
					"reputation",
					"qca",
				},
			}),
		},
		{
			Name: authtypes.ModuleName,
			Config: appconfig.WrapAny(&authmodulev1.Module{
				Bech32Prefix:             "qor",
				ModuleAccountPermissions: moduleAccPerms,
			}),
		},
		{
			Name:   vestingtypes.ModuleName,
			Config: appconfig.WrapAny(&vestingmodulev1.Module{}),
		},
		{
			Name: banktypes.ModuleName,
			Config: appconfig.WrapAny(&bankmodulev1.Module{
				BlockedModuleAccountsOverride: blockAccAddrs,
			}),
		},
		{
			Name: stakingtypes.ModuleName,
			Config: appconfig.WrapAny(&stakingmodulev1.Module{
				Bech32PrefixValidator: "qorvaloper",
				Bech32PrefixConsensus: "qorvalcons",
			}),
		},
		{
			Name:   slashingtypes.ModuleName,
			Config: appconfig.WrapAny(&slashingmodulev1.Module{}),
		},
		{
			Name: "tx",
			Config: appconfig.WrapAny(&txconfigv1.Config{
				SkipAnteHandler: true,
			}),
		},
		{
			Name:   genutiltypes.ModuleName,
			Config: appconfig.WrapAny(&genutilmodulev1.Module{}),
		},
		{
			Name:   authz.ModuleName,
			Config: appconfig.WrapAny(&authzmodulev1.Module{}),
		},
		{
			Name:   upgradetypes.ModuleName,
			Config: appconfig.WrapAny(&upgrademodulev1.Module{}),
		},
		{
			Name:   distrtypes.ModuleName,
			Config: appconfig.WrapAny(&distrmodulev1.Module{}),
		},
		{
			Name:   evidencetypes.ModuleName,
			Config: appconfig.WrapAny(&evidencemodulev1.Module{}),
		},
		{
			Name:   minttypes.ModuleName,
			Config: appconfig.WrapAny(&mintmodulev1.Module{}),
		},
		{
			Name: group.ModuleName,
			Config: appconfig.WrapAny(&groupmodulev1.Module{
				MaxExecutionPeriod: durationpb.New(time.Second * 1209600),
				MaxMetadataLen:     255,
			}),
		},
		{
			Name:   nft.ModuleName,
			Config: appconfig.WrapAny(&nftmodulev1.Module{}),
		},
		{
			Name:   feegrant.ModuleName,
			Config: appconfig.WrapAny(&feegrantmodulev1.Module{}),
		},
		{
			Name:   govtypes.ModuleName,
			Config: appconfig.WrapAny(&govmodulev1.Module{}),
		},
		{
			Name:   consensustypes.ModuleName,
			Config: appconfig.WrapAny(&consensusmodulev1.Module{}),
		},
		{
			Name:   circuittypes.ModuleName,
			Config: appconfig.WrapAny(&circuitmodulev1.Module{}),
		},
		{
			Name:   epochstypes.ModuleName,
			Config: appconfig.WrapAny(&epochsmodulev1.Module{}),
		},
		{
			Name:   protocolpooltypes.ModuleName,
			Config: appconfig.WrapAny(&protocolpoolmodulev1.Module{}),
		},
	}

	// AppConfig is the full application configuration including module wiring
	// and dependency injection supplies.
	AppConfig = depinject.Configs(appconfig.Compose(&appv1alpha1.Config{
		Modules: ModuleConfig,
	}),
		depinject.Supply(
			map[string]module.AppModuleBasic{
				genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
				govtypes.ModuleName: gov.NewAppModuleBasic(
					[]govclient.ProposalHandler{},
				),
			},
		),
	)
)
