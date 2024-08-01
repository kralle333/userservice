package mongodb

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strings"
	"userservice/internal/domain/model/listusers"
	"userservice/internal/domain/model/updateuser"
	"userservice/internal/util/crypto"
	"userservice/internal/util/time"
	"userservice/internal/util/validation"
)

var errInvalidField = errors.New("invalid field")
var errInvalidComparer = errors.New("invalid comparer")
var errNoCursor = errors.New("no cursor")
var errInvalidCursorNoSeparator = errors.New("cursor is missing separator")
var errInvalidCursorBadMongoID = errors.New("cursor has incorrect mongoid")
var errNoSorting = errors.New("invalid sort field")

var userFieldToStringMap = map[listusers.UserField]string{
	listusers.UserFieldFirstName:  "first_name",
	listusers.UserFieldSecondName: "second_name",
	listusers.UserFieldCountry:    "country",
	listusers.UserFieldEmail:      "email",
	listusers.UserFieldNickname:   "nickname",
}
var orderingToStringMap = map[listusers.Comparer]string{
	listusers.ComparerGreaterThan:      "$gt",
	listusers.ComparerGreaterThanEqual: "$gte",
	listusers.ComparerLessThan:         "$lt",
	listusers.ComparerLessThanEqual:    "$lte",
	listusers.ComparerEqual:            "$eq",
}

func fieldToString(field listusers.UserField) (string, error) {
	parsed, ok := userFieldToStringMap[field]
	if !ok {
		return "", errInvalidField
	}
	return parsed, nil
}

func comparerString(comparer listusers.Comparer) (string, error) {
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

func getPaginationFilter(rawCursor string, sorting *listusers.SortInfo) (bson.D, error) {
	mongoID, value, err := deconstructCursor(rawCursor)

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

func getFilterFilter(filter *listusers.FilterInfo) (bson.D, error) {

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

// constructListUsersFilter using the approach found here https://medium.com/swlh/mongodb-pagination-fast-consistent-ece2a97070f3
// TLDR: keyset pagination using internal mongodb _id for ties, which is possible because of how _id are generated
func constructListUsersFilter(filter *listusers.FilterInfo, sorting *listusers.SortInfo, rawCursor string) bson.D {

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

func constructListUsersSortBy(sortBy *listusers.SortInfo) bson.D {
	if sortBy == nil {
		return bson.D{{"_id", 1}}
	}

	order := 1
	if sortBy.Order != nil && *sortBy.Order == listusers.OrderingDescending {
		order = -1
	}

	parsed, err := fieldToString(sortBy.By)
	if err != nil {
		return bson.D{{"_id", order}}
	}
	return bson.D{{parsed, order}, {"_id", order}}

}

func constructUpdateFilter(modifiedUser updateuser.Request, passwordSalt string) (bson.D, error) {
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

func getValue(user DBUser, field listusers.UserField) string {
	switch field {
	case listusers.UserFieldUnspecified:
		return ""
	case listusers.UserFieldFirstName:
		return user.FirstName
	case listusers.UserFieldSecondName:
		return user.LastName
	case listusers.UserFieldNickname:
		return user.Nickname
	case listusers.UserFieldEmail:
		return user.Email
	case listusers.UserFieldCountry:
		return user.Country
	}

	log.Warn().Msgf("unhandled userfield for user value extraction: %s", field)
	return ""
}

func constructCursor(user DBUser, sortByField *listusers.SortInfo) string {
	sortFilter := ""
	if sortByField != nil && sortByField.By != listusers.UserFieldUnspecified {
		sortFilter = getValue(user, sortByField.By)
	}
	return fmt.Sprintf("%s:%s", user.MongoDBID.Hex(), sortFilter)
}

func deconstructCursor(cursor string) (string, string, error) {
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
