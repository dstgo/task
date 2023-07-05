# task
a simply waitGroup wrap package

## Install
```
go get github.com/dstgo/task
```

## Usage

simple usage

```go
task := NewTask(func(err error) {
    log.Println(err)
})

task.AddJobs(func() {
    time.Sleep(time.Second)
    log.Println(1)
})

task.AddJobs(func() {
    time.Sleep(time.Second * 2)
    log.Println(22)
})

task.AddJobs(func() {
    time.Sleep(time.Second * 3)
    panic("hh")
})

task.Run()
```

usage in for range
```go
task := NewTask(func(err error) {
    log.Println(err)
})

for i := 0; i < 100; i++ {
    i := i
    task.AddJobs(func() {
        time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
        log.Println(i)
    })
}

task.Run()
```