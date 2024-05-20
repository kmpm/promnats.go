package main

import (
	"context"

	"github.com/kmpm/promnats.go/internal/apps/discovery"
)

func main() {

	discovery.Run(context.Background())

}
