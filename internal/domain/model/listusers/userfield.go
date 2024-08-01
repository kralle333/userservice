package listusers

type UserField int32

const (
	UserFieldUnspecified UserField = 0
	UserFieldFirstName   UserField = 1
	UserFieldSecondName  UserField = 2
	UserFieldNickname    UserField = 3
	UserFieldEmail       UserField = 4
	UserFieldCountry     UserField = 5
)
