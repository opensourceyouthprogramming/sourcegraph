package local

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"src.sourcegraph.com/sourcegraph/go-sourcegraph/sourcegraph"
	"sourcegraph.com/sqs/pbtypes"
	"src.sourcegraph.com/sourcegraph/auth/authutil"
	"src.sourcegraph.com/sourcegraph/auth/idkey"
	"src.sourcegraph.com/sourcegraph/conf"
	"src.sourcegraph.com/sourcegraph/fed"
	"src.sourcegraph.com/sourcegraph/sgx/buildvar"
)

var Meta sourcegraph.MetaServer = &meta{}

type meta struct{}

var _ sourcegraph.MetaServer = (*meta)(nil)

var serverStart = time.Now().UTC()

func (s *meta) Status(ctx context.Context, _ *pbtypes.Void) (*sourcegraph.ServerStatus, error) {
	hostname, _ := os.Hostname()

	buildInfo, _ := json.MarshalIndent(buildvar.All, "\t", "  ")

	return &sourcegraph.ServerStatus{
		Info: fmt.Sprintf("hostname: %s\nuptime: %s\nbuild info:\n\t%s", hostname, time.Since(serverStart)/time.Second*time.Second, buildInfo),
	}, nil
}

func (s *meta) Config(ctx context.Context, _ *pbtypes.Void) (*sourcegraph.ServerConfig, error) {
	c := &sourcegraph.ServerConfig{
		Version:               buildvar.Version,
		AppURL:                conf.AppURL(ctx).String(),
		AllowAnonymousReaders: authutil.ActiveFlags.AllowAnonymousReaders,
		IDKey:      idkey.FromContext(ctx).ID,
		AuthSource: authutil.ActiveFlags.Source,
	}

	c.IsFederationRoot = fed.Config.IsRoot
	if u := fed.Config.RootURL(); u != nil {
		c.FederationRootURL = u.String()
	}

	return c, nil
}

func (s *meta) PubKey(ctx context.Context, _ *pbtypes.Void) (*sourcegraph.ServerPubKey, error) {
	idKey := idkey.FromContext(ctx)
	if idKey == nil {
		return nil, grpc.Errorf(codes.Unavailable, "public key unavailable")
	}
	pubKey, err := x509.MarshalPKIXPublicKey(idKey.Public())
	if err != nil {
		return nil, err
	}
	return &sourcegraph.ServerPubKey{
		Key: string(pubKey),
	}, nil
}
