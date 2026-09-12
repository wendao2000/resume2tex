package main

import (
	"context"
	"fmt"
	"os"

	"github.com/wendao2000/resume2tex/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "resume2tex:", err)
		os.Exit(1)
	}
}
