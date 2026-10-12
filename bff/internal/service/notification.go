package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

// notificationLimit は通知一覧で返す最大件数（新しい順）
const notificationLimit = 50

var ErrNotificationNotFound = errors.New("通知が見つかりません")

// Notifications は通知一覧と未読の数
type Notifications struct {
	Unread        int
	Notifications []domain.Notification
}

type NotificationService interface {
	// GetNotifications はユーザーへの通知を新しい順に最大 50 件と、未読の数を返す
	GetNotifications(userID int) (Notifications, error)
	// MarkRead はユーザーへの通知を既読にする。なければ ErrNotificationNotFound を返す
	MarkRead(id int, userID int) error
	// MarkAllRead はユーザーへの通知をすべて既読にする
	MarkAllRead(userID int) error
}

type notificationService struct {
	notifications repository.NotificationRepository
}

func NewNotificationService(n repository.NotificationRepository) NotificationService {
	return &notificationService{notifications: n}
}

func (s *notificationService) GetNotifications(userID int) (Notifications, error) {
	list, err := s.notifications.FindByUser(userID, notificationLimit)
	if err != nil {
		return Notifications{}, err
	}
	unread, err := s.notifications.CountUnread(userID)
	if err != nil {
		return Notifications{}, err
	}
	return Notifications{Unread: unread, Notifications: list}, nil
}

func (s *notificationService) MarkRead(id int, userID int) error {
	found, err := s.notifications.MarkRead(id, userID)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotificationNotFound
	}
	return nil
}

func (s *notificationService) MarkAllRead(userID int) error {
	return s.notifications.MarkAllRead(userID)
}

// notifyWatchers は課題のウォッチャーのうち、操作した actorID と skip のユーザー以外に通知を作る
func notifyWatchers(watchers repository.WatcherRepository, notifications repository.NotificationRepository, todoID int, actorID int, kind domain.NotificationKind, detail string, skip map[int]bool) error {
	users, err := watchers.FindByTodo(todoID)
	if err != nil {
		return err
	}
	list := []domain.Notification{}
	for _, user := range users {
		if user.ID == actorID || skip[user.ID] {
			continue
		}
		list = append(list, domain.Notification{UserID: user.ID, TodoID: todoID, ActorID: actorID, Kind: kind, Detail: detail})
	}
	return notifications.Create(list)
}

// notifyMentions はメンションされたユーザー（userIDs）のうち、操作した actorID 以外の存在するユーザーに
// 「メンションされました」の通知を作り、通知したユーザーの ID を返す
func notifyMentions(users repository.UserRepository, notifications repository.NotificationRepository, todoID int, actorID int, userIDs []int, detail string) (map[int]bool, error) {
	notified := map[int]bool{}
	list := []domain.Notification{}
	for _, id := range userIDs {
		if id == actorID {
			continue
		}
		user, err := users.FindByID(id)
		if err != nil {
			return nil, err
		}
		if user == nil {
			continue
		}
		notified[id] = true
		list = append(list, domain.Notification{UserID: id, TodoID: todoID, ActorID: actorID, Kind: domain.NotificationMentioned, Detail: detail})
	}
	if err := notifications.Create(list); err != nil {
		return nil, err
	}
	return notified, nil
}

// notifyingHistory は変更履歴を記録したときに、変わった課題のウォッチャーへ通知を作る HistoryRepository。
// 課題を変える操作はすべて変更履歴を記録するので、どの操作で変わっても通知が届く
type notifyingHistory struct {
	repository.HistoryRepository
	watchers      repository.WatcherRepository
	notifications repository.NotificationRepository
}

// WithNotifications は、変更履歴の記録に合わせてウォッチャーへ通知を作る HistoryRepository を返す。
// 課題を変えるサービスには、これで包んだ HistoryRepository を渡す
func WithNotifications(h repository.HistoryRepository, w repository.WatcherRepository, n repository.NotificationRepository) repository.HistoryRepository {
	return &notifyingHistory{HistoryRepository: h, watchers: w, notifications: n}
}

// Add は変更履歴を記録し、課題ごとに 1 件の「変更しました」の通知を作る。作成だけの記録では通知しない
func (h *notifyingHistory) Add(entries []domain.HistoryEntry) error {
	if err := h.HistoryRepository.Add(entries); err != nil {
		return err
	}
	byTodo := map[int][]domain.HistoryEntry{}
	order := []int{}
	for _, e := range entries {
		if _, ok := byTodo[e.TodoID]; !ok {
			order = append(order, e.TodoID)
		}
		byTodo[e.TodoID] = append(byTodo[e.TodoID], e)
	}
	for _, todoID := range order {
		fields := domain.ChangedFields(byTodo[todoID])
		if fields == "" {
			continue
		}
		if err := notifyWatchers(h.watchers, h.notifications, todoID, byTodo[todoID][0].UserID, domain.NotificationUpdated, fields, nil); err != nil {
			return err
		}
	}
	return nil
}
