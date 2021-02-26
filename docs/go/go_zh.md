# Go相关特性

# Defer
go语言的延迟语句，在当前函数结束前会进行调用，在return执行完后执行
。
defer执行顺序为后入先出，类似栈  
如以下代码, 执行结果为4,3,2,1,0：
```go
for i := 0; i < 5; i++ {
    defer fmt.Printf("%d ", i)
}
```

# New
用new关键字创建对象时，它不会初始化内存，只是把它用zero表示
new (T)返回的是*T，即T的指针。
如：
```go
p := new (T) // type *T
var p T // type T
```

# Make
用make关键字创建slice, map, array，并且返回初始化后的对象
，而不是指针。

# Array
数组，赋值给另一个数组时，是复制所有的值；传递数组给函数时，是传递了该数组的copy，而不是传递数组的指针过去。

# Slice
slice在数组的基础上做了一层封装；
slice中有一个指向数组的指针，所以将slice赋值给另一个变量时，指向的是同一个数组；传递给函数使用时，对slice的修改
会影响数组实际内容，对原数据可见。
slice长度可变。

# map
map和slice一样，维护一个指向底层数据结构的指针，所以传递map后，修改map会对原数据有影响。

# Blank
```go
import _ "net/http/pprof"
```
导入该包的目的主要是该包init方法中注册了可以提供调试信息、获取界面数据的http handle，不需要包的实际功能。

# Goroutine/Channel

