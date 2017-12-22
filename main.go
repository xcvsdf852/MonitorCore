package main

import (
	"MonitorCore/lib"
	"time"
)

func main() {
	e := etcd.New("127.0.0.1", 2379, 5)
	e.SendHeartBeat()
	go func() {
		for range time.Tick(time.Second * time.Duration(e.HeartbeatTempo)){
			e.SendHeartBeat()
		}
	}()
	e.Watch("mission/")
}
