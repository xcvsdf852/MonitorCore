package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/gorilla/handlers"
	"github.com/nsqio/go-nsq"
	"github.com/spf13/viper"
)

type schema struct {
	Duration  int64
	Extrainfo string
	Op        string
	Value     string
}

type mission struct {
	No     string
	Title  string
	UserID string
	Schema schema
}

type monitorCore struct {
	id                 int64
	name               string
	duration           int64
	heartbeatKeyPrefix string
	missionKeyPrefix   string
	lastExecTimePrefix string
	role               string
	memberList         []string
	missionList        map[string]mission
	lastExecTimeList   map[string]int64
	cli                *clientv3.Client
	nsq                *nsq.Producer
}

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags)
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connecting to NSQd...")
	producer, err := nsq.NewProducer(fmt.Sprintf("%s:%d", viper.GetString("NSQ_HOST"), viper.GetInt("NSQ_PORT")), nsq.NewConfig())
	if err != nil {
		panic(err)
	}

	log.Println("Connecting to ETCD...")
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("http://%s:%d", viper.GetString("ETCD_HOST"), viper.GetInt("ETCD_PORT"))},
		DialTimeout: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	mc := &monitorCore{
		id:                 time.Now().UnixNano(),
		name:               viper.GetString("PROJECT_NAME"),
		duration:           viper.GetInt64("DURATION"),
		heartbeatKeyPrefix: fmt.Sprintf("%s/%s", viper.GetString("ETCD_PREFIX"), "heartbeat"),
		missionKeyPrefix:   fmt.Sprintf("%s/%s", viper.GetString("ETCD_PREFIX"), "mission"),
		lastExecTimePrefix: fmt.Sprintf("%s/%s", viper.GetString("ETCD_PREFIX"), "lastExecTime"),
		role:               "member",
		cli:                cli,
		nsq:                producer,
	}
	defer mc.cli.Close()
	defer mc.nsq.Stop()

	readInAllMission(mc)
	log.Printf("Total mission : %d\n", len(mc.missionList))
	go watchMission(mc)
	go watchLastExecTime(mc)
	go mc.listenHTTPRequest()
	mc.SendHeartBeat()
	for range time.Tick(time.Second * time.Duration(mc.duration)) {
		mc.SendHeartBeat()
		mc.UpdateMemberList()
		log.Printf("Who am I? [%s]%d\n", mc.role, mc.id)
		if mc.role == "master" {
			mc.CheckMission()
		}
	}
}

func (mc *monitorCore) putRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		req.ParseForm()
		no := req.Form.Get("no")
		userID := req.Form.Get("userId")
		title := req.Form.Get("title")
		duration := req.Form.Get("duration")
		extrainfo := req.Form.Get("extrainfo")
		op := req.Form.Get("op")
		value := req.Form.Get("value")

		if no == "" || title == "" || userID == "" || duration == "" || extrainfo == "" || op == "" || value == "" {
			io.WriteString(w, "Parameter is wrong! need: no, userId, title, duration, extrainfo, op, value")
		} else {
			dur, _ := strconv.ParseInt(duration, 10, 64)
			m := mission{
				No:     no,
				Title:  title,
				UserID: userID,
				Schema: schema{
					Duration:  dur,
					Extrainfo: extrainfo,
					Op:        op,
					Value:     value,
				},
			}
			missionKey := fmt.Sprintf("%s/%s/%s", mc.missionKeyPrefix, m.UserID, m.No)
			mToJSON, _ := json.Marshal(m)
			log.Println("put mission:\n" + string(mToJSON[:]) + "\n")
			_, err := mc.cli.Put(context.TODO(), missionKey, string(mToJSON[:]))
			if err != nil {
				log.Println(err)
			}
		}
		io.WriteString(w, "OK")
	} else {
		io.WriteString(w, "Wrong HTTP request method.")
	}
}

func (mc *monitorCore) infoRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		mcInfo := make(map[string]interface{})
		mcInfo["id"] = mc.id
		mcInfo["name"] = mc.name
		mcInfo["role"] = mc.role
		mcInfo["duration"] = mc.duration
		mcInfo["missionCount"] = len(mc.missionList)
		mcInfo["memberList"] = mc.memberList
		mcInfoStr, _ := json.Marshal(mcInfo)
		io.WriteString(w, string(mcInfoStr[:]))
	} else {
		io.WriteString(w, "Wrong HTTP request method.")
	}
}

