package etcd

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"log"
	"time"
	"fmt"
)

type Etcd struct {
	id int64
	host string
	port int
	HeartbeatTempo int64
	endpoints []string
}

func New(host string, port int, heartbeatTempo int64) (etcd * Etcd){
	return &Etcd{
		id: time.Now().UnixNano(),
		host: host,
		port: port,
		HeartbeatTempo: heartbeatTempo,
		endpoints: []string{fmt.Sprintf("http://%s:%d", host, port)},
	}
}

func (etcd *Etcd) Watch(key string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: etcd.endpoints,
		DialTimeout: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	
	rch := cli.Watch(context.Background(), key, clientv3.WithPrefix())
	for wresp := range rch {
		for _, ev := range wresp.Events {
			fmt.Printf("%s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
		}
	}
}

func (etcd *Etcd) SendHeartBeat(){
	key := fmt.Sprintf("MonitorCoreHB_%d", etcd.id)
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: etcd.endpoints,
		DialTimeout: time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	
	// minimum lease TTL is 5-second
	resp, err := cli.Grant(context.TODO(), etcd.HeartbeatTempo + 1)
	if err != nil {
		log.Fatal(err)
	}

	// after 5 seconds, the key 'foo' will be removed
	aliveMsg := fmt.Sprintf("alive at %s", time.Now().Format("2006-01-02 15:04:05"));
	_, err = cli.Put(context.TODO(), key, aliveMsg, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Fatal(err)
	}
}