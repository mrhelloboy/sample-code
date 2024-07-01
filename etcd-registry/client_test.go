package etcd_registry

import (
	"context"
	userv1 "github.com/mrhelloboy/etcd-registry/api/gen/user/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

type ServerDiscoveryTestSuite struct {
	suite.Suite
	client *clientv3.Client
}

func TestClient(t *testing.T) {
	suite.Run(t, new(ServerDiscoveryTestSuite))
}

func (s *ServerDiscoveryTestSuite) SetupSuite() {
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:12379"},
	})
	s.NoError(err)
	s.client = client
}

func (s *ServerDiscoveryTestSuite) TestClient() {
	rb, err := resolver.NewBuilder(s.client)
	require.NoError(s.T(), err)

	cc, err := grpc.NewClient(
		"etcd:///production/service/user",
		grpc.WithResolvers(rb),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(s.T(), err)

	uc := userv1.NewUserClient(cc)
	user, err := uc.GetUser(context.Background(), &userv1.GetUserRequest{Id: 1})
	require.NoError(s.T(), err)
	s.T().Log(user.User)
}
