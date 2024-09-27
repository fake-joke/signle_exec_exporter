package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestTimedMainExecution(t *testing.T) {
	// 设置执行间隔
	interval := 2 * time.Second
	// 设置执行次数
	executions := 3

	// 捕获标准输出
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	for i := 0; i < executions; i++ {
		start := time.Now()
		main()
		time.Sleep(interval - time.Since(start))
	}

	// 恢复标准输出
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// 验证输出
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != executions {
		t.Errorf("Expected %d lines of output, got %d", executions, len(lines))
	}

	for i, line := range lines {
		if !strings.Contains(line, "Main function executed at:") {
			t.Errorf("Line %d does not contain expected output", i+1)
		}
	}

	// 验证执行间隔
	for i := 1; i < len(lines); i++ {
		t1, _ := time.Parse(time.RFC3339, strings.TrimPrefix(lines[i-1], "Main function executed at: "))
		t2, _ := time.Parse(time.RFC3339, strings.TrimPrefix(lines[i], "Main function executed at: "))
		actualInterval := t2.Sub(t1)
		if actualInterval < interval-100*time.Millisecond || actualInterval > interval+100*time.Millisecond {
			t.Errorf("Interval between execution %d and %d was %v, expected about %v", i, i+1, actualInterval, interval)
		}
	}
}
