package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/spf13/viper"
)

type monitorCore struct {
	id         int64
	name       string
	host       string
	port       int
	heartbeat  int64
	keyprefix  string
	role       string
	memberlist []string
	cli        *clientv3.Client
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
		role:      "member",
		cli:       cli,
	}

	mc.Stdout(fmt.Sprintf("ID: %d, Starting monitor...\n", mc.id))
	defer mc.cli.Close()
	go func() {
		mc.SendHeartBeat()
		for range time.Tick(time.Second * time.Duration(mc.heartbeat)) {
			mc.SendHeartBeat()
			mc.UpdateMemberList()
			mc.Stdout(fmt.Sprintf("Who am I: %s\n", mc.role))
		}
	}()
	mc.Watch("/mission")
}

func (mc *monitorCore) Watch(key string) {
	mc.Stdout(fmt.Sprintf("Watching by '%s'\n", key))
	rch := mc.cli.Watch(context.Background(), key, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}

func (mc *monitorCore) SendHeartBeat() {
	mc.Stdout("Send heartbeat")
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

func (mc *monitorCore) UpdateMemberList() {
	mc.memberlist = []string{}
	resp, _ := mc.cli.Get(context.TODO(), mc.keyprefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		mc.memberlist = append(mc.memberlist, string(ev.Value[:]))
	}
	sort.Strings(mc.memberlist)
	if(strconv.FormatInt(mc.id, 10) == mc.memberlist[0]){
		mc.role = "master"
	}
}

func (mc *monitorCore) Stdout(msg string) {
	t := time.Now()
	fmt.Printf("[%s] %s\n", t.Format("2017-12-23 23:59:59.000"), msg)
}