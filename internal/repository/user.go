package repository

import (
	"context"
	"errors"
	"time"

	"github.com/wuonhai/webook/internal/domain"
	"github.com/wuonhai/webook/internal/repository/dao"
)

var (
	ErrUserDuplicateEmail = dao.ErrUserDuplicateEmail
	ErrUserNotFound       = dao.ErrUserNotFound
)

type UserRepository struct {
	dao *dao.UserDao
}

func NewUserRepository(dao *dao.UserDao) *UserRepository {
	return &UserRepository{
		dao: dao,
	}
}

func (r *UserRepository) Create(ctx context.Context, userDo domain.User) error {

	return r.dao.Insert(ctx, dao.User{
		Email:    userDo.Email,
		Password: userDo.Password,
	})
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {

	userPo, err := r.dao.FindByEmail(ctx, email)
	if errors.Is(err, dao.ErrUserNotFound) {
		return domain.User{}, ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}

	var ctime time.Time
	if userPo.Ctime != 0 {
		ctime = time.UnixMilli(userPo.Ctime).UTC()
	}

	return domain.User{
		Id:       userPo.Id,
		Email:    userPo.Email,
		Password: userPo.Password,
		Ctime:    ctime,
	}, nil
}

func (r *UserRepository) FindById(ctx context.Context, id int64) (domain.User, error) {
	userPo, err := r.dao.FindById(ctx, id)
	if errors.Is(err, dao.ErrUserNotFound) {
		return domain.User{}, ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, err
	}

	var birthday *time.Time
	if userPo.Birthday != nil {
		t := time.UnixMilli(*userPo.Birthday).UTC()
		birthday = &t
	}
	var ctime time.Time
	if userPo.Ctime != 0 {
		ctime = time.UnixMilli(userPo.Ctime).UTC()
	}

	return domain.User{
		Id:       userPo.Id,
		Email:    userPo.Email,
		Password: userPo.Password,
		Nickname: userPo.Nickname,
		Birthday: birthday,
		Bio:      userPo.Bio,
		Ctime:    ctime,
	}, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, update domain.UserProfileUpdate) error {
	updatePo := dao.UserProfileUpdate{
		Id:            update.UserId,
		Nickname:      update.Nickname,
		ClearBirthday: update.ClearBirthday,
		Bio:           update.Bio,
	}
	if !update.ClearBirthday && update.Birthday != nil {
		ms := update.Birthday.UTC().UnixMilli()
		updatePo.Birthday = &ms
	}
	return r.dao.UpdateProfileById(ctx, updatePo)
}
