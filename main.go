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

type mission struct {
	ID           string
	Owner        string
	No           string
	Name         string
	Duration     int
	DurationUnit string `json:"duration_unit"`
	Condition    string
	Extrainfo    string
}

type monitorCore struct {
	id                 int64
	name               string
	duration           int64
	heartbeatKeyPrefix string
	missionKeyPrefix   string
	nextExecTimePrefix string
	role               string
	memberList         []string
	missionList        map[string]mission
	nextExecTimeList   map[string]int64
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
		nextExecTimePrefix: fmt.Sprintf("%s/%s", viper.GetString("ETCD_PREFIX"), "nextExecTime"),
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
		if err := req.ParseForm(); err != nil {
			log.Printf("ParseForm() err: %v\n", err)
			errorResponse(w, http.StatusUnprocessableEntity, viper.GetString("ERROR_CODE_UNPROCESSABLE_ENTITY"), viper.GetString("ERROR_MSG_UNPROCESSABLE_ENTITY"))
			return
		}
		id := req.FormValue("id")
		owner := req.FormValue("owner")
		no := req.FormValue("no")
		name := req.FormValue("name")
		durationStr := req.FormValue("duration")
		durationUnit := req.FormValue("duration_unit")
		condition := req.FormValue("condition")
		extrainfo := req.FormValue("extrainfo")

		if id == "" || owner == "" || no == "" || name == "" || durationStr == "" || durationUnit == "" || condition == "" {
			errorResponse(w, http.StatusUnprocessableEntity, viper.GetString("ERROR_CODE_UNPROCESSABLE_ENTITY"), viper.GetString("ERROR_MSG_UNPROCESSABLE_ENTITY"))
		} else {
			duration, _ := strconv.Atoi(durationStr)
			m := mission{
				ID:           id,
				Owner:        owner,
				No:           no,
				Name:         name,
				Duration:     duration,
				DurationUnit: durationUnit,
				Condition:    condition,
				Extrainfo:    extrainfo,
			}
			missionKey := fmt.Sprintf("%s/%s", mc.missionKeyPrefix, m.ID)
			mToJSON, _ := json.Marshal(m)
			log.Println("put mission:\n" + string(mToJSON[:]) + "\n")
			_, err := mc.cli.Put(context.TODO(), missionKey, string(mToJSON[:]))
			if err != nil {
				log.Println(err)
			}
			okResponse(w, make(map[string]interface{}))
		}
	} else {
		errorResponse(w, http.StatusMethodNotAllowed, viper.GetString("ERROR_CODE_HTTP_METHOD_NOT_ALLOWED"), viper.GetString("ERROR_MSG_HTTP_METHOD_NOT_ALLOWED"))
	}
}

func (mc *monitorCore) infoRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		mcInfo := make(map[string]interface{})
		mcInfo["id"] = mc.id
		mcInfo["name"] = mc.name
		mcInfo["role"] = mc.role
		mcInfo["duration"] = mc.duration
		mcInfo["mission_count"] = len(mc.missionList)
		mcInfo["member_list"] = mc.memberList
		okResponse(w, mcInfo)
	} else {
		errorResponse(w, http.StatusMethodNotAllowed, viper.GetString("ERROR_CODE_HTTP_METHOD_NOT_ALLOWED"), viper.GetString("ERROR_MSG_HTTP_METHOD_NOT_ALLOWED"))
	}
}

func errorResponse(w http.ResponseWriter, httpStatus int, errorCode string, msg string, extraInfo ...interface{}) {
	errorRes := make(map[string]interface{})
	errorRes["code"] = errorCode
	errorRes["msg"] = msg
	errorRes["extraInfo"] = extraInfo

	res := make(map[string]interface{})
	res["status"] = "error"
	res["data"] = errorRes

	resStr, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	io.WriteString(w, string(resStr[:]))
}

