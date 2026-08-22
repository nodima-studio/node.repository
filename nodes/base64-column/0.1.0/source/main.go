package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	runnersdk "github.com/nodima-studio/nodima-sdk/go"
	runnerv1 "github.com/nodima-studio/nodima-sdk/runner/v1"
)

const (
	configColumn   = "column"
	configAction   = "action"
	configEncoding = "encoding"
	actionEncode   = "encode"
	actionDecode   = "decode"
	encodingStd    = "standard"
	encodingURL    = "url"
)

type runner struct{}

type settings struct {
	column   string
	action   string
	encoding *base64.Encoding
}

func (runner) Run(ctx context.Context, input runnersdk.Input, output runnersdk.Output) error {
	var configured settings
	initialized := false
	for {
		message, err := input.Next(ctx)
		if errors.Is(err, io.EOF) {
			if initialized {
				return errors.New("Base64 Column input ended before input_end")
			}
			return errors.New("Base64 Column input ended before initialize")
		}
		if err != nil {
			return err
		}
		switch message.Type {
		case runnerv1.MessageInitialize:
			if initialized {
				return errors.New("Base64 Column initialized more than once")
			}
			configured, err = parseSettings(message.Config)
			if err != nil {
				return err
			}
			initialized = true
			ready := runnerv1.NewMessage(runnerv1.MessageReady)
			ready.ExecutionID, ready.NodeID = message.ExecutionID, message.NodeID
			if err := output.Emit(ctx, ready); err != nil {
				return err
			}
		case runnerv1.MessageInputBatch:
			if !initialized {
				return errors.New("Base64 Column input_batch received before initialize")
			}
			transformed, err := transform(*message.Batch, configured)
			if err != nil {
				return err
			}
			result := runnerv1.NewMessage(runnerv1.MessageOutputBatch)
			result.ExecutionID, result.NodeID, result.PortID, result.Batch = message.ExecutionID, message.NodeID, "output", &transformed
			if err := output.Emit(ctx, result); err != nil {
				return err
			}
		case runnerv1.MessageInputEnd:
			if !initialized {
				return errors.New("Base64 Column input_end received before initialize")
			}
			completed := runnerv1.NewMessage(runnerv1.MessageCompleted)
			completed.ExecutionID, completed.NodeID = message.ExecutionID, message.NodeID
			return output.Emit(ctx, completed)
		default:
			return fmt.Errorf("Base64 Column cannot handle message %q", message.Type)
		}
	}
}

func parseSettings(config map[string]string) (settings, error) {
	column := strings.TrimSpace(config[configColumn])
	if column == "" {
		return settings{}, errors.New(`Base64 Column requires a non-empty "column" configuration`)
	}
	action := config[configAction]
	if action == "" {
		action = actionEncode
	}
	encodingName := config[configEncoding]
	if encodingName == "" {
		encodingName = encodingStd
	}
	var encoding *base64.Encoding
	switch encodingName {
	case encodingStd:
		encoding = base64.StdEncoding
	case encodingURL:
		encoding = base64.URLEncoding
	default:
		return settings{}, fmt.Errorf("Base64 Column encoding %q must be %q or %q", encodingName, encodingStd, encodingURL)
	}
	if action != actionEncode && action != actionDecode {
		return settings{}, fmt.Errorf("Base64 Column action %q must be %q or %q", action, actionEncode, actionDecode)
	}
	return settings{column: column, action: action, encoding: encoding}, nil
}

func transform(batch runnerv1.Batch, configured settings) (runnerv1.Batch, error) {
	for index, column := range batch.Columns {
		if column.Name != configured.column {
			continue
		}
		transformed, err := transformColumn(column, configured)
		if err != nil {
			return runnerv1.Batch{}, err
		}
		result := batch
		result.Columns = append([]runnerv1.Column(nil), batch.Columns...)
		result.Columns[index] = transformed
		return result, nil
	}
	return runnerv1.Batch{}, fmt.Errorf("Base64 Column input has no column %q", configured.column)
}

func transformColumn(column runnerv1.Column, configured settings) (runnerv1.Column, error) {
	result := runnerv1.Column{Name: column.Name, Type: runnerv1.DataTypeString, Valid: column.Valid, String: make([]string, columnCount(column))}
	for index := range result.String {
		if len(column.Valid) != 0 && !column.Valid[index] {
			continue
		}
		value := valueAt(column, index)
		if configured.action == actionEncode {
			result.String[index] = configured.encoding.EncodeToString(value)
			continue
		}
		decoded, err := configured.encoding.DecodeString(string(value))
		if err != nil {
			return runnerv1.Column{}, fmt.Errorf("Base64 Column cannot decode row %d of column %q: %w", index, column.Name, err)
		}
		result.String[index] = string(decoded)
	}
	return result, nil
}

func columnCount(column runnerv1.Column) int {
	if column.Type == runnerv1.DataTypeBytes {
		return len(column.Bytes)
	}
	return len(column.String)
}

func valueAt(column runnerv1.Column, index int) []byte {
	if column.Type == runnerv1.DataTypeBytes {
		return column.Bytes[index]
	}
	return []byte(column.String[index])
}

func main() { runnersdk.Main(runner{}) }
