package service

import (
	"testing"
	"time"
)

func TestLoginLimiter(t *testing.T) {
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	l := newLoginLimiter(func() time.Time { return now })

	// 上限の 1 回手前まではロックしない
	for i := 0; i < loginFailureLimit-1; i++ {
		l.fail("taro@example.com")
	}
	if l.locked("taro@example.com") {
		t.Fatal("locked before reaching the limit")
	}
	// 成功すると数え直す
	l.succeed("taro@example.com")
	l.fail("taro@example.com")
	if l.locked("taro@example.com") {
		t.Fatal("locked after success reset")
	}

	// 上限に達するとロックし、ほかのメールアドレスには影響しない
	for i := 0; i < loginFailureLimit-1; i++ {
		l.fail("taro@example.com")
	}
	if !l.locked("taro@example.com") {
		t.Fatal("not locked after reaching the limit")
	}
	if l.locked("hanako@example.com") {
		t.Fatal("other email is locked")
	}

	// ロックの時間が過ぎると解ける
	now = now.Add(loginLockDuration)
	if l.locked("taro@example.com") {
		t.Fatal("still locked after the lock duration")
	}

	// 数える期間を過ぎた失敗は数えない
	for i := 0; i < loginFailureLimit-1; i++ {
		l.fail("jiro@example.com")
	}
	now = now.Add(loginFailureWindow)
	l.fail("jiro@example.com")
	if l.locked("jiro@example.com") {
		t.Fatal("locked by failures outside the window")
	}
	// 期限の切れた記録は消える
	if _, ok := l.failures["taro@example.com"]; ok {
		t.Error("expired record of taro remains")
	}
}
