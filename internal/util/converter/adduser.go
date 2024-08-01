package converter

import (
	"userservice/internal/domain/model/adduser"
	"userservice/proto/grpc"
)

func ToAddUserRequest(request *grpc.AddUserRequest) adduser.Request {
	return adduser.Request{
		FirstName: request.User.FirstName,
		LastName:  request.User.LastName,
		Nickname:  request.User.Nickname,
		Password:  request.User.Password,
		Email:     request.User.Email,
		Country:   request.User.Country,
	}
}
