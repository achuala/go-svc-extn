// Package cache provides a unified interface for local and remote caching with advanced features:
//   - Individual hash field expiration (Valkey 9.0+)
//   - Automatic TTL extension on access (ApplyTouch)
//   - Distributed locking with Redlock algorithm
//   - Multi-device session management
//
// Complete Example: Multi-Device Session Manager with Individual Expiration
//
//	type SessionToken struct {
//	    Token     string
//	    UserID    string
//	    DeviceID  string
//	    CreatedAt time.Time
//	}
//
//	type SessionManager struct {
//	    cache cache.Cache[SessionToken]
//	}
//
//	func NewSessionManager() (*SessionManager, error) {
//	    // Create cache with touch enabled for activity-based sessions
//	    c, err, cleanup := cache.NewCache(&cache.CacheConfig[SessionToken]{
//	        Mode:            "remote",
//	        CacheName:       "sessions",
//	        DefaultTTL:      30 * time.Minute,  // Sessions extend to 30min on each access
//	        RemoteCacheAddr: "localhost:6379",
//	        ApplyTouch:      true,              // Extend TTL on HGet
//	        SerDe:           cache.NewJSONSerDe[SessionToken](),
//	    })
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    return &SessionManager{cache: c}, nil
//	}
//
//	// Login creates a new session with device-specific expiration
//	func (sm *SessionManager) Login(ctx context.Context, userID, deviceType string) (SessionToken, error) {
//	    sessionID := generateSessionID()
//	    token := SessionToken{
//	        Token:     generateToken(),
//	        UserID:    userID,
//	        DeviceID:  sessionID,
//	        CreatedAt: time.Now(),
//	    }
//
//	    // Different TTLs based on device type
//	    var ttl time.Duration
//	    switch deviceType {
//	    case "mobile":
//	        ttl = 1 * time.Hour
//	    case "web":
//	        ttl = 8 * time.Hour
//	    case "desktop":
//	        ttl = 30 * 24 * time.Hour
//	    default:
//	        ttl = 2 * time.Hour
//	    }
//
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//	    err := sm.cache.HSetWithTTL(ctx, key, sessionID, token, ttl)
//	    return token, err
//	}
//
//	// ValidateSession checks and extends session TTL (because ApplyTouch=true)
//	func (sm *SessionManager) ValidateSession(ctx context.Context, userID, sessionID string) (SessionToken, error) {
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//
//	    // HGet automatically extends TTL to DefaultTTL (30 minutes)
//	    token, found := sm.cache.HGet(ctx, key, sessionID)
//	    if !found {
//	        return SessionToken{}, errors.New("session not found or expired")
//	    }
//
//	    return token, nil
//	}
//
//	// GetActiveSessions returns all active sessions for a user
//	func (sm *SessionManager) GetActiveSessions(ctx context.Context, userID string) ([]SessionToken, error) {
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//	    sessions, err := sm.cache.HGetAll(ctx, key)
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    tokens := make([]SessionToken, 0, len(sessions))
//	    for _, token := range sessions {
//	        tokens = append(tokens, token)
//	    }
//	    return tokens, nil
//	}
//
//	// Logout revokes a specific session
//	func (sm *SessionManager) Logout(ctx context.Context, userID, sessionID string) error {
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//	    return sm.cache.HDel(ctx, key, sessionID)
//	}
//
//	// LogoutAll revokes all sessions for a user
//	func (sm *SessionManager) LogoutAll(ctx context.Context, userID string) error {
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//	    return sm.cache.Delete(ctx, key)
//	}
//
//	// ExtendSession manually extends a session (for premium features)
//	func (sm *SessionManager) ExtendSession(ctx context.Context, userID, sessionID string, extension time.Duration) error {
//	    key := fmt.Sprintf("user:%s:sessions", userID)
//	    success, err := sm.cache.HExpire(ctx, key, sessionID, extension)
//	    if err != nil {
//	        return err
//	    }
//	    if !success {
//	        return errors.New("session not found")
//	    }
//	    return nil
//	}
//
// Distributed Lock Example: Prevent Concurrent Job Processing
//
//	func ProcessJobSafely(cache cache.Cache[any], jobID string) error {
//	    ctx := context.Background()
//	    lockKey := fmt.Sprintf("job:%s:lock", jobID)
//
//	    // Acquire distributed lock
//	    lockCtx, cancel, err := cache.LockWithContext(ctx, lockKey)
//	    if err != nil {
//	        return fmt.Errorf("failed to acquire lock: %w", err)
//	    }
//	    defer cancel() // Always release lock
//
//	    // Process job safely - no other worker can process this job
//	    select {
//	    case <-lockCtx.Done():
//	        return errors.New("lock lost during processing")
//	    default:
//	        // Do work
//	        processJob(jobID)
//	        return nil
//	    }
//	}
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"google.golang.org/protobuf/proto"
)

