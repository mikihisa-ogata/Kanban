package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

// SessionDuration はログインのセッションの有効期間
const SessionDuration = 30 * 24 * time.Hour

var (
	ErrEmailTaken         = errors.New("このメールアドレスは既に登録されています")
	ErrInvalidCredentials = errors.New("メールアドレスまたはパスワードが正しくありません")
	ErrUnauthenticated    = errors.New("ログインしてください")
)

type AuthService interface {
	// Signup はユーザーを作成してログインし、ユーザーとセッションのトークンを返す。
	// 入力が不正なら domain のエラー、メールアドレスが登録済みなら ErrEmailTaken を返す
	Signup(displayName, email, password string) (domain.User, string, error)
	// Login はメールアドレスとパスワードを確かめてログインし、ユーザーとセッションのトークンを返す。
	// 一致しなければ ErrInvalidCredentials を返す
	Login(email, password string) (domain.User, string, error)
	// Logout はトークンのセッションを削除する。セッションがなくてもエラーにしない
	Logout(token string) error
	// CurrentUser はトークンのセッションのユーザーを返す。トークンが空・無効・期限切れなら ErrUnauthenticated を返す
	CurrentUser(token string) (domain.User, error)
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewAuthService(u repository.UserRepository, s repository.SessionRepository) AuthService {
	return &authService{userRepo: u, sessionRepo: s}
}

func (s *authService) Signup(displayName, email, password string) (domain.User, string, error) {
	displayName = strings.TrimSpace(displayName)
	email = domain.NormalizeEmail(email)
	if err := domain.ValidateSignup(displayName, email, password); err != nil {
		return domain.User{}, "", err
	}
	existing, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return domain.User{}, "", err
	}
	if existing != nil {
		return domain.User{}, "", ErrEmailTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, "", err
	}
	id, err := s.userRepo.Create(domain.User{
		DisplayName:  displayName,
		Email:        email,
		PasswordHash: string(hash),
		AvatarColor:  domain.AvatarColorFor(email),
	})
	if errors.Is(err, repository.ErrDuplicateEmail) {
		return domain.User{}, "", ErrEmailTaken
	}
	if err != nil {
		return domain.User{}, "", err
	}
	return s.startSession(id)
}

func (s *authService) Login(email, password string) (domain.User, string, error) {
	user, err := s.userRepo.FindByEmail(domain.NormalizeEmail(email))
	if err != nil {
		return domain.User{}, "", err
	}
	if user == nil {
		return domain.User{}, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return domain.User{}, "", ErrInvalidCredentials
	}
	// 期限切れのセッションはログインのついでに片付ける
	if err := s.sessionRepo.DeleteExpired(); err != nil {
		return domain.User{}, "", err
	}
	return s.startSession(user.ID)
}

func (s *authService) Logout(token string) error {
	if token == "" {
		return nil
	}
	return s.sessionRepo.Delete(hashToken(token))
}

func (s *authService) CurrentUser(token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, ErrUnauthenticated
	}
	userID, err := s.sessionRepo.FindUserID(hashToken(token))
	if err != nil {
		return domain.User{}, err
	}
	if userID == 0 {
		return domain.User{}, ErrUnauthenticated
	}
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return domain.User{}, err
	}
	if user == nil {
		return domain.User{}, ErrUnauthenticated
	}
	return *user, nil
}

// startSession はユーザーのセッションを作り、ユーザーとトークンを返す
func (s *authService) startSession(userID int) (domain.User, string, error) {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return domain.User{}, "", err
	}
	if user == nil {
		return domain.User{}, "", ErrUnauthenticated
	}
	token, err := newSessionToken()
	if err != nil {
		return domain.User{}, "", err
	}
	if err := s.sessionRepo.Create(hashToken(token), userID, time.Now().Add(SessionDuration)); err != nil {
		return domain.User{}, "", err
	}
	return *user, token, nil
}

// newSessionToken は推測できないセッションのトークン（64 文字の 16 進数）を返す
func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hashToken はトークンを保存用の SHA-256（16 進数）にする。DB が漏れてもトークンを使えないようにするため
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
