package starfish_sdk

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/exp/constraints"
	"gorm.io/gorm"
)

// Waiting 信号阻塞等待
func Waiting() os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	sign := <-ch
	return sign
}

// Context 创建一个上下文
func Context() context.Context {
	return context.Background()
}

// Point 返回数据指针
func Point[P any](p P) *P {
	return &p
}

// If 如果条件成立，返回r1，不成立返回r2
func If[R any](logic bool, r1, r2 R) R {
	if logic {
		return r1
	} else {
		return r2
	}
}

// If 如果条件成立，返回r1，不成立返回r2
func IfCall[R any](logic bool, r1, r2 func() R) R {
	if logic {
		return r1()
	} else {
		return r2()
	}
}

// NilDefault 如果data为nil, 则返回default
func NilDefault[P any](data *P, defaultVal P) P {
	if data == nil {
		return defaultVal
	} else {
		return *data
	}
}

// FindOne 找到任何一个是否满足条件, 如果未找到返回一个nil
func FindOne[S ~[]E, E *any](datas S, f func(item E) bool) E {
	for i := range datas {
		data := datas[i]
		r := f(data)
		if r {
			return data
		}
	}
	return nil
}

// FindAny 找到任何一个是否满足条件
func FindAny[S ~[]E, E *any](s S, f func(item E) bool) bool {
	return FindOne(s, f) != nil
}

// Filter 数据过滤
func Filter[S ~[]E, E any](datas S, f func(item E) bool) S {
	result := make(S, 0, len(datas))
	for i := range datas {
		data := datas[i]
		r := f(data)
		if r {
			result = append(result, data)
		}
	}
	return result
}

// Match 条件匹配
func Match[T comparable](v T, matchers ...T) bool {
	for i := range matchers {
		if v == matchers[i] {
			return true
		}
	}
	return false
}

// Sum 数值累加
func Sum[S ~[]P, P constraints.Integer | constraints.Float](datas S) P {
	var result P
	for i := range datas {
		data := datas[i]
		result += data
	}
	return result
}

// SumCall 数值累加，自定义累加过程
func SumCall[S ~[]P, P constraints.Integer | constraints.Float, R any](datas S, f func(r R, p P) R) R {
	var result R
	for i := range datas {
		result = f(result, datas[i])
	}
	return result
}

// Map 数据转换
func Map[S1 ~[]S1P, S2 ~[]S2P, S1P any, S2P any](datas S1, f func(item S1P) S2P) S2 {
	result := make(S2, 0, len(datas))
	for i := range datas {
		data := datas[i]
		r := f(data)
		result = append(result, r)
	}
	return result
}

// 安全调用Goroutine
func Go(call func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				Log().AddCallerSkip(1).Error(err)
			}
		}()
		call()
	}()
}

// Check 检查逻辑是否为true
func Check(expr bool, throwErr ...error) {
	if !expr {
		if throwErr != nil {
			panic(throwErr[0])
		} else {
			panic(errors.New("检查失败"))
		}
	}
}

// CheckNil 检查不为Nil
func CheckNil[V *any](v V, throwErr ...error) {
	if v == nil {
		if throwErr != nil {
			panic(throwErr[0])
		} else {
			panic(errors.New("检查失败"))
		}
	}
}

// CheckError 检查err不为nil
func CheckError(err error, throwErr ...error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	if err != nil {
		if throwErr != nil {
			panic(throwErr[0])
		} else {
			panic(err)
		}
	}
}
