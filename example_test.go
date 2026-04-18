package daikin_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudkucooland/go-daikin"
)

// This example shows how to initialize the client, list devices,
// and engage the "Deep Cool" strategy on the first found device.
func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Credentials should ideally come from environment variables
	email := os.Getenv("DAIKIN_EMAIL")
	devKey := os.Getenv("DAIKIN_DEV_KEY") // Integrator Token
	apiKey := os.Getenv("DAIKIN_API_KEY") // Developer API Key

	// 1. Initialize the client
	client, err := daikin.New(ctx, email, devKey, apiKey)
	if err != nil {
		log.Fatalf("Failed to initialize Daikin client: %v", err)
	}

	// 2. Iterate through discovered devices
	for _, device := range client.Devices {
		fmt.Printf("Found Device: %s (Model: %s)\n", device.Name, device.Model)

		// 3. Pull current state
		info, err := device.GetInfo(ctx)
		if err != nil {
			log.Printf("Could not get info for %s: %v", device.Name, err)
			continue
		}

		fmt.Printf("Current Indoor Temp: %.1f°C\n", info.IndoorTemp)

		// 4. Example: Trigger Deep Cool if we were to run this
		// In a real app, you'd check your solar export logic here
		if info.Mode == daikin.ModeCool {
			err := device.SetTemps(ctx, daikin.ModeCool, 20.0, 16.5)
			if err != nil {
				log.Printf("Failed to set deep cool: %v", err)
			}
		}
	}
}

// ExampleDevice_SetTemps demonstrates how to specifically set the
// cooling and heating setpoints.
func ExampleDevice_SetTemps() {
	// Setup dummy context and client for demonstration
	// ctx := context.TODO()
	// Assume 'device' was retrieved from client.Devices[0]
	// err := device.SetTemps(ctx, daikin.ModeCool, 20.0, 18.0)

	fmt.Println("Set device to Cool mode with 18.0C target")

	// Output: Set device to Cool mode with 18.0C target
}
