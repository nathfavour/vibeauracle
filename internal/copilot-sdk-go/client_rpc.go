package copilot

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func parseCliUrl(url string) (string, int) {
	// Remove protocol if present
	cleanUrl := regexp.MustCompile(`^https?://`).ReplaceAllString(url, "")

	// Check if it's just a port number
	if matched, _ := regexp.MatchString(`^\d+$`, cleanUrl); matched {
		port, err := strconv.Atoi(cleanUrl)
		if err != nil || port <= 0 || port > 65535 {
			panic(fmt.Sprintf("Invalid port in CLIUrl: %s", url))
		}
		return "localhost", port
	}

	// Parse host:port format
	parts := regexp.MustCompile(`:`).Split(cleanUrl, 2)
	if len(parts) != 2 {
		panic(fmt.Sprintf("Invalid CLIUrl format: %s. Expected 'host:port', 'http://host:port', or 'port'", url))
	}

	host := parts[0]
	if host == "" {
		host = "localhost"
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil || port <= 0 || port > 65535 {
		panic(fmt.Sprintf("Invalid port in CLIUrl: %s", url))
	}

	return host, port
}

func (c *Client) startCLIServer() error {
	args := []string{"--server", "--log-level", c.options.LogLevel}

	// Choose transport mode
	if c.options.UseStdio {
		args = append(args, "--stdio")
	} else if c.options.Port > 0 {
		args = append(args, "--port", strconv.Itoa(c.options.Port))
	}

	// If CLIPath is a .js file, run it with node
	command := c.options.CLIPath
	if strings.HasSuffix(c.options.CLIPath, ".js") {
		command = "node"
		args = append([]string{c.options.CLIPath}, args...)
	}

	c.process = exec.Command(command, args...)

	// Set working directory if specified
	if c.options.Cwd != "" {
		c.process.Dir = c.options.Cwd
	}

	// Set environment if specified
	if len(c.options.Env) > 0 {
		c.process.Env = c.options.Env
	}

	if c.options.UseStdio {
		// For stdio mode, we need stdin/stdout pipes
		stdin, err := c.process.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		stdout, err := c.process.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		stderr, err := c.process.StderrPipe()
		if err != nil {
			return fmt.Errorf("failed to create stderr pipe: %w", err)
		}

		// Read stderr in background
		go func() {
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				// Optionally log stderr
			}
		}()

		if err := c.process.Start(); err != nil {
			return fmt.Errorf("failed to start CLI server: %w", err)
		}

		// Create JSON-RPC client immediately
		c.client = NewJSONRPCClient(stdin, stdout)
		c.setupNotificationHandler()
		c.client.Start()

		return nil
	} else {
		// For TCP mode, capture stdout to get port number
		stdout, err := c.process.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		if err := c.process.Start(); err != nil {
			return fmt.Errorf("failed to start CLI server: %w", err)
		}

		// Wait for port announcement
		scanner := bufio.NewScanner(stdout)
		timeout := time.After(10 * time.Second)
		portRegex := regexp.MustCompile(`listening on port (\d+)`)

		for {
			select {
			case <-timeout:
				return fmt.Errorf("timeout waiting for CLI server to start")
			default:
				if scanner.Scan() {
					line := scanner.Text()
					if matches := portRegex.FindStringSubmatch(line); len(matches) > 1 {
						port, err := strconv.Atoi(matches[1])
						if err != nil {
							return fmt.Errorf("failed to parse port: %w", err)
						}
						c.actualPort = port
						return nil
					}
				}
			}
		}
	}
}

// connectToServer establishes a connection to the server.
func (c *Client) connectToServer() error {
	if c.options.UseStdio {
		// Already connected via stdio in startCLIServer
		return nil
	}

	// Connect via TCP
	return c.connectViaTcp()
}

// connectViaTcp connects to the CLI server via TCP socket.
func (c *Client) connectViaTcp() error {
	if c.actualPort == 0 {
		return fmt.Errorf("server port not available")
	}

	// Create TCP connection with 10 second timeout
	address := net.JoinHostPort(c.actualHost, fmt.Sprintf("%d", c.actualPort))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to CLI server at %s: %w", address, err)
	}

	c.conn = conn

	// Create JSON-RPC client with the connection
	c.client = NewJSONRPCClient(conn, conn)
	c.setupNotificationHandler()
	c.client.Start()

	return nil
}

// setupNotificationHandler configures handlers for session events, tool calls, and permission requests.
func (c *Client) setupNotificationHandler() {
	c.client.SetNotificationHandler(func(method string, params map[string]interface{}) {
		if method == "session.event" {
			// Extract sessionId and event
			sessionID, ok := params["sessionId"].(string)
			if !ok {
				return
			}

			// Marshal the event back to JSON and unmarshal into typed struct
			eventJSON, err := json.Marshal(params["event"])
			if err != nil {
				return
			}

			event, err := UnmarshalSessionEvent(eventJSON)
			if err != nil {
				return
			}

			// Dispatch to session
			c.sessionsMux.Lock()
			session, ok := c.sessions[sessionID]
			c.sessionsMux.Unlock()

			if ok {
				session.dispatchEvent(event)
			}
		}
	})

	c.client.SetRequestHandler("tool.call", c.handleToolCallRequest)
	c.client.SetRequestHandler("permission.request", c.handlePermissionRequest)
}
