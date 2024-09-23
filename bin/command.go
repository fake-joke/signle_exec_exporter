package bin

import (
	"bytes"
	"context"
	"fmt"
	"go_collector/utils"
	"os/exec"
	"strings"
	"time"
)

const CmdTimeout = 10 * time.Second

func RunCommand(filename string, args ...string) ([]byte, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, CmdTimeout)
	defer cancel()

	rootDir := utils.GetBinDir()
	osType := utils.GetOsType()
	switch osType {
	case "linux":
		filename = rootDir + "/" + filename
	case "windows":
		filename = rootDir + "\\" + filename
	default:
		fmt.Println("Unknown OS")
	}
	fmt.Println("cmd:", filename+" "+strings.Join(args, " "))
	cmd := exec.CommandContext(ctx, filename, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, err
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Exec command timeout")
		} else {
			return out, err
		}
	}

	return out, nil
}

func RunCommandAndReturnBytes(filename string, args ...string) bytes.Buffer {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, CmdTimeout)
	defer cancel()

	rootDir := utils.GetBinDir()
	osType := utils.GetOsType()
	switch osType {
	case "linux":
		filename = rootDir + "/" + filename
	case "windows":
		filename = rootDir + "\\" + filename
	default:
		fmt.Println("Unknown OS")
	}
	cmd := exec.CommandContext(ctx, filename, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("Run RunCommandAndReturnBytes Error: ", filename)
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			fmt.Println("Exec command timeout")
		} else {
			fmt.Println("Run RunCommandAndReturnBytes Error: ", filename, err.Error())
		}
	}

	return out
}