// Lock errors
var (
	// ErrNotLocked is returned when a lock cannot be acquired.
	ErrNotLocked = errors.New("lock not acquired")
)

// Cache is the interface that defines the caching operations.
type Cache[T any] interface {
	// Returns the value for the given key.
	// If the key is not found, it returns nil and false.
	Get(ctx context.Context, key string) (T, bool)
	// Sets the value for the given key.
	// If the key already exists, it returns an error.
	Set(ctx context.Context, key string, value T) error
	// Deletes the value for the given key.
	// If the key is not found, it returns an error.
	Delete(ctx context.Context, key string) error
	// DeleteMulti deletes multiple keys at once.
	// Returns the number of keys that were deleted.
	DeleteMulti(ctx context.Context, keys ...string) (int64, error)
	// Sets the expiration time for the given key.
	Expire(ctx context.Context, key string, ttl time.Duration) error
	// Sets the value for the given key with a specific TTL.
	SetWithTTL(ctx context.Context, key string, value T, ttl time.Duration) error

	// Increment increments the integer value of a key by delta.
	// Returns the new value.
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	// Decrement decrements the integer value of a key by delta.
	// Returns the new value.
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	// HSet sets the value of a field in a hash.
	HSet(ctx context.Context, key string, field string, value T) error

	// HSetWithTTL sets the value of a field in a hash with individual field expiration (Valkey 9.0+).
	// Each field in the same hash can have different expiration times.
	//
	// Use Case: Multi-Device Session Management
	//
	// Example - Store multiple login sessions for a user, each with device-specific expiration:
	//   userKey := "user:123:sessions"
	//   cache.HSetWithTTL(ctx, userKey, "session:mobile", mobileToken, 1*time.Hour)
	//   cache.HSetWithTTL(ctx, userKey, "session:web", webToken, 8*time.Hour)
	//   cache.HSetWithTTL(ctx, userKey, "session:desktop", desktopToken, 30*24*time.Hour)
	//
	// Benefits:
	//   - Mobile session expires after 1 hour of inactivity
	//   - Web session expires after 8 hours
	//   - Desktop session persists for 30 days
	//   - All sessions stored together under one key
	//   - Each expires independently
	HSetWithTTL(ctx context.Context, key string, field string, value T, ttl time.Duration) error

	// HGet gets the value of a field in a hash.
	// If ApplyTouch is enabled in CacheConfig, this uses HGETEX to automatically extend
	// the field's TTL to DefaultTTL on each access (Valkey 9.0+).
	//
	// Use Case: Keep Active Sessions Alive
	//
	// Example with ApplyTouch enabled:
	//   // Create cache with ApplyTouch = true, DefaultTTL = 30 minutes
	//   cache := NewCache(&CacheConfig{ApplyTouch: true, DefaultTTL: 30*time.Minute})
	//
	//   // Store session with initial 30 minute TTL
	//   cache.HSetWithTTL(ctx, "user:123:sessions", "session:abc", token, 30*time.Minute)
	//
	//   // Every time you validate the session, TTL extends to 30 minutes from now
	//   token, found := cache.HGet(ctx, "user:123:sessions", "session:abc")
	//   // Session stays alive as long as user is active
	//
	// Example without ApplyTouch (fixed expiration):
	//   // Create cache with ApplyTouch = false
	//   cache := NewCache(&CacheConfig{ApplyTouch: false})
	//
	//   // Session expires exactly after initial TTL, regardless of access
	//   cache.HSetWithTTL(ctx, "user:123:sessions", "session:api", apiToken, 1*time.Hour)
	HGet(ctx context.Context, key string, field string) (T, bool)

	// HGetAll gets all fields and values in a hash.
	// Useful for retrieving all active sessions for a user.
	//
	// Example:
	//   sessions, _ := cache.HGetAll(ctx, "user:123:sessions")
	//   for sessionID, token := range sessions {
	//       fmt.Printf("Session %s: %v\n", sessionID, token)
	//   }
	HGetAll(ctx context.Context, key string) (map[string]T, error)

	// HDel deletes one or more fields from a hash.
	// Use for revoking specific sessions without affecting others.
	//
	// Example - Logout from specific device:
	//   cache.HDel(ctx, "user:123:sessions", "session:mobile")
	//   // Mobile logged out, but web and desktop sessions remain active
	HDel(ctx context.Context, key string, fields ...string) error

	// HExpire sets expiration time on an existing hash field (Valkey 9.0+).
	// Returns true if the field exists and expiration was set.
	// Use for manually extending or setting TTL on existing sessions.
	//
	// Example - Extend premium user session:
	//   success, _ := cache.HExpire(ctx, "user:123:sessions", "session:web", 24*time.Hour)
	//   if success {
	//       log.Println("Premium user session extended to 24 hours")
	//   }
	HExpire(ctx context.Context, key string, field string, ttl time.Duration) (bool, error)

	// HTTL returns the remaining TTL of a hash field in seconds.
	// Returns:
	//   - Positive value: seconds remaining until expiration
	//   - -1: field exists but has no expiration
	//   - -2: field doesn't exist
	//
	// Example - Check session expiration:
	//   ttl, _ := cache.HTTL(ctx, "user:123:sessions", "session:mobile")
	//   switch {
	//   case ttl > 3600:
	//       log.Println("Session has more than 1 hour remaining")
	//   case ttl > 0:
	//       log.Printf("Session expires in %d seconds", ttl)
	//   case ttl == -2:
	//       log.Println("Session not found or expired")
	//   }
	HTTL(ctx context.Context, key string, field string) (int64, error)

	// LockWithContext acquires a distributed lock for the given key using the Redlock algorithm.
	// This method WAITS until the lock is acquired or context is canceled.
	//
	// Features (when using RemoteCacheValkey with valkeylock):
	//   - Implements Redlock algorithm for distributed safety
	//   - Automatic lock extension - lock is renewed periodically
	//   - Context cancellation - returned context auto-cancels if lock is lost
	//   - Cluster-aware - works across Valkey cluster/sentinel nodes
	//
	// Returns:
	//   - lockCtx: A new context that is canceled when the lock is lost
	//   - cancel: Function to release the lock - MUST be called when done
	//   - error: Error if lock cannot be acquired
	//
	// Use Case 1: Prevent Concurrent Processing
	//
	// Example - Ensure only one worker processes a job:
	//   ctx := context.Background()
	//   lockCtx, cancel, err := cache.LockWithContext(ctx, "job:123:lock")
	//   if err != nil {
	//       return err
	//   }
	//   defer cancel() // Always release lock
	//
	//   // Process job safely - no other worker can acquire this lock
	//   processJob(lockCtx, "job:123")
	//
	//   // If lockCtx is canceled, the lock was lost (Valkey down, network issue)
	//   select {
	//   case <-lockCtx.Done():
	//       return errors.New("lock lost during processing")
	//   default:
	//       // Job completed successfully
	//   }
	//
	// Use Case 2: Critical Section Protection
	//
	// Example - Atomic multi-step operation:
	//   lockCtx, cancel, err := cache.LockWithContext(ctx, "user:123:balance:lock")
	//   if err != nil {
	//       return err
	//   }
	//   defer cancel()
	//
	//   // Read-Modify-Write safely
	//   balance := getBalance(lockCtx, "user:123")
	//   balance += amount
	//   updateBalance(lockCtx, "user:123", balance)
	//   // No race condition - lock prevents concurrent modifications
	//
	// Use Case 3: Leader Election
	//
	// Example - Ensure only one instance runs scheduled task:
	//   lockCtx, cancel, err := cache.LockWithContext(ctx, "cron:daily-report:lock")
	//   if err != nil {
	//       log.Println("Another instance is running the report")
	//       return nil
	//   }
	//   defer cancel()
	//
	//   generateDailyReport(lockCtx)
	//
	// Important Notes:
	//   - Always defer cancel() to ensure lock is released
	//   - Check lockCtx.Done() for long-running operations
	//   - Lock is automatically renewed until cancel() is called
	//   - For RemoteCacheValkey: uses valkeylock with Redlock algorithm
	//   - For LocalCacheRistretto: simple in-memory lock (NOT distributed)
	LockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error)

	// TryLockWithContext attempts to acquire a lock without waiting.
	// Returns ErrNotLocked immediately if the lock is already held by another process.
	//
	// Use Case: Non-Blocking Lock Acquisition
	//
	// Example - Skip processing if already running:
	//   lockCtx, cancel, err := cache.TryLockWithContext(ctx, "task:cleanup:lock")
	//   if errors.Is(err, ErrNotLocked) {
	//       log.Println("Cleanup already running, skipping...")
	//       return nil
	//   }
	//   if err != nil {
	//       return err
	//   }
	//   defer cancel()
	//
	//   runCleanupTask(lockCtx)
	//
	// Example - Rate limiting with locks:
	//   lockCtx, cancel, err := cache.TryLockWithContext(ctx, fmt.Sprintf("rate:user:%s", userID))
	//   if errors.Is(err, ErrNotLocked) {
	//       return errors.New("too many requests, please retry later")
	//   }
	//   defer cancel()
	//
	//   processRequest(lockCtx)
	//
	// Difference from LockWithContext:
	//   - LockWithContext: WAITS for lock to become available
	//   - TryLockWithContext: FAILS immediately if lock is held
	TryLockWithContext(ctx context.Context, key string) (context.Context, context.CancelFunc, error)
}

