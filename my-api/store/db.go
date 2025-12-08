package store

import (
	"fmt"
	"sync"
	"time"
)

// User 是我們的資料模型
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Storage 定義了資料庫的行為 (Interface)，這是為了以後可以隨時換成 Postgres/MySQL
type Storage interface {
	Create(User) error
	Get(string) (User, error)
	List() ([]User, error)
}

// MemoryStore 是 Storage 的一個實作 (存在記憶體中)
type MemoryStore struct {
	mu    sync.RWMutex
	users map[string]User
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users: make(map[string]User),
	}
}

func (s *MemoryStore) Create(u User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[u.ID]; ok {
		return fmt.Errorf("user already exists")
	}
	s.users[u.ID] = u
	return nil
}

func (s *MemoryStore) Get(id string) (User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return User{}, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *MemoryStore) List() ([]User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var users []User
	for _, u := range s.users {
		users = append(users, u)
	}
	return users, nil
}
