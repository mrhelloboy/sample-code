package etcd_registry

import (
	"context"
	userv1 "github.com/mrhelloboy/etcd-registry/api/gen/user/v1"
	"strconv"
)

type UserSrv struct {
	userv1.UnimplementedUserServer
}

func (u UserSrv) GetUser(ctx context.Context, request *userv1.GetUserRequest) (*userv1.GetUserReply, error) {
	return &userv1.GetUserReply{
		User: &userv1.UserInfo{
			Id:   request.Id,
			Name: "test" + strconv.Itoa(int(request.Id)),
		},
	}, nil
}
