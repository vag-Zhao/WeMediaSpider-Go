package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"WeMediaSpider/backend/internal/models"

	bolt "go.etcd.io/bbolt"
)

var (
	articlesBucket = []byte("articles")
	accountsBucket = []byte("accounts")
)

// Manager 缓存管理器
type Manager struct {
	db         *bolt.DB
	expireTime time.Duration
}

// NewManager 创建缓存管理器
func NewManager(expireHours int) (*Manager, error) {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".wemediaspider")
	os.MkdirAll(cacheDir, 0755)

	dbPath := filepath.Join(cacheDir, "cache.db")
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	// 创建 buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(articlesBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(accountsBucket); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &Manager{
		db:         db,
		expireTime: time.Duration(expireHours) * time.Hour,
	}, nil
}

// Close 关闭数据库
func (m *Manager) Close() error {
	return m.db.Close()
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

// SaveArticles 保存文章缓存
func (m *Manager) SaveArticles(accountFakeid string, articles []models.Article) error {
	data, err := json.Marshal(articles)
	if err != nil {
		return err
	}

	entry := CacheEntry{
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(articlesBucket)
		return b.Put([]byte(accountFakeid), entryData)
	})
}

// GetArticles 获取文章缓存
func (m *Manager) GetArticles(accountFakeid string) ([]models.Article, bool) {
	var articles []models.Article

	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(articlesBucket)
		data := b.Get([]byte(accountFakeid))
		if data == nil {
			return nil
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		// 检查是否过期
		if time.Since(time.Unix(entry.Timestamp, 0)) > m.expireTime {
			return nil
		}

		return json.Unmarshal(entry.Data, &articles)
	})

	if err != nil || len(articles) == 0 {
		return nil, false
	}

	return articles, true
}

// SaveAccount 保存公众号信息
func (m *Manager) SaveAccount(account models.Account) error {
	data, err := json.Marshal(account)
	if err != nil {
		return err
	}

	entry := CacheEntry{
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	return m.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountsBucket)
		return b.Put([]byte(account.Fakeid), entryData)
	})
}

// GetAccount 获取公众号信息
func (m *Manager) GetAccount(fakeid string) (models.Account, bool) {
	var account models.Account

	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(accountsBucket)
		data := b.Get([]byte(fakeid))
		if data == nil {
			return nil
		}

		var entry CacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}

		// 检查是否过期
		if time.Since(time.Unix(entry.Timestamp, 0)) > m.expireTime {
			return nil
		}

		return json.Unmarshal(entry.Data, &account)
	})

	if err != nil || account.Fakeid == "" {
		return models.Account{}, false
	}

	return account, true
}

// ClearExpired 清除过期缓存
func (m *Manager) ClearExpired() error {
	now := time.Now().Unix()

	return m.db.Update(func(tx *bolt.Tx) error {
		// 清除过期文章
		b := tx.Bucket(articlesBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry CacheEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				continue
			}
			if now-entry.Timestamp > int64(m.expireTime.Seconds()) {
				c.Delete()
			}
		}

		// 清除过期公众号
		b = tx.Bucket(accountsBucket)
		c = b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var entry CacheEntry
			if err := json.Unmarshal(v, &entry); err != nil {
				continue
			}
			if now-entry.Timestamp > int64(m.expireTime.Seconds()) {
				c.Delete()
			}
		}

		return nil
	})
}

// ClearAll 清除所有缓存
func (m *Manager) ClearAll() error {
	return m.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(articlesBucket); err != nil {
			return err
		}
		if err := tx.DeleteBucket(accountsBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucket(articlesBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucket(accountsBucket); err != nil {
			return err
		}
		return nil
	})
}

// GetStats 获取缓存统计
func (m *Manager) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	err := m.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(articlesBucket)
		stats["articles"] = b.Stats().KeyN

		b = tx.Bucket(accountsBucket)
		stats["accounts"] = b.Stats().KeyN

		return nil
	})

	return stats, err
}
