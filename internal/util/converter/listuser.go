package converter

import (
	"userservice/internal/domain/model/listusers"
	"userservice/proto/grpc"
)

func ConvertUserField(protoField grpc.UserField) listusers.UserField {
	return listusers.UserField(protoField)
}

func ConvertComparer(protoComparer grpc.Comparer) listusers.Comparer {
	return listusers.Comparer(protoComparer)
}

func ConvertPageInfo(protoPageInfo *grpc.PageInfo) *listusers.PageInfo {
	if protoPageInfo == nil {
		return nil
	}
	return &listusers.PageInfo{
		Limit:  protoPageInfo.Limit,
		Cursor: protoPageInfo.Cursor,
	}
}

func ConvertOrdering(protoOrdering *grpc.Ordering) *listusers.Ordering {
	if protoOrdering == nil {
		return nil
	}
	ordering := listusers.Ordering(*protoOrdering)
	return &ordering
}

func ConvertSortInfo(protoSortInfo *grpc.SortInfo) *listusers.SortInfo {
	if protoSortInfo == nil {
		return nil
	}
	return &listusers.SortInfo{
		By:    ConvertUserField(protoSortInfo.By),
		Order: ConvertOrdering(protoSortInfo.Order),
	}
}

func ConvertFilterInfo(protoFilterInfo *grpc.FilterInfo) *listusers.FilterInfo {
	if protoFilterInfo == nil {
		return nil
	}
	return &listusers.FilterInfo{
		Left:     ConvertUserField(protoFilterInfo.Left),
		Comparer: ConvertComparer(protoFilterInfo.Comparer),
		Right:    protoFilterInfo.Right,
	}
}

// ConvertListUsersRequest converts a Protobuf Request to a Go Request
func ConvertListUsersRequest(protoRequest *grpc.ListUsersRequest) *listusers.Request {
	if protoRequest == nil {
		return nil
	}
	return &listusers.Request{
		Sorting:   ConvertSortInfo(protoRequest.Sorting),
		Filtering: ConvertFilterInfo(protoRequest.Filtering),
		Paging:    ConvertPageInfo(protoRequest.Paging),
	}
}
