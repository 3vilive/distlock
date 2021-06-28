package distlock

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
	uuid "github.com/satori/go.uuid"
)

var (
	DefaultExpired     = time.Duration(30 * time.Second)
	DefaultTimeout     = time.Duration(5 * time.Second)
	DefaultSleepPerTry = time.Duration(50 * time.Millisecond)
)

var (
	ErrAcquireLockTimeout = errors.New("acquire lock timeout")
)

type Config struct {
	Expire      time.Duration
	Timeout     time.Duration
	SleepPerTry time.Duration
}

type ApplyConfig = func(c *Config)

func WithExpire(expire time.Duration) ApplyConfig {
	return func(c *Config) {
		c.Expire = expire
	}
}

func WithTimeout(timeout time.Duration) ApplyConfig {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

type Lock struct {
	Redis    redis.Cmdable
	Resource string
	Key      string
	LockID   string
}

func (l *Lock) Release() error {
	if l == nil {
		return nil
	}

	if l.Redis == nil {
		return errors.New("redis is nil")
	}

	script := `
	if redis.call("get", KEYS[1]) == ARGV[1] then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
	`
	result, err := l.Redis.Eval(script, []string{l.Key}, l.LockID).Result()
	if err != nil {
		return err
	}

	resultVal, _ := result.(int64)
	if resultVal == 0 {
		log.Printf("release resource(%s) with mismatch lock id\n", l.Resource)
	}

	return nil
}

// Acquire a lock
//
// quick start:
//	lock, err := distlock.Acquire("my_resource", distlock.WithTimeout(10 * time.Second))
//	if err != nil {
//		return err
//	}
//	defer lock.Release()
//	// do somthing ...
func AcquireWithRedis(resouce string, r redis.Cmdable, applyConfigs ...ApplyConfig) (*Lock, error) {
	conf := Config{
		Expire:      DefaultExpired,
		Timeout:     DefaultTimeout,
		SleepPerTry: DefaultSleepPerTry,
	}

	for _, applyOn := range applyConfigs {
		applyOn(&conf)
	}

	key := fmt.Sprintf("distlock:%s", resouce)
	lockID := uuid.NewV4().String()

	tryAt := time.Now()
	for {
		got, err := r.SetNX(key, lockID, conf.Expire).Result()
		if err != nil {
			return nil, err
		}

		if got {
			return &Lock{
				Redis:    r,
				Resource: resouce,
				Key:      key,
				LockID:   lockID,
			}, nil
		}

		if time.Now().Sub(tryAt) >= conf.Timeout {
			break
		}

		time.Sleep(conf.SleepPerTry)
	}

	return nil, ErrAcquireLockTimeout
}
