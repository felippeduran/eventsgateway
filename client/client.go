// eventsgateway
// https://github.com/topfreegames/eventsgateway
//
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2018 Top Free Games <backend@tfgco.com>

package client

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	pb "github.com/topfreegames/protos/eventsgateway/grpc/generated"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

// GRPCClient struct
type GRPCClient struct {
	config        *viper.Viper
	client        pb.GRPCForwarderClient
	logger        log.FieldLogger
	serverAddress string
	topic         string
}

func (g *GRPCClient) sendEvent(name, topic string, props map[string]string) error {
	g.logger.WithFields(log.Fields{
		"operation": "eventRequest",
		"name":      name,
	}).Debug("getting event request")

	req := &pb.Event{
		Id:        uuid.NewV4().String(),
		Name:      name,
		Topic:     topic,
		Props:     props,
		Timestamp: time.Now().UnixNano() / int64(time.Millisecond),
	}

	_, err := g.client.SendEvent(context.Background(), req)
	return err
}

// Send sends an event to another server via grpc using the client's configured topic
func (g *GRPCClient) Send(name string, props map[string]string) error {
	l := g.logger.WithFields(log.Fields{
		"operation": "send",
		"event":     name,
	})
	l.Info("sending event")
	err := g.sendEvent(name, g.topic, props)
	if err != nil {
		l.WithError(err).Error("send event failed")
	}
	l.Info("successfully sended event")
	return err
}

// SendToTopic sends an event to another server via grpc using an explicit topic
func (g *GRPCClient) SendToTopic(name, topic string, props map[string]string) error {
	l := g.logger.WithFields(log.Fields{
		"operation": "sendToTopic",
		"event":     name,
		"topic":     topic,
	})
	l.Info("sending event")
	err := g.sendEvent(name, topic, props)
	if err != nil {
		l.WithError(err).Error("send event failed")
	}
	l.Info("successfully sended event")
	return err
}

func (g *GRPCClient) configure(configPrefix string, client pb.GRPCForwarderClient) error {
	if configPrefix != "" && !strings.HasSuffix(configPrefix, ".") {
		configPrefix = strings.Join([]string{configPrefix, "."}, "")
	}
	g.topic = g.config.GetString(fmt.Sprintf("%sclient.kafkatopic", configPrefix))
	if g.topic == "" {
		return errors.New("no kafka topic informed")
	}

	g.serverAddress = g.config.GetString(fmt.Sprintf("%sclient.grpc.serveraddress", configPrefix))
	if g.serverAddress == "" {
		return errors.New("no grpc server address informed")
	}

	g.logger = g.logger.WithFields(log.Fields{
		"source":     "eventsgateway/client",
		"topic":      g.topic,
		"serverAddr": g.serverAddress,
	})

	if client != nil {
		g.client = client
		return nil
	}

	g.logger.WithFields(log.Fields{
		"operation":    "configure",
		"serverAdress": g.serverAddress,
	}).Info("connecting to grpc server")
	conn, err := grpc.Dial(g.serverAddress, grpc.WithInsecure())
	fmt.Println(conn, err)
	if err != nil {
		return err
	}
	g.client = pb.NewGRPCForwarderClient(conn)
	return nil
}

// NewClient returns a new GRPCClient
func NewClient(configPrefix string, config *viper.Viper, logger log.FieldLogger, client pb.GRPCForwarderClient) (*GRPCClient, error) {
	g := &GRPCClient{
		config: config,
		logger: logger,
	}
	err := g.configure(configPrefix, client)
	if err != nil {
		g.logger.WithError(err).Error("failed to configure client")
		return nil, err
	}
	return g, nil
}
