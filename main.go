package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/gbrlsnchs/jwt/v3"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/lyswifter/api"
	"github.com/lyswifter/full"
	ljwt "github.com/lyswifter/jwt"
	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"golang.org/x/xerrors"
)

func main() {
	fmt.Println("main running")

	ipaddr := flag.String("ip", "127.0.0.1", "ip address specify")
	port := flag.String("port", "1234", "port specify")
	t := flag.String("type", "server", "node type specify")
	destApi := flag.String("dest", "xxxxx", "destination api info specify")

	flag.Parse()

	if *t == "server" {
		ml, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s/http", *ipaddr, *port))
		if err != nil {
			return
		}

		pk, err := genLibp2pKey()
		if err != nil {
			return
		}

		kbytes, err := pk.Bytes()
		if err != nil {
			return
		}

		apialg := (*ljwt.APIAlg)(jwt.NewHS256(kbytes))
		auth, err := ljwt.AuthNew(context.TODO(), api.AllPermissions, apialg)
		if err != nil {
			return
		}

		fmt.Printf("AuthInfo: %s:%s\n", auth, ml.String())

		err = serveRPC(&full.FullNodeAPI{
			APISecret: apialg,
		}, ml, 512)
		if err != nil {
			return
		}
	}

	if *t == "client" {
		ainfo := ParseApiInfo(*destApi)

		fmt.Printf("parseApiInfo: %s %s\n", ainfo.Addr, ainfo.Token)

		addr, err := ainfo.DialArgs()
		if err != nil {
			fmt.Printf("dial: %s\n", err)
			return
		}

		var res api.FullStruct
		closer, err := jsonrpc.NewMergeClient(context.TODO(), addr, "MultiRPC",
			[]interface{}{
				&res.Internal,
			}, ainfo.AuthHeader())
		if err != nil {
			fmt.Printf("newClient: %s\n", err)
			return
		}

		defer closer()

		ticker := time.NewTicker(2)
		ctx := context.Background()
		for {
			select {
			case <-ticker.C:
				fmt.Println("Loop")

				err := res.FuncA(context.TODO())
				if err != nil {
					fmt.Printf("err: %s\n", err.Error())
				}

				fmt.Println("FuncA")

				// auth, err := res.AuthNew(context.TODO(), api.AllPermissions)
				// if err != nil {
				// 	fmt.Printf("err: %s\n", err.Error())
				// 	return
				// }

				// fmt.Printf("auth: %s\n", auth)
			case <-ctx.Done():
				fmt.Println("done")
			}
		}
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

func genLibp2pKey() (crypto.PrivKey, error) {
	pk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

var (
	infoWithToken = regexp.MustCompile("^[a-zA-Z0-9\\-_]+?\\.[a-zA-Z0-9\\-_]+?\\.([a-zA-Z0-9\\-_]+)?:.+$")
)

type APIInfo struct {
	Addr  string
	Token []byte
}

func ParseApiInfo(s string) APIInfo {
	var tok []byte
	if infoWithToken.Match([]byte(s)) {
		sp := strings.SplitN(s, ":", 2)
		tok = []byte(sp[0])
		s = sp[1]
	}

	return APIInfo{
		Addr:  s,
		Token: tok,
	}
}

func (a APIInfo) DialArgs() (string, error) {
	ma, err := multiaddr.NewMultiaddr(a.Addr)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(a.Addr)
	if err != nil {
		return "", err
	}
	return a.Addr + "/rpc/v0", nil
}

func (a APIInfo) Host() (string, error) {
	ma, err := multiaddr.NewMultiaddr(a.Addr)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return addr, nil
	}

	spec, err := url.Parse(a.Addr)
	if err != nil {
		return "", err
	}
	return spec.Host, nil
}

func (a APIInfo) AuthHeader() http.Header {
	if len(a.Token) != 0 {
		headers := http.Header{}
		headers.Add("Authorization", "Bearer "+string(a.Token))
		return headers
	}
	return nil
}
