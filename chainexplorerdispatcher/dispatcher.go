package chainexplorerdispatcher

import (
	"context"
	"fmt"

	"math/big"
	"runtime/debug"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/dapplink-labs/chain-explorer-api/common"
	"github.com/dapplink-labs/chain-explorer-api/common/account"
	"github.com/dapplink-labs/chain-explorer-api/common/chain"
	"github.com/dapplink-labs/chain-explorer-api/explorer"
	"github.com/dapplink-labs/chain-explorer-api/explorer/etherscan"
	"github.com/dapplink-labs/chain-explorer-api/explorer/oklink"
)

type CommonRequest interface {
	GetChainExporerName() string
}

type CommonReply = chain.SupportChainExplorerResponse

type ChainExplorerName = string

type ChainExplorerDispatcher struct {
	RegistryExplorer map[ChainExplorerName]explorer.ChainExplorerAdaptor
}

type ExplorerClientConfig struct {
	ExplorerName string
	Key          string
	BaseURL      string
	Verbose      bool
	Timeout      time.Duration
}

func NewExplorerClient(cliConfigs []ExplorerClientConfig) (*ChainExplorerDispatcher, error) {
	dispatcher := ChainExplorerDispatcher{
		RegistryExplorer: make(map[ChainExplorerName]explorer.ChainExplorerAdaptor),
	}
	for _, cliCfg := range cliConfigs {
		var adaptor explorer.ChainExplorerAdaptor
		var err error
		if cliCfg.ExplorerName == etherscan.ChainExplorerName {
			adaptor, err = oklink.NewChainExplorerAdaptor(cliCfg.Key, cliCfg.BaseURL, cliCfg.Verbose, cliCfg.Timeout)
			if err != nil {
				fmt.Println("failed to setup oklink", "chain", cliCfg, "error", err)
			}
		}
		if cliCfg.ExplorerName == oklink.ChainExplorerName {
			adaptor, err = oklink.NewChainExplorerAdaptor(cliCfg.Key, cliCfg.BaseURL, cliCfg.Verbose, cliCfg.Timeout)
			if err != nil {
				fmt.Println("failed to setup chain", "oklink", cliCfg, "error", err)
			}
		}
		dispatcher.RegistryExplorer[cliCfg.ExplorerName] = adaptor
	}
	return &dispatcher, nil
}

func (d *ChainExplorerDispatcher) Interceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Println("panic error", "msg", e)
			fmt.Println(string(debug.Stack()))
			err = status.Errorf(codes.Internal, "Panic err: %v", e)
		}
	}()

	pos := strings.LastIndex(info.FullMethod, "/")
	method := info.FullMethod[pos+1:]

	chainExplorer := req.(CommonRequest).GetChainExporerName()
	fmt.Println(method, "chain", chainExplorer, "req", req)

	resp, err = handler(ctx, req)
	fmt.Println("Finish handling", "resp", resp, "err", err)
	return
}

func (cea *ChainExplorerDispatcher) preHandler(req interface{}) (resp *CommonReply) {
	chainExporerName := req.(CommonRequest).GetChainExporerName()
	if _, ok := cea.RegistryExplorer[chainExporerName]; !ok {
		return &CommonReply{
			Ok: false,
		}
	}
	return nil
}

func (cea *ChainExplorerDispatcher) GetAccountBalance(request *account.AccountBalanceRequest) (*account.AccountBalanceResponse, error) {
	resp := cea.preHandler(request)
	if resp != nil {
		return &account.AccountBalanceResponse{
			Account: request.Account,
			Balance: (*common.BigInt)(big.NewInt(0)),
		}, nil
	}
	return cea.RegistryExplorer[request.ExplorerName].GetAccountBalance(request)
}
