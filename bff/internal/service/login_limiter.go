package service

import (
	"sync"
	"time"
)

const (
	// loginFailureLimit は loginFailureWindow の間に続けて失敗できる回数。超えるとロックする
	loginFailureLimit = 5
	// loginFailureWindow は失敗を数える期間。最初の失敗からこの時間が過ぎたら数え直す
	loginFailureWindow = 15 * time.Minute
	// loginLockDuration はロックしてからログインを受け付けない時間
	loginLockDuration = 15 * time.Minute
)

// loginLimiter はメールアドレスごとにログインの失敗を数え、続けて失敗したらしばらくログインを止める（パスワードの総当たり対策）。
// 記録はメモリに持つので、bff を再起動すると消える
type loginLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	failures map[string]*loginFailures
}

type loginFailures struct {
	count       int
	firstAt     time.Time
	lockedUntil time.Time
}

func newLoginLimiter(now func() time.Time) *loginLimiter {
	return &loginLimiter{now: now, failures: map[string]*loginFailures{}}
}

// locked は email のログインがロック中かを返す
func (l *loginLimiter) locked(email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.failures[email]
	return ok && l.now().Before(f.lockedUntil)
}

// fail は email のログインの失敗を記録する。失敗が上限に達したらロックする
func (l *loginLimiter) fail(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.removeExpired(now)
	f, ok := l.failures[email]
	if !ok || now.Sub(f.firstAt) >= loginFailureWindow {
		f = &loginFailures{firstAt: now}
		l.failures[email] = f
	}
	f.count++
	if f.count >= loginFailureLimit {
		f.lockedUntil = now.Add(loginLockDuration)
	}
}

// succeed は email のログインの成功を記録し、失敗の回数を消す
func (l *loginLimiter) succeed(email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, email)
}

// removeExpired は数える期間もロックも過ぎた記録を消す（存在しないメールアドレスの記録がたまり続けないように）
func (l *loginLimiter) removeExpired(now time.Time) {
	for email, f := range l.failures {
		if now.Sub(f.firstAt) >= loginFailureWindow && !now.Before(f.lockedUntil) {
			delete(l.failures, email)
		}
	}
}
