package mongodb

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"userservice/internal/db"
	"userservice/internal/util/crypto"
	"userservice/internal/util/time"
	"userservice/internal/util/validation"
	"userservice/proto/grpc"
)

var errInvalidField = errors.New("invalid field")
var errInvalidComparer = errors.New("invalid comparer")
var errNoCursor = errors.New("no cursor")
var errInvalidCursorNoSeparator = errors.New("cursor is missing separator")
var errInvalidCursorBadMongoID = errors.New("cursor has incorrect mongoid")
var errNoSorting = errors.New("invalid sort field")

var userFieldToStringMap = map[grpc.UserField]string{
	grpc.UserField_USER_FIELD_FIRST_NAME:  "first_name",
	grpc.UserField_USER_FIELD_SECOND_NAME: "second_name",
	grpc.UserField_USER_FIELD_COUNTRY:     "country",
	grpc.UserField_USER_FIELD_EMAIL:       "email",
	grpc.UserField_USER_FIELD_NICKNAME:    "nickname",
}
var orderingToStringMap = map[grpc.Comparer]string{
	grpc.Comparer_COMPARER_GREATER_THAN:       "$gt",
	grpc.Comparer_COMPARER_GREATER_THAN_EQUAL: "$gte",
	grpc.Comparer_COMPARER_LESS_THAN:          "$lt",
	grpc.Comparer_COMPARER_LESS_THAN_EQUAL:    "$lte",
	grpc.Comparer_COMPARER_EQUAL:              "$eq",
}

func fieldToString(field grpc.UserField) (string, error) {
	parsed, ok := userFieldToStringMap[field]
	if !ok {
		return "", errInvalidField
	}
	return parsed, nil
}

func comparerString(comparer grpc.Comparer) (string, error) {
	parsed, ok := orderingToStringMap[comparer]
	if !ok {
		return "", errInvalidComparer
	}
	return parsed, nil
}

func getPaginationFilterWhenNoSorting(mongoID string) bson.D {
	if mongoID == "" {
		return bson.D{}
	}
	return bson.D{{"_id", bson.D{{"$gt", mongoID}}}}
}

func getPaginationFilter(rawCursor string, sorting *grpc.SortInfo) (bson.D, error) {
	mongoID, value, err := DeconstructCursor(rawCursor)

	// let's log when getting a bad cursor to make it easier to debug with frontenders
	if err != nil && !errors.Is(err, errNoCursor) {
		return bson.D{}, err
	}

	// determining if we have valid sorting
	if sorting == nil {
		return getPaginationFilterWhenNoSorting(mongoID), nil
	}

	sortByFieldString, err := fieldToString(sorting.By)
	if err != nil {
		log.Warn().Err(err).Msgf("error creating sorting string %s", sorting.By)
		return getPaginationFilterWhenNoSorting(mongoID), nil
	}

	// we have sorting, lets construct the filter
	return bson.D{
		{"$or", bson.A{
			bson.D{{sortByFieldString, bson.D{{"$gt", value}}}},
			bson.D{ // In case of ties, use _id instead
				{sortByFieldString, value},
				{"_id", bson.D{{"$gt", rawCursor}}},
			},
		}},
	}, nil
}

func getFilterFilter(filter *grpc.FilterInfo) (bson.D, error) {

	// 3 components of this filter: left side (fieldFormat), center (orderingFormat) and right (string

	// first component
	parsed, err := fieldToString(filter.Left)
	if err != nil {
		return bson.D{}, err
	}

	// second component
	orderingString, err := comparerString(filter.Comparer)
	if err != nil {
		return bson.D{}, err
	}

	// final filter, third component does not need any parsing
	return bson.D{
		{"$or", bson.A{
			bson.D{{parsed, bson.D{{orderingString, filter.Right}}}},
		}},
	}, nil
}

