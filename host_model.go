package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type hostModelExecutionRequest struct {
	pluginapi.HostModelExecutionRequest
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

type rpcStreamEmitRequest struct {
	StreamID string `json:"stream_id"`
	Payload  []byte `json:"payload,omitempty"`
	Error    string `json:"error,omitempty"`
}

type rpcStreamCloseRequest struct {
	StreamID string `json:"stream_id"`
	Error    string `json:"error,omitempty"`
}

func executeHostModel(exec pluginapi.ExecutorRequest, hostCallbackID, model string, body []byte) (pluginapi.HostModelExecutionResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostModelExecute, hostModelExecutionRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol: hostProtocol(exec),
			ExitProtocol:  hostProtocol(exec),
			Model:         model,
			Stream:        false,
			Body:          body,
			Headers:       cloneHeader(exec.Headers),
			Query:         cloneValues(exec.Query),
			Alt:           exec.Alt,
		},
		HostCallbackID: hostCallbackID,
	})
	if errCall != nil {
		return pluginapi.HostModelExecutionResponse{}, errCall
	}
	var resp pluginapi.HostModelExecutionResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostModelExecutionResponse{}, fmt.Errorf("decode host.model.execute result: %w", errUnmarshal)
	}
	return resp, nil
}

func startHostModelStream(exec pluginapi.ExecutorRequest, hostCallbackID, model string, body []byte) (pluginapi.HostModelStreamResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostModelExecuteStream, hostModelExecutionRequest{
		HostModelExecutionRequest: pluginapi.HostModelExecutionRequest{
			EntryProtocol: hostProtocol(exec),
			ExitProtocol:  hostProtocol(exec),
			Model:         model,
			Stream:        true,
			Body:          body,
			Headers:       cloneHeader(exec.Headers),
			Query:         cloneValues(exec.Query),
			Alt:           exec.Alt,
		},
		HostCallbackID: hostCallbackID,
	})
	if errCall != nil {
		return pluginapi.HostModelStreamResponse{}, errCall
	}
	var resp pluginapi.HostModelStreamResponse
	if errUnmarshal := json.Unmarshal(result, &resp); errUnmarshal != nil {
		return pluginapi.HostModelStreamResponse{}, fmt.Errorf("decode host.model.execute_stream result: %w", errUnmarshal)
	}
	return resp, nil
}

func readHostModelStream(streamID string) (pluginapi.HostModelStreamReadResponse, error) {
	result, errCall := callHost(pluginabi.MethodHostModelStreamRead, pluginapi.HostModelStreamReadRequest{StreamID: streamID})
	if errCall != nil {
		return pluginapi.HostModelStreamReadResponse{}, errCall
	}
	var chunk pluginapi.HostModelStreamReadResponse
	if errUnmarshal := json.Unmarshal(result, &chunk); errUnmarshal != nil {
		return pluginapi.HostModelStreamReadResponse{}, fmt.Errorf("decode host.model.stream_read result: %w", errUnmarshal)
	}
	return chunk, nil
}

func closeHostModelStream(streamID string) error {
	if strings.TrimSpace(streamID) == "" {
		return nil
	}
	_, errCall := callHost(pluginabi.MethodHostModelStreamClose, pluginapi.HostModelStreamCloseRequest{StreamID: streamID})
	return errCall
}

func emitPluginStreamChunk(streamID string, payload []byte) error {
	if strings.TrimSpace(streamID) == "" {
		return fmt.Errorf("plugin stream id is required")
	}
	_, errCall := callHost(pluginabi.MethodHostStreamEmit, rpcStreamEmitRequest{
		StreamID: streamID,
		Payload:  payload,
	})
	return errCall
}

func closePluginStream(streamID, errMsg string) {
	if strings.TrimSpace(streamID) == "" {
		return
	}
	_, _ = callHost(pluginabi.MethodHostStreamClose, rpcStreamCloseRequest{
		StreamID: streamID,
		Error:    strings.TrimSpace(errMsg),
	})
}
