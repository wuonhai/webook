package domain

import "time"

type User struct {
	Id       int64
	Email    string
	Password string
	Nickname *string
	Birthday *time.Time
	Bio      *string
	Ctime    time.Time
}

// UserProfileUpdate represents partial profile updates where nil pointer fields remain unchanged.
type UserProfileUpdate struct {
	UserId        int64
	Nickname      *string    // nil = unchanged; non-nil = update
	Birthday      *time.Time // non-nil = update to UTC midnight date
	ClearBirthday bool       // true = clear birthday to NULL
	Bio           *string    // nil = unchanged; "" = clear to NULL; non-empty = update
}