// CacheConfig is the configuration for the cache.
type CacheConfig[T any] struct {
	// local/remote/cluster, default is local
	Mode            string
	CacheName       string
	RemoteCacheAddr string
	// Default time to live for the key. See also ApplyTouch
	DefaultTTL  time.Duration
	MaxElements uint64
	// Set this to true in order to extend the TTL of the key
	ApplyTouch bool
	SerDe      SerDe[T]
}

type SerDe[T any] interface {
	Serialize(value T) ([]byte, error)
	Deserialize(data []byte) (T, error)
}

type JSONSerDe[T any] struct{}

func (s *JSONSerDe[T]) Serialize(value T) ([]byte, error) {
	return json.Marshal(value)
}

func (s *JSONSerDe[T]) Deserialize(data []byte) (T, error) {
	var value T
	return value, json.Unmarshal(data, &value)
}

func NewJSONSerDe[T any]() *JSONSerDe[T] {
	return &JSONSerDe[T]{}
}

type ProtoSerDe[T proto.Message] struct{}

func (s *ProtoSerDe[T]) Serialize(value T) ([]byte, error) {
	return proto.Marshal(value)
}

func (s *ProtoSerDe[T]) Deserialize(data []byte) (T, error) {
	var value T
	err := proto.Unmarshal(data, value)
	return value, err
}

// NewCache creates a new cache instance based on the provided configuration.
func NewCache[T any](cacheCfg *CacheConfig[T]) (Cache[T], error, func()) {
	if cacheCfg.Mode == "local" {
		return NewLocalCacheRistretto(cacheCfg)
	}
	return NewRemoteCacheValkey(cacheCfg)
}
