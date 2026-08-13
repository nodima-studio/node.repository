package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	runnersdk "github.com/nodima-studio/nodima-sdk/go"
	runnerv1 "github.com/nodima-studio/nodima-sdk/runner/v1"
)

type runner struct{}

func (runner) Run(ctx context.Context, input runnersdk.Input, output runnersdk.Output) error {
	initialize, err := input.Next(ctx)
	if err != nil {
		return err
	}
	if initialize.Type != runnerv1.MessageInitialize {
		return fmt.Errorf("Add Prefix expected initialize, got %q", initialize.Type)
	}
	column := strings.TrimSpace(initialize.Config["column"])
	if column == "" {
		return errors.New("Add Prefix requires a column")
	}
	ready := runnerv1.NewMessage(runnerv1.MessageReady)
	ready.ExecutionID = initialize.ExecutionID
	ready.NodeID = initialize.NodeID
	if err := output.Emit(ctx, ready); err != nil {
		return err
	}
	for {
		message, err := input.Next(ctx)
		if errors.Is(err, io.EOF) {
			return errors.New("Add Prefix input ended before input_end")
		}
		if err != nil {
			return err
		}
		switch message.Type {
		case runnerv1.MessageInputBatch:
			batch := *message.Batch
			found := false
			for index := range batch.Columns {
				if batch.Columns[index].Name != column {
					continue
				}
				found = true
				if batch.Columns[index].Type != runnerv1.DataTypeString {
					return fmt.Errorf("Add Prefix column %q is not a string", column)
				}
				values := append([]string(nil), batch.Columns[index].String...)
				for row := range values {
					if len(batch.Columns[index].Valid) == 0 || batch.Columns[index].Valid[row] {
						values[row] = initialize.Config["prefix"] + values[row]
					}
				}
				batch.Columns[index].String = values
			}
			if !found {
				return fmt.Errorf("Add Prefix input has no column %q", column)
			}
			result := runnerv1.NewMessage(runnerv1.MessageOutputBatch)
			result.ExecutionID = initialize.ExecutionID
			result.NodeID = initialize.NodeID
			result.PortID = "output"
			result.Batch = &batch
			if err := output.Emit(ctx, result); err != nil {
				return err
			}
		case runnerv1.MessageInputEnd:
			completed := runnerv1.NewMessage(runnerv1.MessageCompleted)
			completed.ExecutionID = initialize.ExecutionID
			completed.NodeID = initialize.NodeID
			return output.Emit(ctx, completed)
		default:
			return fmt.Errorf("Add Prefix cannot handle message %q", message.Type)
		}
	}
}

func main() { runnersdk.Main(runner{}) }
