package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/tgoodwin/sleeve/pkg/replay"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var inputFilePath string

func init() {
	flag.StringVar(&inputFilePath, "input", "", "Path to the input file")
	flag.Parse()
}

type DummyController struct {
	client client.Client
}

var _ reconcile.Reconciler = &DummyController{}

func (dc *DummyController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func main() {
	if inputFilePath == "" {
		fmt.Println("Please provide the input file path using the -input flag")
		return
	}
	fmt.Printf("Input file path: %s\n", inputFilePath)
	file, err := os.Open(inputFilePath)
	if err != nil {
		fmt.Printf("Error opening input file: %v\n", err)
		return
	}
	defer file.Close()
	data, err := os.ReadFile(inputFilePath)
	if err != nil {
		fmt.Printf("Error reading input file: %v\n", err)
		return
	}
	lineCount := 0
	for _, b := range data {
		if b == '\n' {
			lineCount++
		}
	}
	fmt.Printf("Number of lines in the file: %d\n", lineCount)

	builder := &replay.Builder{}
	// trace can contain information from multiple different controller
	if err := builder.FromTrace(data); err != nil {
		fmt.Printf("Error building replay from trace: %v\n", err)
		return
	}
	// GetPlayer constructs a replayer for a specific controller in the trace
	revisionHarness, err := builder.ConstructHarness("FakeRevision")
	if err != nil {
		fmt.Printf("Error constructing replay harness: %v\n", err)
		return
	}

	// after constructing the replay factory, we need to
	reconciler := &DummyController{
		client: revisionHarness.ReplayClient(),
	}

	player := revisionHarness.Load(reconciler)
	if err := player.Run(); err != nil {
		fmt.Printf("Error running replay: %v\n", err)
		return
	}
}
