package logger

import (
	"fmt"
	"time"
)

func Info(format string, args ...any) {
	fmt.Printf(
		"[%s] INFO  %s\n",
		time.Now().Format("15:04:05"),
		fmt.Sprintf(format, args...),
	)
}

func Error(format string, args ...any) {
	fmt.Printf(
		"[%s] ERROR %s\n",
		time.Now().Format("15:04:05"),
		fmt.Sprintf(format, args...),
	)
}
