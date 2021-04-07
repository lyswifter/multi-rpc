package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	mrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
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

type KeyInfo struct {
	PrivateKey []byte
}

var random = mrand.New(mrand.NewSource(time.Now().UnixNano() | int64(os.Getpid())))
var AUTH_API_INFO = ""

func init() {
	if p := os.Getenv("AUTH_API_INFO"); p != "" {
		AUTH_API_INFO = p
	}
}

func main() {
	pathname := "AUTH_API_INFO"

	ipaddr := flag.String("ip", "127.0.0.1", "ip address specify")
	port := flag.String("port", "1234", "port specify")
	t := flag.String("type", "server", "node type specify")

	flag.Parse()

	if *t == "server" {
		bt, erra := os.ReadFile(pathname)
		if erra != nil {
			if os.IsNotExist(erra) {
				fmt.Printf("file is not exist: %s\n", pathname)
			}

			if os.IsExist(erra) {
				fmt.Printf("file is exist: %s\n", pathname)
			}
		}

		var targetAddr string
		var targetKinfo KeyInfo
		if len(bt) > 0 {
			bts := strings.Split(string(bt), "\n")
			for _, addr := range bts {
				if strings.Contains(addr, *port) {
					targetAddr = strings.Split(addr, "#")[0]

					tmp := KeyInfo{}
					erra := json.Unmarshal([]byte(strings.Split(addr, "#")[1]), &tmp)
					if erra != nil {
						return
					}

					targetKinfo = tmp
					break
				}
			}
		}

		fmt.Printf("Already targetAddr: %s\n", targetAddr)

		var ml multiaddr.Multiaddr
		var err error
		var kbytes []byte
		if targetAddr != "" && len(targetKinfo.PrivateKey) > 0 {
			info := ParseApiInfo(targetAddr)

			ml, err = multiaddr.NewMultiaddr(info.Addr)
			if err != nil {
				return
			}

			kbytes = targetKinfo.PrivateKey

			fmt.Printf("already kbytes: %v\n", kbytes)
		} else {
			ml, err = multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%s/http", *ipaddr, *port))
			if err != nil {
				return
			}

			pk, err := genLibp2pKey()
			if err != nil {
				return
			}

			kbytes, err = pk.Bytes()
			if err != nil {
				return
			}

			kinfo := KeyInfo{
				PrivateKey: kbytes,
			}

			kinfostring, err := json.Marshal(kinfo)
			if err != nil {
				return
			}

			fmt.Printf("newly kbytes: %v\n", kinfo)

			apialg := (*ljwt.APIAlg)(jwt.NewHS256(kbytes))
			auth, err := ljwt.AuthNew(context.TODO(), api.AllPermissions, apialg)
			if err != nil {
				return
			}

			var f *os.File
			_, err = os.Stat(pathname)
			if err != nil {
				if !os.IsExist(err) {
					f, err = os.OpenFile(pathname, os.O_CREATE|os.O_RDWR, 0666)
					if err != nil {
						return
					}
				}
			} else {
				f, err = os.OpenFile(pathname, os.O_APPEND|os.O_RDWR, 0666)
				if err != nil {
					return
				}
				_, err = f.WriteString("\n")
				if err != nil {
					return
				}
			}

			defer f.Close()

			destStr := fmt.Sprintf("%s:%s#%s", auth, ml.String(), kinfostring)
			_, err = f.WriteString(destStr)
			if err != nil {
				return
			}

			fmt.Printf("AuthInfo: %s\n", destStr)
		}

		apialg := (*ljwt.APIAlg)(jwt.NewHS256(kbytes))

		fmt.Println("RPC server is running...")

		err = serveRPC(&full.FullNodeAPI{
			APISecret: apialg,
		}, ml, 512)
		if err != nil {
			fmt.Printf("serveRPC: %s", err)
			return
		}
	}

	if *t == "client" {
		//read all running api info
		bt, err := os.ReadFile(pathname)
		if err != nil {
			return
		}
		bts := strings.Split(string(bt), "\n")
		firstOne := strings.Split(bts[0], "#")[0]

		ainfo := ParseApiInfo(firstOne)
		fmt.Printf("parseApiInfo first: %s %s\n", ainfo.Addr, ainfo.Token)

		addr, err := ainfo.DialArgs()
		if err != nil {
			fmt.Printf("dial: %s\n", err)
			return
		}

		var res api.FullStruct
		closer, err := jsonrpc.NewMergeClient(context.TODO(), addr, "MultiRPC",
			[]interface{}{
				&res.Internal,
			}, ainfo.AuthHeader(), jsonrpc.WithSwitchFile(AUTH_API_INFO))
		if err != nil {
			fmt.Printf("newClient: %s\n", err)
			return
		}
		defer closer()

		fmt.Println("RPC client is running...")

		var count = 0
		for {
			count++
			if count > 100 {
				break
			}
			time.Sleep(500 * time.Millisecond)

			fmt.Println("Loop")

			idx, err := res.FuncA(context.TODO(), count)
			if err != nil {
				fmt.Printf("FuncA in: %d err: %s\n", count, err.Error())
				continue
			}

			fmt.Printf("FuncA ok in:%d out:%d\n", count, idx)
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
