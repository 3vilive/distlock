# distlock

a simple distributed lock base on redis

install:

```
go get -u github.com/3vilive/distlock
```

usage:

```go
r := redis.NewClient(&redis.Options{
    Addr: "127.0.0.1:6379",
})

lock, err := distlock.AcquireWithRedis("resource_name", r, distlock.WithTimeout(10*time.Second))
if err != nil {
    panic(err)
}
defer lock.Release()

// do somthing ...
```
