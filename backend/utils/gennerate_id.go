package utils

import (
	"math/rand"
	"sync"
	"time"
)

var (
	seedOnce     sync.Once
	counter      uint32
	counterMutex sync.Mutex
)

func GenerateID() uint64 {
	seedOnce.Do(func() {
		rand.New(rand.NewSource(time.Now().UnixNano()))
	})

	// 获取当前时间戳（纳秒级别）
	timestamp := time.Now().UnixNano()

	// 生成一个随机数
	randomNum := rand.Uint32()

	// 使用计数器增加唯一性
	counterMutex.Lock()
	counter++
	if counter > 0xFFFF {
		counter = 0
	}
	counterMutex.Unlock()

	// 将时间戳的低32位、随机数的32位和计数器的16位组合成一个64位的uint64
	id := uint64(timestamp&0xFFFFFFFF) | (uint64(randomNum) << 32) | (uint64(counter) << 48)

	// 确保生成的ID为18位
	// 18位数字的最小值为100000000000000000，即 0x16345785D8A0000
	min18Digit := uint64(100000000000000000)
	max18Digit := uint64(999999999999999999)
	if id < min18Digit {
		id = min18Digit + (id % (max18Digit - min18Digit + 1))
	}
	if id > max18Digit {
		id = min18Digit + (id % (max18Digit - min18Digit + 1))
	}

	return id
}
