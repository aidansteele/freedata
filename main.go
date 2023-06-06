package main

import (
	"context"
	"fmt"
	"github.com/aidansteele/freedata/awsdial"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"io"
	"log"
	"os"
	"strconv"
	"tailscale.com/tsnet"
	"tailscale.com/types/logger"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	dialer := &awsdial.Dialer{
		Client: ssm.NewFromConfig(cfg),
		Region: cfg.Region,
	}

	s := &tsnet.Server{
		Dir:      "./freedata",
		Hostname: "freedata",
		Logf:     logger.Discard,
	}
	defer s.Close()

	ln, err := s.ListenFunnel("tcp", ":443")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	fmt.Printf("Listening on https://%v\n", s.CertDomains()[0])

	instanceId := os.Args[1]
	port, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal("Must specify a numeric port")
	}

	for {
		client, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("accepted from %s\n", client.RemoteAddr())

		target, err := dialer.Dial(ctx, instanceId, port)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("dialed to %s\n", target.RemoteAddr())

		go func() {
			defer client.Close()
			io.Copy(client, target)
		}()

		go func() {
			defer target.Close()
			io.Copy(target, client)
		}()
	}
}
