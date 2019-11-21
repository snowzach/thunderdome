package redis

import (
	"encoding/json"

	"github.com/go-redis/redis"

	"git.coinninja.net/backend/thunderdome/tdrpc"
)

type channel struct {
	sub    *redis.PubSub
	client *client
}

// Publish will publish a message to any listeners on the channel
func (c *client) Publish(bucket string, key string, in interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return c.client.Publish(c.prefix+bucket+Delimeter+key, string(data)).Err()
}

// Subscribe will listen for messages on a particular key
func (c *client) Subscribe(bucket string, key string) (*channel, error) {
	sub := c.client.Subscribe(c.prefix + bucket + Delimeter + key)
	_, err := sub.Receive()
	if err != nil {
		return nil, err
	}
	return &channel{
		sub:    sub,
		client: c,
	}, nil
}

// Channel get a channel of transaction
func (c *channel) Channel() <-chan *tdrpc.LedgerRecord {
	dataChan := make(chan *tdrpc.LedgerRecord)
	go func() {
		subChan := c.sub.Channel()
		for {
			// Get a message
			m := <-subChan
			// Channel closed
			if m == nil {
				close(dataChan)
				return
			}
			//
			var data = new(tdrpc.LedgerRecord)
			err := json.Unmarshal([]byte(m.Payload), data)
			if err != nil {
				c.client.logger.Errorw("Could not unmarshal", "error", err, "payload", m.Payload)
				continue
			}
			dataChan <- data
		}
	}()
	return dataChan
}

func (c *channel) Close() {
	c.sub.Close()
}
