package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/weaveworks/flux/agent"
	"github.com/weaveworks/flux/common/daemon"
	"github.com/weaveworks/flux/common/store"
	"github.com/weaveworks/flux/common/store/etcdstore"
	"github.com/weaveworks/flux/common/version"

	log "github.com/Sirupsen/logrus"
)

func main() {
	var (
		hostIP  string
		network string
	)
	flag.StringVar(&hostIP, "host-ip", "", "IP address for instances with mapped ports")
	flag.StringVar(&network, "network-mode", agent.LOCAL, fmt.Sprintf(`Kind of network to assume for containers (either "%s" or "%s")`, agent.LOCAL, agent.GLOBAL))
	flag.Parse()

	if !agent.IsValidNetworkMode(network) {
		fmt.Fprintf(os.Stderr, "Unknown network mode \"%s\"\n\n", network)
		flag.Usage()
		os.Exit(1)
	}

	log.Infof("flux agent version %s", version.Version())

	hostIpFrom := "argument"

	if hostIP == "" {
		hostIP = os.Getenv("HOST_IP")
		hostIpFrom = `$HOST_IP in environment`
	}

	if hostIP == "" {
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("Unable to determine host IP via hostname: %s", err)
		}
		ip, err := net.ResolveIPAddr("ip", hostname)
		if err != nil {
			log.Fatalf("Unable to determine host IP via hostname: %s", err)
		}
		hostIP = ip.String()
		hostIpFrom = fmt.Sprintf(`resolving hostname '%s'`, hostname)
	}

	log.Infof(`Using host IP address '%s' from %s`, hostIP, hostIpFrom)

	st, err := etcdstore.NewFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	containerUpdates := make(chan agent.ContainerUpdate)
	serviceUpdates := make(chan store.ServiceUpdate)

	conf := agent.SyncInstancesConfig{
		HostIP:  hostIP,
		Network: network,
		Store:   st,

		ContainerUpdates: containerUpdates,
		ServiceUpdates:   serviceUpdates,
	}

	daemon.Main(daemon.Aggregate(
		daemon.Restart(10*time.Second,
			agent.DockerListenerStartFunc(containerUpdates)),
		daemon.Restart(10*time.Second,
			store.WatchServicesStartFunc(st,
				store.QueryServiceOptions{WithContainerRules: true},
				serviceUpdates)),
		conf.StartFunc()))
}