// ConstructListUsersFilter using the approach found here https://medium.com/swlh/mongodb-pagination-fast-consistent-ece2a97070f3
// TLDR: keyset pagination using internal mongodb _id for ties, which is possible because of how _id are generated
func ConstructListUsersFilter(filter *grpc.FilterInfo, sorting *grpc.SortInfo, rawCursor string) bson.D {

	paginationFilter, err := getPaginationFilter(rawCursor, sorting)

	if err != nil && !errors.Is(err, errNoSorting) {
		log.Warn().Err(err).Msgf("invalid sorting: %s", sorting)
	}
	if filter == nil {
		return paginationFilter
	}

	filteringFilter, err := getFilterFilter(filter)

	if err != nil {
		return paginationFilter
	}

	return bson.D{
		{"$and", bson.A{
			filteringFilter,
			paginationFilter,
		}},
	}
}

func ConstructListUsersSortBy(sortBy *grpc.SortInfo) bson.D {
	if sortBy == nil {
		return bson.D{{"_id", grpc.Ordering_ORDERING_ASCENDING}}
	}

	order := 1
	if sortBy.Order != nil && *sortBy.Order == grpc.Ordering_ORDERING_DESCENDING {
		order = -1
	}

	parsed, err := fieldToString(sortBy.By)
	if err != nil {
		return bson.D{{"_id", order}}
	}
	return bson.D{{parsed, order}, {"_id", order}}

}

func ConstructUpdateFilter(modifiedUser *grpc.ModifyUserRequestUser, passwordSalt string) (bson.D, error) {

	// use optional property to only update what the client wishes to update
	a := bson.D{}
	if modifiedUser.FirstName != nil {
		a = append(a, bson.E{Key: "first_name", Value: *modifiedUser.FirstName})
	}
	if modifiedUser.LastName != nil {
		a = append(a, bson.E{Key: "last_name", Value: *modifiedUser.LastName})
	}
	if modifiedUser.Email != nil {
		if !validation.IsEmailValid(*modifiedUser.Email) {
			return nil, errors.New("invalid email address")
		}
		a = append(a, bson.E{Key: "email", Value: *modifiedUser.Email})
	}
	if modifiedUser.Nickname != nil {
		a = append(a, bson.E{Key: "nickname", Value: *modifiedUser.Nickname})
	}
	if modifiedUser.Country != nil {
		err := validation.ValidateCountryCode(*modifiedUser.Country)
		if err != nil {
			return nil, err
		}
		a = append(a, bson.E{Key: "country", Value: *modifiedUser.Country})
	}
	if modifiedUser.Password != nil {
		newPassword := crypto.GenerateHashedPassword(*modifiedUser.Password, passwordSalt)
		a = append(a, bson.E{Key: "password", Value: newPassword})
	}

	now := time.DBNow()
	a = append(a, bson.E{Key: "updated_at", Value: now})

	return bson.D{{"$set", a}}, nil
}

func getValue(user db.User, field grpc.UserField) string {
	switch field {
	case grpc.UserField_USER_FIELD_UNSPECIFIED:
		return ""
	case grpc.UserField_USER_FIELD_FIRST_NAME:
		return user.FirstName
	case grpc.UserField_USER_FIELD_SECOND_NAME:
		return user.LastName
	case grpc.UserField_USER_FIELD_NICKNAME:
		return user.Nickname
	case grpc.UserField_USER_FIELD_EMAIL:
		return user.Email
	case grpc.UserField_USER_FIELD_COUNTRY:
		return user.Country
	}

	log.Warn().Msgf("unhandled userfield for user value extraction: %s", field)
	return ""
}

func ConstructCursor(user db.User, sortByField grpc.UserField) string {
	value := getValue(user, sortByField)
	return fmt.Sprintf("%s:%s", user.MongoDBID.Hex(), value)
}

func DeconstructCursor(cursor string) (string, string, error) {
	if cursor == "" {
		return "", "", errNoCursor
	}
	split := strings.Split(cursor, ":")
	if len(split) != 2 {
		return "", "", errInvalidCursorNoSeparator
	}

	rawID, value := split[0], split[1]

	_, err := primitive.ObjectIDFromHex(rawID)
	if err != nil {
		return "", "", errInvalidCursorBadMongoID
	}

	return rawID, value, nil
}
