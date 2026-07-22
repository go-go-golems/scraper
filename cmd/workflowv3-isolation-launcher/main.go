package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	cgroup := flag.String("cgroup", "", "preconfigured cgroup v2 directory")
	flag.Parse()
	arguments := flag.Args()
	clean := filepath.Clean(*cgroup)
	if !filepath.IsAbs(clean) || !strings.HasPrefix(clean, "/sys/fs/cgroup/") || len(arguments) == 0 || !filepath.IsAbs(arguments[0]) {
		fmt.Fprintln(os.Stderr, "invalid isolation launcher invocation")
		os.Exit(2)
	}
	if err := os.WriteFile(filepath.Join(clean, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "isolation launcher could not join cgroup")
		os.Exit(3)
	}
	if err := syscall.Exec(arguments[0], arguments, []string{"LANG=C.UTF-8", "PATH=/nonexistent"}); err != nil {
		fmt.Fprintln(os.Stderr, "isolation launcher could not execute sandbox")
		os.Exit(4)
	}
}
