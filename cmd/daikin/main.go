package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudkucooland/go-daikin"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:  "daikin",
		Usage: "CLI for interacting with Daikin Skyport API",
		Commands: []*cli.Command{
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all registered devices",
				Action:  listDevices,
			},
			{
				Name:    "info",
				Aliases: []string{"get"},
				Usage:   "Get full status of a device",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Device ID (defaults to first device if omitted)"},
				},
				Action: getDeviceInfo,
			},
			{
				Name:  "set-temp",
				Usage: "Set heat and cool setpoints",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Device ID"},
					&cli.FloatFlag{Name: "cool", Value: 24.0, Usage: "Cool setpoint in Celsius"},
					&cli.FloatFlag{Name: "heat", Value: 18.0, Usage: "Heat setpoint in Celsius"},
					&cli.IntFlag{Name: "mode", Value: int(daikin.ModeCool), Usage: "System Mode (1:Heat, 2:Cool, 3:Auto)"},
				},
				Action: setTemps,
			},
			{
				Name:  "schedule",
				Usage: "Revert device to cloud schedule",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Device ID"},
				},
				Action: setSchedule,
			},
			{
				Name:  "dr",
				Usage: "Set Demand Response offset",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "id", Usage: "Device ID"},
					&cli.FloatFlag{Name: "offset", Value: 0, Usage: "Degree offset (C)"},
					&cli.BoolFlag{Name: "on", Value: false, Usage: "Enable/Disable DR"},
				},
				Action: setDR,
			},
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cmd.Run(ctx, os.Args); err != nil {
		slog.Error("execution failed", "error", err)
		os.Exit(1)
	}
}

// Helper to initialize client and find the right device
func getClientAndDevice(ctx context.Context, id string) (*daikin.Client, *daikin.Device, error) {
	client, err := daikin.New(ctx, os.Getenv("DAIKIN_EMAIL"), os.Getenv("DAIKIN_DEVELOPER_KEY"), os.Getenv("DAIKIN_API_KEY"))
	if err != nil {
		return nil, nil, err
	}

	if len(client.Devices) == 0 {
		return nil, nil, fmt.Errorf("no devices found")
	}

	if id == "" {
		return client, &client.Devices[0], nil
	}

	for i := range client.Devices {
		if client.Devices[i].ID == id {
			return client, &client.Devices[i], nil
		}
	}

	return nil, nil, fmt.Errorf("device ID %s not found", id)
}

func listDevices(ctx context.Context, cmd *cli.Command) error {
	client, err := daikin.New(ctx, os.Getenv("DAIKIN_EMAIL"), os.Getenv("DAIKIN_DEVELOPER_KEY"), os.Getenv("DAIKIN_API_KEY"))
	if err != nil {
		return err
	}

	fmt.Printf("%-24s %-20s %s\n", "DEVICE ID", "NAME", "MODEL")
	for _, d := range client.Devices {
		fmt.Printf("%-24s %-20s %s\n", d.ID, d.Name, d.Model)
	}
	return nil
}

func getDeviceInfo(ctx context.Context, cmd *cli.Command) error {
	_, dev, err := getClientAndDevice(ctx, cmd.String("id"))
	if err != nil {
		return err
	}

	info, err := dev.GetInfo(ctx)
	if err != nil {
		return err
	}

	out, _ := json.MarshalIndent(info, "", "  ")
	fmt.Printf("Device: %s (%s)\n%s\n", dev.Name, dev.ID, string(out))
	return nil
}

func setTemps(ctx context.Context, cmd *cli.Command) error {
	_, dev, err := getClientAndDevice(ctx, cmd.String("id"))
	if err != nil {
		return err
	}

	mode := daikin.SystemMode(cmd.Int("mode"))
	cool := cmd.Float("cool")
	heat := cmd.Float("heat")

	fmt.Printf("Setting %s to mode %d (H: %.1f C, C: %.1f C)...\n", dev.Name, mode, heat, cool)
	return dev.SetTemps(ctx, mode, heat, cool)
}

func setSchedule(ctx context.Context, cmd *cli.Command) error {
	_, dev, err := getClientAndDevice(ctx, cmd.String("id"))
	if err != nil {
		return err
	}

	fmt.Printf("Reverting %s to schedule...\n", dev.Name)
	return dev.SetModeSchedule(ctx)
}

func setDR(ctx context.Context, cmd *cli.Command) error {
	_, dev, err := getClientAndDevice(ctx, cmd.String("id"))
	if err != nil {
		return err
	}

	active := cmd.Bool("on")
	offset := cmd.Float("offset")

	fmt.Printf("Setting Demand Response on %s: active=%v, offset=%.1f\n", dev.Name, active, offset)
	return dev.SetDemandResponse(ctx, active, offset)
}
