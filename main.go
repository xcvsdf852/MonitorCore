package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/spf13/viper"
)

type monitorCore struct {
	id        int64
	name      string
	host      string
	port      int
	heartbeat int64
	keyprefix string
	cli       *clientv3.Client
}

func main() {
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", viper.GetString("etcd_host"), viper.GetInt("etcd_port"))},
		DialTimeout: time.Second,
	})

	if err != nil {
		log.Fatal(err)
	}

	mc := monitorCore{
		id:        time.Now().UnixNano(),
		name:      viper.GetString("name"),
		host:      viper.GetString("etcd_host"),
		port:      viper.GetInt("etcd_port"),
		heartbeat: viper.GetInt64("heart_beat_timer"),
		keyprefix: "/heartbeat/monitorcore",
		cli:       cli,
	}

	defer mc.cli.Close()

	go func() {
		for range time.Tick(time.Second * time.Duration(mc.heartbeat)) {
			mc.SendHeartBeat()
		}
	}()
	mc.MemberList()
	mc.Watch("mission")
}

func (mc *monitorCore) Watch(key string) {
	fmt.Printf("Watching by '%s'\n", key)
	rch := mc.cli.Watch(context.Background(), key, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}

func (mc *monitorCore) SendHeartBeat() {
	fmt.Printf("Send heartbeat\n")
	key := fmt.Sprintf("%s/%d", mc.keyprefix, mc.id)
	resp, err := mc.cli.Grant(context.TODO(), mc.heartbeat+1)
	if err != nil {
		log.Fatal(err)
	}

	_, err = mc.cli.Put(context.TODO(), key, fmt.Sprintf("%d", mc.id), clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
}

func (mc *monitorCore) MemberList() {
	resp, _ := mc.cli.Get(context.TODO(), mc.keyprefix, clientv3.WithPrefix())

	fmt.Printf("Member list:\n")
	for _, ev := range resp.Kvs {
		fmt.Printf("%s : %s\n", ev.Key, ev.Value)
	}
}
