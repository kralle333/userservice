package listusers

import "userservice/internal/domain/model"

type Response struct {
	Next  PageInfo     `json:"next"`
	Users []model.User `json:"users"`
}
