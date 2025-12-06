package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"go-cache-server-mini/internal/api/dto"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

type cliState struct {
	addr    string
	timeout time.Duration
	client  *apiClient
}

type apiClient struct {
	baseURL    string
	httpClient *http.Client
}

func newAPIClient(addr string, timeout time.Duration) *apiClient {
	return &apiClient{
		baseURL:    normalizeBase(addr),
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *apiClient) configure(addr string, timeout time.Duration) {
	c.baseURL = normalizeBase(addr)
	c.httpClient.Timeout = timeout
}

func normalizeBase(addr string) string {
	trimmed := strings.TrimRight(addr, "/")
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	return "http://" + trimmed
}

func (c *apiClient) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	var payload io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(b)
	}

	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		data, _ := io.ReadAll(resp.Body)
		var parsed map[string]any
		if len(data) > 0 && json.Unmarshal(data, &parsed) == nil {
			if msg, ok := parsed["error"].(string); ok && msg != "" {
				return fmt.Errorf("%s: %s", resp.Status, msg)
			}
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			text = resp.Status
		}
		return errors.New(text)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	return decoder.Decode(out)
}

func parseValue(input string) (json.RawMessage, error) {
	raw := json.RawMessage(input)
	if json.Valid(raw) {
		return raw, nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func newRootCmd(state *cliState) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "cache-cli",
		Short:         "CLI for go-cache-server-mini",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if state.client == nil {
				state.client = newAPIClient(state.addr, state.timeout)
				return nil
			}
			state.client.configure(state.addr, state.timeout)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&state.addr, "addr", "http://localhost:8080", "API server address")
	cmd.PersistentFlags().DurationVar(&state.timeout, "timeout", 5*time.Second, "HTTP request timeout")

	cmd.AddCommand(
		newPingCmd(state),
		newMetricCmd(state),
		newGetCmd(state),
		newSetCmd(state),
		newSetNXCmd(state),
		newGetSetCmd(state),
		newDelCmd(state),
		newExistsCmd(state),
		newKeysCmd(state),
		newExpireCmd(state),
		newTTLCmd(state),
		newPersistCmd(state),
		newFlushCmd(state),
		newIncrCmd(state),
		newDecrCmd(state),
		newMGetCmd(state),
		newMSetCmd(state),
	)
	return cmd
}

func newPingCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check if the API server is reachable",
		RunE: func(cmd *cobra.Command, args []string) error {
			var res map[string]any
			if err := state.client.do(cmd.Context(), http.MethodGet, "/ping", nil, nil, &res); err != nil {
				return err
			}
			return printJSON(res)
		},
	}
}

func newMetricCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "metric",
		Short: "Fetch Prometheus metrics from the API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := state.client.httpClient.Get(state.client.baseURL + "/metrics")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= http.StatusBadRequest {
				data, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("error fetching metrics: %s", strings.TrimSpace(string(data)))
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func newSetCmd(state *cliState) *cobra.Command {
	var ttl int64
	cmd := &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a value for the given key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := parseValue(args[1])
			if err != nil {
				return fmt.Errorf("invalid value: %w", err)
			}
			req := dto.SetRequest{
				Key:   args[0],
				Value: value,
				TTL:   ttl,
			}
			return state.client.do(cmd.Context(), http.MethodPost, "/set", nil, req, nil)
		},
	}
	cmd.Flags().Int64Var(&ttl, "ttl", 0, "TTL in seconds (omit or 0 to use default)")
	return cmd
}

func newSetNXCmd(state *cliState) *cobra.Command {
	var ttl int64
	cmd := &cobra.Command{
		Use:   "setnx KEY VALUE",
		Short: "Set a value only if the key does not exist",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := parseValue(args[1])
			if err != nil {
				return fmt.Errorf("invalid value: %w", err)
			}
			req := dto.SetRequest{
				Key:   args[0],
				Value: value,
				TTL:   ttl,
			}
			var res struct {
				Success bool `json:"success"`
			}
			if err := state.client.do(cmd.Context(), http.MethodPost, "/setnx", nil, req, &res); err != nil {
				return err
			}
			fmt.Println(res.Success)
			return nil
		},
	}
	cmd.Flags().Int64Var(&ttl, "ttl", 0, "TTL in seconds (omit or 0 to use default)")
	return cmd
}

func newGetCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "get KEY",
		Short: "Get a value by key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var res dto.ValueResponse
			query := url.Values{"key": {args[0]}}
			if err := state.client.do(cmd.Context(), http.MethodGet, "/get", query, nil, &res); err != nil {
				return err
			}
			fmt.Println(string(res.Value))
			return nil
		},
	}
}

func newGetSetCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "getset KEY VALUE",
		Short: "Swap the value and return the previous one",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := parseValue(args[1])
			if err != nil {
				return fmt.Errorf("invalid value: %w", err)
			}
			req := dto.GetSetRequest{
				Key:   args[0],
				Value: value,
			}
			var res dto.ValueResponse
			if err := state.client.do(cmd.Context(), http.MethodPost, "/getset", nil, req, &res); err != nil {
				return err
			}
			fmt.Println(string(res.Value))
			return nil
		},
	}
}

func newDelCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "del KEY",
		Short: "Delete a key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"key": {args[0]}}
			return state.client.do(cmd.Context(), http.MethodDelete, "/del", query, nil, nil)
		},
	}
}

func newExistsCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "exists KEY",
		Short: "Check if a key exists",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var res struct {
				Exists bool `json:"exists"`
			}
			query := url.Values{"key": {args[0]}}
			if err := state.client.do(cmd.Context(), http.MethodGet, "/exists", query, nil, &res); err != nil {
				return err
			}
			fmt.Println(res.Exists)
			return nil
		},
	}
}

func newKeysCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "keys",
		Short: "List all keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			var res struct {
				Keys []string `json:"keys"`
			}
			if err := state.client.do(cmd.Context(), http.MethodGet, "/keys", nil, nil, &res); err != nil {
				return err
			}
			for _, key := range res.Keys {
				fmt.Println(key)
			}
			return nil
		},
	}
}

func newExpireCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "expire KEY TTL",
		Short: "Update TTL for a key (seconds, <=0 deletes)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ttl, err := parseTTL(args[1])
			if err != nil {
				return err
			}
			req := dto.ExpireRequest{
				Key: args[0],
				TTL: ttl,
			}
			return state.client.do(cmd.Context(), http.MethodPost, "/expire", nil, req, nil)
		},
	}
}

func parseTTL(raw string) (int64, error) {
	ttl, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 0, fmt.Errorf("invalid ttl: %w", err)
	}
	return int64(ttl.Seconds()), nil
}

func newTTLCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "ttl KEY",
		Short: "Show remaining TTL in seconds (-1 for persistent)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var res struct {
				TTL int64 `json:"ttl"`
			}
			query := url.Values{"key": {args[0]}}
			if err := state.client.do(cmd.Context(), http.MethodGet, "/ttl", query, nil, &res); err != nil {
				return err
			}
			fmt.Println(res.TTL)
			return nil
		},
	}
}

func newPersistCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "persist KEY",
		Short: "Persist a key (remove TTL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := url.Values{"key": {args[0]}}
			return state.client.do(cmd.Context(), http.MethodPost, "/persist", query, nil, nil)
		},
	}
}

func newFlushCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "flush",
		Short: "Remove all keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return state.client.do(cmd.Context(), http.MethodPost, "/flush", nil, nil, nil)
		},
	}
}

func newIncrCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "incr KEY",
		Short: "Increment an integer value and print the new value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var res dto.ValueResponse
			query := url.Values{"key": {args[0]}}
			if err := state.client.do(cmd.Context(), http.MethodPost, "/incr", query, nil, &res); err != nil {
				return err
			}
			fmt.Println(string(res.Value))
			return nil
		},
	}
}

func newDecrCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "decr KEY",
		Short: "Decrement an integer value and print the new value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var res dto.ValueResponse
			query := url.Values{"key": {args[0]}}
			if err := state.client.do(cmd.Context(), http.MethodPost, "/decr", query, nil, &res); err != nil {
				return err
			}
			fmt.Println(string(res.Value))
			return nil
		},
	}
}

func newMGetCmd(state *cliState) *cobra.Command {
	return &cobra.Command{
		Use:   "mget KEY [KEY...]",
		Short: "Fetch multiple keys",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := dto.MGetRequest{Keys: args}
			var res dto.MGetResponse
			if err := state.client.do(cmd.Context(), http.MethodPost, "/mget", nil, req, &res); err != nil {
				return err
			}
			return printJSON(res.KV)
		},
	}
}

func newMSetCmd(state *cliState) *cobra.Command {
	var ttl int64
	cmd := &cobra.Command{
		Use:   "mset KEY VALUE [KEY VALUE]...",
		Short: "Set multiple keys with the same TTL",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args)%2 != 0 {
				return errors.New("mset requires pairs of KEY VALUE")
			}
			kv := make(map[string]json.RawMessage, len(args)/2)
			for i := 0; i < len(args); i += 2 {
				value, err := parseValue(args[i+1])
				if err != nil {
					return fmt.Errorf("invalid value for key %s: %w", args[i], err)
				}
				kv[args[i]] = value
			}
			req := dto.MSetRequest{
				KV:  kv,
				TTL: ttl,
			}
			return state.client.do(cmd.Context(), http.MethodPost, "/mset", nil, req, nil)
		},
	}
	cmd.Flags().Int64Var(&ttl, "ttl", 0, "TTL in seconds for all keys (omit or 0 to use default)")
	return cmd
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	state := &cliState{}

	args := os.Args[1:]
	interactive, filteredArgs := extractInteractive(args)

	if interactive || len(filteredArgs) == 0 {
		if err := runInteractive(ctx, state, filteredArgs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	rootCmd := newRootCmd(state)
	rootCmd.SetContext(ctx)
	rootCmd.SetArgs(filteredArgs)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func extractInteractive(args []string) (bool, []string) {
	interactive := false
	var filtered []string
	for _, arg := range args {
		if arg == "--interactive" || arg == "-i" {
			interactive = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return interactive, filtered
}

func runInteractive(ctx context.Context, state *cliState, initialArgs []string) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Interactive mode. Type 'help' for usage, 'exit' to quit.")

	cmd := newRootCmd(state)
	cmd.SetContext(ctx)

	if len(initialArgs) > 0 {
		if err := cmd.ParseFlags(initialArgs); err != nil {
			return err
		}
	}

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println()
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		if line == "help" {
			line = "--help"
		}

		args, err := splitArgs(line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse error:", err)
			continue
		}

		cmd.SetArgs(args)

		if err := cmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func splitArgs(line string) ([]string, error) {
	args, err := shlex.Split(line)
	if err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}
	return args, nil
}
