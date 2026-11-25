package dao

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrUserDuplicateEmail = errors.New("email conflict")
	ErrUserNotFound       = gorm.ErrRecordNotFound
)

type UserDao struct {
	db *gorm.DB
}

func NewUserDao(db *gorm.DB) *UserDao {
	return &UserDao{
		db: db,
	}
}

func (dao *UserDao) Insert(ctx context.Context, userPo User) error {

	// consider to encrypt the password before saving to database
	now := time.Now().UnixMilli()
	userPo.Ctime = now
	userPo.Utime = now

	err := dao.db.WithContext(ctx).Create(&userPo).Error

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictErrNo uint16 = 1062
		if mysqlErr.Number == uniqueConflictErrNo {
			return ErrUserDuplicateEmail
		}
	}
	return err
}

func (dao *UserDao) FindByEmail(ctx context.Context, email string) (User, error) {
	var userPo User

	err := dao.db.WithContext(ctx).Where("email = ?", email).First(&userPo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}

	return userPo, err
}

func (dao *UserDao) FindById(ctx context.Context, id int64) (User, error) {
	var userPo User

	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&userPo).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrUserNotFound
	}

	return userPo, err
}

// UserProfileUpdate carries nullable fields for partial updates in the users table.
type UserProfileUpdate struct {
	Id            int64
	Nickname      *string
	Birthday      *int64
	ClearBirthday bool
	Bio           *string
}

func buildProfileUpdates(update UserProfileUpdate, now int64) map[string]any {
	updates := map[string]any{
		"utime": now,
	}
	if update.Nickname != nil {
		updates["nickname"] = *update.Nickname
	}
	if update.ClearBirthday {
		// Use untyped nil so GORM generates SET birthday = NULL instead of skipping or binding typed nil.
		updates["birthday"] = nil
	} else if update.Birthday != nil {
		updates["birthday"] = *update.Birthday
	}
	if update.Bio != nil {
		if *update.Bio == "" {
			updates["bio"] = nil
		} else {
			updates["bio"] = *update.Bio
		}
	}
	return updates
}

func (dao *UserDao) UpdateProfileById(ctx context.Context, update UserProfileUpdate) error {
	updates := buildProfileUpdates(update, time.Now().UnixMilli())
	return dao.db.WithContext(ctx).
		Model(&User{}).
		Where("id = ?", update.Id).
		Updates(updates).Error
}

type User struct {
	Id       int64  `gorm:"primaryKey;autoIncrement"`
	Email    string `gorm:"unique"`
	Password string

	// create time use milliseconds to avoid timezone problem
	Ctime int64
	// update time use milliseconds to avoid timezone problem
	Utime int64

	Nickname *string `gorm:"type:varchar(128)"`
	Birthday *int64
	Bio      *string `gorm:"type:varchar(1024)"`
}
