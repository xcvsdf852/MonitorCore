package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/spf13/viper"
)

type schema struct {
	Duration  int
	Extrainfo string
	Op        string
	Value     string
}

type mission struct {
	No     string
	Title  string
	Schema schema
}

type monitorCore struct {
	id                 int64
	name               string
	host               string
	port               int
	heartbeatTimer     int64
	heartbeatKeyPrefix string
	missionKeyPrefix   string
	role               string
	memberList         []string
	missionList        map[string]mission
	cli                *clientv3.Client
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	log.Println("Connecting to ETCD...")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", viper.GetString("etcd_host"), viper.GetInt("etcd_port"))},
		DialTimeout: time.Second,
	})
	if err != nil {
		log.Println(err)
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
	defer mc.cli.Close()

	mc.ReadInAllMission()
	go mc.Watch()
	mc.SendHeartBeat()
	for range time.Tick(time.Second * time.Duration(mc.heartbeatTimer)) {
		mc.SendHeartBeat()
		mc.UpdateMemberList()
		log.Printf("Who am I? [%s]%d\n", mc.role, mc.id)
	}
}

func (mc *monitorCore) Watch() {
	log.Printf("Watching by '%s'", mc.missionKeyPrefix)
	rch := mc.cli.Watch(context.Background(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			m := mission{}
			err := json.Unmarshal(ev.Kv.Value, &m)
			if err != nil {
				log.Println(err)
			}
			mc.missionList[string(ev.Kv.Key)] = m
		}
	}
}

func (mc *monitorCore) CheckMission() {

}

func (mc *monitorCore) SendHeartBeat() {
	key := fmt.Sprintf("%s/%d", mc.heartbeatKeyPrefix, mc.id)
	resp, err := mc.cli.Grant(context.TODO(), mc.heartbeatTimer+1)
	if err != nil {
		log.Println(err)
	}

	_, err = mc.cli.Put(context.TODO(), key, fmt.Sprintf("%d", mc.id), clientv3.WithLease(resp.ID))
	if err != nil {
		log.Println(err)
	}
}

func (mc *monitorCore) UpdateMemberList() {
	mc.memberList = []string{}
	resp, _ := mc.cli.Get(context.TODO(), mc.heartbeatKeyPrefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		mc.memberList = append(mc.memberList, string(ev.Value[:]))
	}
	sort.Strings(mc.memberList)
	if strconv.FormatInt(mc.id, 10) == mc.memberList[0] {
		mc.role = "master"
	}
}

// PrettyPrint something
func PrettyPrint(v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	log.Println(string(b))
}

func (mc *monitorCore) ReadInAllMission() {
	log.Printf("Read in all mission with key: %s\n", mc.missionKeyPrefix)
	mc.missionList = make(map[string]mission)
	resp, _ := mc.cli.Get(context.TODO(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		m := mission{}
		err := json.Unmarshal(ev.Value[:], &m)
		if err != nil {
			log.Println(err)
		}
		mc.missionList[string(ev.Key[:])] = m
	}
}
