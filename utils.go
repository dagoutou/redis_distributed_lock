package redis_distributed_lock

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func GetCurrentProcessID() string {
	return strconv.Itoa(os.Getpid())
}

func GetCurrentGoroutineID() string {
	buf := make([]byte, 128)
	buf = buf[:runtime.Stack(buf, false)]
	stackInfo := string(buf)
	return strings.TrimSpace(strings.Split(strings.Split(stackInfo, "[running]")[0], "goroutine")[1])
}

func GetProcessAdnGoroutineIDStr() string {
	return fmt.Sprintf("%s_%s", GetCurrentProcessID(), GetCurrentGoroutineID())
}
