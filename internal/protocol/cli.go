package protocol

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// CLI handles protocol management commands from the command line.
type CLI struct {
	manager *Manager
}

// NewCLI creates a new CLI handler using the global Manager.
// Returns nil if the protocol ecosystem is not initialized.
func NewCLI() *CLI {
	if !IsInitialized() {
		return nil
	}
	return &CLI{manager: GlobalManager()}
}

// Run executes a CLI command with the given args.
// Supported commands: list, start, stop, restart, status, health
func (c *CLI) Run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: x-ui protocol <list|start|stop|restart|status|health> [id]")
	}

	switch args[0] {
	case "list":
		return c.list()
	case "start":
		if len(args) < 2 {
			return fmt.Errorf("usage: x-ui protocol start <id>")
		}
		return c.start(ProtocolID(args[1]))
	case "stop":
		if len(args) < 2 {
			return fmt.Errorf("usage: x-ui protocol stop <id>")
		}
		return c.stop(ProtocolID(args[1]))
	case "restart":
		if len(args) < 2 {
			return fmt.Errorf("usage: x-ui protocol restart <id>")
		}
		return c.restart(ProtocolID(args[1]))
	case "status":
		if len(args) < 2 {
			return fmt.Errorf("usage: x-ui protocol status <id>")
		}
		return c.status(ProtocolID(args[1]))
	case "health":
		if len(args) < 2 {
			return fmt.Errorf("usage: x-ui protocol health <id>")
		}
		return c.health(ProtocolID(args[1]))
	default:
		return fmt.Errorf("unknown protocol command %q: use list, start, stop, restart, status, or health", args[0])
	}
}

func (c *CLI) list() error {
	ids := c.manager.ListProtocols()
	if len(ids) == 0 {
		fmt.Println("No protocols registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tSTATUS\tHEALTHY\tPORT")
	fmt.Fprintln(w, "--\t----\t--------\t------\t-------\t----")

	for _, id := range ids {
		p := c.manager.registry.Get(id)
		if p == nil {
			continue
		}
		info := p.Info()
		status := p.Status()
		healthy := c.healthStatus(p)
		portStr := "-"
		if bp, ok := p.(BaseProtocol); ok && bp.Port() > 0 {
			portStr = fmt.Sprintf("%d", bp.Port())
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%v\t%s\n", id, info.Name, info.Category, status, healthy, portStr)
	}
	return w.Flush()
}

func (c *CLI) start(id ProtocolID) error {
	info := GetProtocolInfo(id)
	if info == nil {
		return fmt.Errorf("unknown protocol: %s", id)
	}
	fmt.Printf("Starting %s (%s)...\n", info.Name, id)
	if err := c.manager.StartProtocol(id); err != nil {
		return fmt.Errorf("failed to start %s: %w", id, err)
	}
	fmt.Printf("✓ %s started successfully\n", info.Name)
	return nil
}

func (c *CLI) stop(id ProtocolID) error {
	info := GetProtocolInfo(id)
	if info == nil {
		return fmt.Errorf("unknown protocol: %s", id)
	}
	fmt.Printf("Stopping %s (%s)...\n", info.Name, id)
	if err := c.manager.StopProtocol(id); err != nil {
		return fmt.Errorf("failed to stop %s: %w", id, err)
	}
	fmt.Printf("✓ %s stopped successfully\n", info.Name)
	return nil
}

func (c *CLI) restart(id ProtocolID) error {
	info := GetProtocolInfo(id)
	if info == nil {
		return fmt.Errorf("unknown protocol: %s", id)
	}
	fmt.Printf("Restarting %s (%s)...\n", info.Name, id)
	if err := c.manager.RestartProtocol(id); err != nil {
		return fmt.Errorf("failed to restart %s: %w", id, err)
	}
	fmt.Printf("✓ %s restarted successfully\n", info.Name)
	return nil
}

func (c *CLI) status(id ProtocolID) error {
	p := c.manager.registry.Get(id)
	if p == nil {
		return fmt.Errorf("protocol %s not found", id)
	}

	info := p.Info()
	status := p.Status()

	fmt.Printf("Protocol:    %s\n", info.Name)
	fmt.Printf("ID:          %s\n", info.ID)
	fmt.Printf("Category:    %s\n", info.Category)
	fmt.Printf("Status:      %s\n", status)
	fmt.Printf("Description: %s\n", info.Description)
	fmt.Printf("Source:      %s\n", info.Source)
	fmt.Printf("Xray Native: %v\n", info.XrayNative)

	if bp, ok := p.(BaseProtocol); ok {
		port := bp.Port()
		if port > 0 {
			fmt.Printf("Port:        %d\n", port)
		} else {
			fmt.Println("Port:        (not configured)")
		}
	}

	if ss, ok := p.(StandaloneService); ok {
		installed := ss.IsInstalled()
		fmt.Printf("Installed:   %v\n", installed)
		fmt.Printf("Service:     %s\n", ss.ServiceName())
	}

	if tw, ok := p.(TransportWrapper); ok {
		supported := tw.SupportedProtocols()
		strs := make([]string, len(supported))
		for i, sp := range supported {
			strs[i] = string(sp)
		}
		fmt.Printf("Supports:    %s\n", strings.Join(strs, ", "))
	}

	cfg, err := p.Config()
	if err == nil && cfg != nil {
		fmt.Printf("Config:      %+v\n", cfg)
	}

	return nil
}

func (c *CLI) health(id ProtocolID) error {
	p := c.manager.registry.Get(id)
	if p == nil {
		return fmt.Errorf("protocol %s not found", id)
	}

	info := p.Info()
	healthy := c.healthStatus(p)

	fmt.Printf("Protocol: %s (%s)\n", info.Name, id)
	fmt.Printf("Status:   %s\n", p.Status())
	fmt.Printf("Healthy:  %v\n", healthy)

	if ss, ok := p.(StandaloneService); ok {
		if err := ss.HealthCheck(); err != nil {
			fmt.Printf("Health:    FAIL - %v\n", err)
			return nil
		}
		fmt.Println("Health:    OK")
	}

	if !healthy {
		return fmt.Errorf("protocol %s is unhealthy", id)
	}
	return nil
}

func (c *CLI) healthStatus(p Protocol) bool {
	status := p.Status()
	if status == StatusRunning {
		return true
	}
	if ss, ok := p.(StandaloneService); ok {
		if ss.IsInstalled() && status != StatusError {
			return ss.HealthCheck() == nil
		}
	}
	return status == StatusRunning
}

// RunCLICommand is the top-level entry point for "x-ui protocol ...".
// It initializes the protocol ecosystem if needed and dispatches the command.
func RunCLICommand(args []string) error {
	cli := NewCLI()
	if cli == nil {
		return fmt.Errorf("protocol ecosystem not initialized")
	}
	return cli.Run(args)
}
