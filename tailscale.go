package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"tailscale.com/client/tailscale"
)

func tsDiscover() {
	tailscale.I_Acknowledge_This_API_Is_Unstable = true

	client := tailscale.NewClient(*tailnet, tailscale.APIKey(os.Getenv("TS_API_KEY")))

	devices, err := client.Devices(context.Background(), tailscale.DeviceAllFields)
	if err != nil {
		log.Fatal(err)
	}

	for _, dev := range devices {
		*targetFlag = append(*targetFlag, dev.Hostname)
	}
}