func okResponse(w http.ResponseWriter, data interface{}) {
	res := make(map[string]interface{})
	res["status"] = "ok"
	res["data"] = data
	resStr, _ := json.Marshal(res)
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, string(resStr[:]))
}

func (mc *monitorCore) listenHTTPRequest() {
	r := http.NewServeMux()
	r.Handle("/put", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(mc.putRequest)))
	r.Handle("/info", handlers.LoggingHandler(os.Stdout, http.HandlerFunc(mc.infoRequest)))
	err := http.ListenAndServe(":"+viper.GetString("HTTP_PORT"), handlers.CompressHandler(r))
	if err != nil {
		log.Fatal(err)
	}
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
	log.Printf("Watching lastExecTime by '%s'", mc.nextExecTimePrefix)
	rch := mc.cli.Watch(context.Background(), mc.nextExecTimePrefix, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			log.Printf("Type：%s\nKey：%s; data：%s\n", ev.Type, string(ev.Kv.Key), string(ev.Kv.Value))
			switch ev.Type {
			case clientv3.EventTypeDelete:
				delete(mc.nextExecTimeList, string(ev.Kv.Key))
			case clientv3.EventTypePut:
				mc.nextExecTimeList[string(ev.Kv.Key)], _ = strconv.ParseInt(string(ev.Kv.Value), 10, 64)
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
		nextExecKey := fmt.Sprintf("%s/%s", mc.nextExecTimePrefix, m.ID)
		fmt.Printf("Index: %d/%d\nKey: %s\n", index, total, key)
		fmt.Printf("Exec timestamp:%d\n", mc.nextExecTimeList[nextExecKey])
		t := time.Unix(mc.nextExecTimeList[nextExecKey], 0)
		fmt.Printf("Exec datetime:%s\n", t.Format("2006-01-02 15:04:05"))
		if mc.nextExecTimeList[nextExecKey] >= 0 && time.Now().Unix() >= mc.nextExecTimeList[nextExecKey] {
			_, err := mc.cli.Put(context.TODO(), nextExecKey, strconv.FormatInt(calcAfterTimestamp(m.Duration, m.DurationUnit), 10))
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
	mc.nextExecTimeList = make(map[string]int64)
	resp, _ := mc.cli.Get(context.TODO(), mc.missionKeyPrefix, clientv3.WithPrefix())
	for _, ev := range resp.Kvs {
		m := mission{}
		err := json.Unmarshal(ev.Value[:], &m)
		if err != nil {
			log.Println(err)
		}
		respNextExecTime, _ := mc.cli.Get(context.TODO(), mc.nextExecTimePrefix, clientv3.WithPrefix())
		for _, evNextExecTime := range respNextExecTime.Kvs {
			mc.nextExecTimeList[string(evNextExecTime.Key[:])], _ = strconv.ParseInt(string(evNextExecTime.Value[:]), 10, 64)
		}
		mc.missionList[string(ev.Key)] = m
	}
}

// MINUTE, HOUR, DAY, WEEK, MONTH
// 每分：本次執行+60 sec
// 每時：每x小時的 00:00
// 每日：每x日的 00:00:00
// 每週：每x週的 週日 00:00:00
// 每月：每x月的 1日 00:00:00
func calcAfterTimestamp(duration int, durationUnit string) int64 {
	now := time.Now()
	switch durationUnit {
	case "MINUTE":
		now = now.Add(time.Duration(duration) * time.Minute)
	case "HOUR":
		now = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.Local)
		now = now.Add(time.Duration(duration) * time.Hour)
	case "DAY":
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		now = now.AddDate(0, 0, duration)
	case "WEEK":
		now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		shiftToDefaultWeekday := int(time.Sunday) - int(now.Weekday())
		now = now.AddDate(0, 0, shiftToDefaultWeekday+(duration*7))
	case "MONTH":
		now = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		now = now.AddDate(0, duration, 0)
	default:
		return -1
	}

	return now.Unix()
}
