package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/spf13/viper"
)

type monitorCore struct {
	id                 int64
	name               string
	host               string
	port               int
	heartbeatTimer     int64
	heartbeatKeyPrefix string
	missionKeyPrefix   string
	role               string
	memberlist         []string
	cli                *clientv3.Client
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
		id:                 time.Now().UnixNano(),
		name:               viper.GetString("name"),
		host:               viper.GetString("etcd_host"),
		port:               viper.GetInt("etcd_port"),
		heartbeatTimer:     viper.GetInt64("heartbeat_timer"),
		heartbeatKeyPrefix: viper.GetString("heartbeat_key_prefix"),
		missionKeyPrefix:   viper.GetString("mission_key_prefix"),
		role:               "member",
		cli:                cli,
	}

	mc.Stdout("Starting monitor...")
	mc.Stdout(fmt.Sprintf("ID: %d", mc.id))
	defer mc.cli.Close()
	go func() {
		mc.SendHeartBeat()
		for range time.Tick(time.Second * time.Duration(mc.heartbeatTimer)) {
			mc.SendHeartBeat()
			mc.UpdateMemberList()
			mc.Stdout(fmt.Sprintf("Who am I? [%s]%d", mc.role, mc.id))
		}
	}()
	mc.Watch()
}

func (mc *monitorCore) Watch() {
	mc.Stdout(fmt.Sprintf("Watching by '%s'", mc.missionKeyPrefix))
	rch := mc.cli.Watch(context.Background(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}

func (mc *monitorCore) SendHeartBeat() {
	mc.Stdout("Send heartbeat to etcd")
	key := fmt.Sprintf("%s/%d", mc.heartbeatKeyPrefix, mc.id)
	resp, err := mc.cli.Grant(context.TODO(), mc.heartbeatTimer+1)
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
	resp, _ := mc.cli.Get(context.TODO(), mc.heartbeatKeyPrefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		mc.memberlist = append(mc.memberlist, string(ev.Value[:]))
	}
	sort.Strings(mc.memberlist)
	if strconv.FormatInt(mc.id, 10) == mc.memberlist[0] {
		mc.role = "master"
	}
}

func (mc *monitorCore) Stdout(msg string) {
	t := time.Now()
	fmt.Printf("[%s] %s\n", t.Format("2006-01-02 15:04:05"), msg)
}
