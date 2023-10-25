# task
sync wait group with cancel context

## Install
```
go get -u github.com/dstgo/task
```

## Usage

with cancel
```go
ctx := context.Background()
t, cancel := task.New(ctx)
defer cancel(nil)
t.Add(func(ctx context.Context) error) {
    for i := 0;;i++ {
	    fmt.Println("w",i*2+1)	
    }
	return nil
})

err := t.Run()
```

with timeout
```go
ctx := context.Background()
t, cancel := task.WithTimeout(ctx, time.Second)
defer cancel(nil)
t.Add(func(ctx context.Context) error) {
    for i := 0;;i++ {
        fmt.Println("w",i*2+1)
    }
    return nil
})

err := t.Run()
```