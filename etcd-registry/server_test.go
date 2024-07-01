package etcd_registry

import (
	"context"
	"fmt"
	userv1 "github.com/mrhelloboy/etcd-registry/api/gen/user/v1"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
	"net"
	"testing"
	"time"
)

type ServerRegisterTestSuite struct {
	suite.Suite
	client *clientv3.Client
}

func TestServer(t *testing.T) {
	suite.Run(t, new(ServerRegisterTestSuite))
}

func (s *ServerRegisterTestSuite) SetupSuite() {
	// etcd 初始化
	client, err := clientv3.New(clientv3.Config{
		Endpoints: []string{"localhost:12379"},
	})
	require.NoError(s.T(), err)
	s.client = client
}

func (s *ServerRegisterTestSuite) TestServer() {

	// 创建一个名为 "production/service/user" 的 endpoint manger
	targetName := "production/service/user"
	em, err := endpoints.NewManager(s.client, targetName)
	require.NoError(s.T(), err)

	// 向注册中心注册服务实例的定位信息（ip+端口）
	addr := "192.168.10.2:8080"
	// endpoint 的 key：targetName + "/" + 唯一实例名称
	// 实例名称：
	//  主机名（host-8080）、
	//  IP 地址（192.168.10.2:8080）
	//  实例 ID （instance-id）
	// 这里使用 IP 地址作为实例名称
	eKey := fmt.Sprintf("%s/%s", targetName, addr)

	// 维持服务实例的有效性：租约
	// 创建一个 ttl 为 30 秒的租约
	lCtx, lCancel := context.WithTimeout(context.Background(), time.Second)
	defer lCancel()
	lease, err := s.client.Grant(lCtx, 30)
	require.NoError(s.T(), err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = em.AddEndpoint(ctx, eKey, endpoints.Endpoint{
		Addr: addr,
	}, clientv3.WithLease(lease.ID))

	// 续约
	kaCtx, kaCancel := context.WithCancel(context.Background())
	go func() {
		kaResp, err := s.client.KeepAlive(kaCtx, lease.ID)
		if err != nil {
			s.T().Log("续约失败", err.Error())
		}
		for resp := range kaResp {
			s.T().Log("续约成功", resp)
		}
	}()

	// 更新服务实例注册信息
	go func() {
		ticker := time.NewTicker(time.Second)
		for now := range ticker.C {
			uCtx, uCancel := context.WithTimeout(context.Background(), time.Second)
			err = em.AddEndpoint(uCtx, eKey, endpoints.Endpoint{
				Addr: addr,
				// 可以是分组信息，权重信息，机房信息
				// 或者动态判定负载时，将负载信息添加到 Metadata 中
				Metadata: map[string]string{
					"version": "v1.0.0",
					"time":    now.Format("2006-01-02 15:04:05"),
				},
			}, clientv3.WithLease(lease.ID))

			uCancel()
		}

	}()

	// 启动服务
	server := grpc.NewServer()
	userv1.RegisterUserServer(server, &UserSrv{})
	l, err := net.Listen("tcp", ":8080")
	require.NoError(s.T(), err)
	err = server.Serve(l)
	s.T().Log("user server error:", err)

	// 退出服务
	// 取消续租
	kaCancel()
	// 删除服务实例
	err = em.DeleteEndpoint(ctx, eKey)
	require.NoError(s.T(), err)
	err = s.client.Close()
	require.NoError(s.T(), err)
	server.GracefulStop()
}