func (mc *monitorCore) listenHTTPRequest() {
	r := http.NewServeMux()
	r.Handle("/put", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(mc.putRequest)))
	r.Handle("/info", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(mc.infoRequest)))
	http.ListenAndServe(":"+viper.GetString("HTTP_PORT"), handlers.CompressHandler(r))
}

func watchMission(mc *monitorCore) {
	log.Printf("Watching mission by '%s'", mc.missionKeyPrefix)
	rch := mc.cli.Watch(context.Background(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Printf("Type：%s\nKey： %s\n", ev.Type, string(ev.Kv.Key))
			switch ev.Type {
			case clientv3.EventTypeDelete:
				delete(mc.missionList, string(ev.Kv.Key))
			case clientv3.EventTypePut:
				m := mission{}
				err := json.Unmarshal(ev.Kv.Value, &m)
				if err != nil {
					log.Println(err)
				}
				mc.missionList[string(ev.Kv.Key)] = m
			default:
				log.Printf("Unexpect type: %s", ev.Type)
			}
		}
	}
}

func watchLastExecTime(mc *monitorCore) {
	log.Printf("Watching lastExecTime by '%s'", mc.lastExecTimePrefix)
	rch := mc.cli.Watch(context.Background(), mc.lastExecTimePrefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Printf("Type：%s\nKey：%s; data：%s\n", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))
			switch ev.Type {
			case clientv3.EventTypeDelete:
				delete(mc.lastExecTimeList, string(ev.Kv.Key))
			case clientv3.EventTypePut:
				mc.lastExecTimeList[string(ev.Kv.Key)], _ = strconv.ParseInt(string(ev.Kv.Value), 10, 64)
			default:
				log.Printf("Unexpect type: %s", ev.Type)
			}
		}
	}
}

func (mc *monitorCore) CheckMission() {
	log.Println("Checking mission...")
	index := 1
	total := len(mc.missionList)
	log.Printf("Total mission: %d\n", total)
	for key, m := range mc.missionList {
		lastExecKey := fmt.Sprintf("%s/%s/%s", mc.lastExecTimePrefix, m.UserID, m.No)
		fmt.Printf("Index: %d/%d\nKey: %s", index, total, key)
		fmt.Printf("\nLast exec time:%d\n", mc.lastExecTimeList[lastExecKey])
		if time.Now().Unix()-mc.lastExecTimeList[lastExecKey] >= m.Schema.Duration {
			_, err := mc.cli.Put(context.TODO(), lastExecKey, strconv.FormatInt(time.Now().Unix(), 10))
			if err != nil {
				log.Println(err)
			}
			mToJSON, _ := json.Marshal(m)
			mc.nsq.Publish(viper.GetString("NSQ_TOPIC"), mToJSON)
		}
		index++
	}
}

func (mc *monitorCore) SendHeartBeat() {
	log.Println("Send heartbeat to ETCD")
	key := fmt.Sprintf("%s/%d", mc.heartbeatKeyPrefix, mc.id)
	resp, err := mc.cli.Grant(context.TODO(), mc.duration+1)
	if err != nil {
		log.Fatal(err)
	}

	_, err = mc.cli.Put(context.TODO(), key, fmt.Sprintf("%d", mc.id), clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
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

func readInAllMission(mc *monitorCore) {
	log.Printf("Read in all mission with key: %s\n", mc.missionKeyPrefix)
	mc.missionList = make(map[string]mission)
	mc.lastExecTimeList = make(map[string]int64)
	resp, _ := mc.cli.Get(context.TODO(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		m := mission{}
		err := json.Unmarshal(ev.Value[:], &m)
		if err != nil {
			log.Println(err)
		}
		respLastExecTime, _ := mc.cli.Get(context.TODO(), mc.lastExecTimePrefix, clientv3.WithPrefix())
		for _, evLastExecTime := range respLastExecTime.Kvs {
			mc.lastExecTimeList[string(evLastExecTime.Key[:])], _ = strconv.ParseInt(string(evLastExecTime.Value[:]), 10, 64)
		}
		mc.missionList[string(ev.Key)] = m
	}
}
