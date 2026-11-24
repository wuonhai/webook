package service

import (
	"context"
	"errors"

	"github.com/wuonhai/webook/internal/domain"
	"github.com/wuonhai/webook/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserDuplicateEmail    = repository.ErrUserDuplicateEmail
	ErrInvalidUserOrPassword = errors.New("invalid user or password")
	ErrUserNotFound          = repository.ErrUserNotFound
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (svc *UserService) Signup(ctx context.Context, user domain.User) error {

	// consider to encrypt the password before saving to database
	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)

	if err != nil {
		return err
	}

	user.Password = string(hash)

	return svc.repo.Create(ctx, user)
}

func (svc *UserService) Login(ctx context.Context, user domain.User) (domain.User, error) {
	userFind, err := svc.repo.FindByEmail(ctx, user.Email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return domain.User{}, ErrInvalidUserOrPassword
	}

	if err != nil {
		return domain.User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userFind.Password), []byte(user.Password)); err != nil {
		return domain.User{}, ErrInvalidUserOrPassword
	}

	return userFind, nil
}

func (svc *UserService) Profile(ctx context.Context, userId int64) (domain.User, error) {
	return svc.repo.FindById(ctx, userId)
}

func (svc *UserService) Edit(ctx context.Context, update domain.UserProfileUpdate) error {
	return svc.repo.UpdateProfile(ctx, update)
}
