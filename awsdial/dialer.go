package awsdial

import (
	"bufio"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/session-manager-plugin/src/datachannel"
	"github.com/aws/session-manager-plugin/src/log"
	"github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session"
	_ "github.com/aws/session-manager-plugin/src/sessionmanagerplugin/session/portsession"
	"github.com/google/uuid"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
)

type Dialer struct {
	Client    *ssm.Client
	Region    string
	localPort int
	mut       sync.Mutex
}

func (d *Dialer) Dial(ctx context.Context, target string, port int) (net.Conn, error) {
	d.mut.Lock()
	defer d.mut.Unlock()

	if d.localPort <= 0 {
		err := d.dial(ctx, target, port)
		if err != nil {
			return nil, fmt.Errorf(": %w", err)
		}
	}

	return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", d.localPort))
}

func (d *Dialer) dial(ctx context.Context, target string, port int) error {
	in := &ssm.StartSessionInput{
		DocumentName: aws.String("AWS-StartPortForwardingSession"),
		Target:       aws.String(target),
		Parameters: map[string][]string{
			"portNumber": {strconv.Itoa(port)},
		},
	}

	start, err := d.Client.StartSession(ctx, in)
	if err != nil {
		return fmt.Errorf("calling StartSession API: %w", err)
	}

	ep, err := ssm.NewDefaultEndpointResolver().ResolveEndpoint(d.Region, ssm.EndpointResolverOptions{})
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	ssmSession := &session.Session{
		DataChannel: &datachannel.DataChannel{},
		SessionId:   *start.SessionId,
		StreamUrl:   *start.StreamUrl,
		TokenValue:  *start.TokenValue,
		Endpoint:    ep.URL,
		ClientId:    uuid.NewString(),
		TargetId:    target,
	}

	r, newStdout, err := os.Pipe()
	if err != nil {
		return fmt.Errorf(": %w", err)
	}

	oldStdout := os.Stdout
	os.Stdout = newStdout
	defer func() {
		os.Stdout = oldStdout
	}()

	go func() {
		err = ssmSession.Execute(log.Logger(false, ssmSession.ClientId))
		if err != nil {
			panic(fmt.Sprintf("%+v", err))
		}
	}()

	scan := bufio.NewScanner(r)
	re := regexp.MustCompile(`Port (\d+) opened for sessionId ([^.]+).`)

	// TODO: we probably need to keep reading this rather than bailing after getting port
	for scan.Scan() {
		matches := re.FindStringSubmatch(scan.Text())
		if len(matches) == 0 {
			continue
		}

		d.localPort, _ = strconv.Atoi(matches[1])
		break
	}

	return nil
}
