package main

import (
	"context"

	"github.com/kmpm/promnats.go/internal/apps/dockerproxy"
)

func main() {

	dockerproxy.Run(context.Background())

}
