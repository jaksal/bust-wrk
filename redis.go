package main

import (
	"log"
	"sync/atomic"

	"github.com/gomodule/redigo/redis"
)

// Redis : ...
var (
	redisDB redis.Conn
)

// InitRedis ...
func InitRedis(url string) error {
	c, err := redis.Dial("tcp", url)
	if err != nil {
		return err
	}
	if _, err := c.Do("PING"); err != nil {
		return err
	}
	log.Println("connect redis! url=", url)
	redisDB = c
	return nil
}

// FinalRedis :
func FinalRedis() {
	redisDB.Close()
	log.Println("disconnect redis!")
}

// pubsub.
func publish(channel string, msg string) error {
	if _, err := redisDB.Do("PUBLISH", channel, msg); err != nil {
		log.Printf("[PUB] Error! %s: message: %s err:=%s\n", channel, msg, err)
		return err
	}
	log.Printf("[PUB] %s: message: %s\n", channel, msg)
	return nil
}

func subscribe(channel string) {
	conn, err := redis.Dial("tcp", redisURL)
	if err != nil {
		panic(err)
	}
	psc := redis.PubSubConn{Conn: conn}
	defer psc.Close()

	psc.PSubscribe(channel)

	for atomic.LoadInt32(&interrupted) == 0 {
		switch v := psc.Receive().(type) {
		case redis.Message:
			log.Printf("[SUB] %s: message: %s\n", v.Channel, v.Data)
			if exit := ParseTest(string(v.Data)); exit {
				return
			}
		case redis.Subscription:
			log.Printf("[TR  ] %s: %s %d\n", v.Channel, v.Kind, v.Count)
		case error:
			log.Println("err! ", v)
			return
		}
	}
}
