package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/lyswifter/api"
	"github.com/lyswifter/full"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"golang.org/x/xerrors"
)

func main() {
	ipaddr := flag.String("ip", "127.0.0.1", "ip address specify")
	port := flag.String("port", "1234", "port specify")

	flag.Parse()

	ml, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s/http", *ipaddr, *port))
	if err != nil {
		return
	}

	err = serveRPC(&full.FullNodeAPI{}, ml, 512)
	if err != nil {
		return
	}
}

func serveRPC(a api.FullApi, addr multiaddr.Multiaddr, maxRequestSize int64) error {
	serverOptions := make([]jsonrpc.ServerOption, 0)
	if maxRequestSize != 0 { // config set
		serverOptions = append(serverOptions, jsonrpc.WithMaxRequestSize(maxRequestSize))
	}
	rpcServer := jsonrpc.NewServer(serverOptions...)
	rpcServer.Register("MultiRPC", api.PermissionedFullAPI(a))
	rpcServer.AliasMethod("rpc.discover", "Filecoin.Discover")

	ah := &auth.Handler{
		Verify: a.AuthVerify,
		Next:   rpcServer.ServeHTTP,
	}

	http.Handle("/rpc/v0", ah)

	lst, err := manet.Listen(addr)
	if err != nil {
		return xerrors.Errorf("could not listen: %w", err)
	}

	srv := &http.Server{
		Handler: http.DefaultServeMux,
		BaseContext: func(listener net.Listener) context.Context {
			return context.Background()
		},
	}

	err = srv.Serve(manet.NetListener(lst))
	if err == http.ErrServerClosed {
		return nil
	}

	return nil
}
