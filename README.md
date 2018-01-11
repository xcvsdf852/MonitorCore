# Prepared before use

1. NSQd started.
2. ETCd started.

# Config

* Priority：
    1. Environment
        * please use "BBOS_MC_" as key prefix, like: [PROJECT_NAME]: BBOS_MC_PROJECT_NAME=TEST
    2. Config File
        * Use flag "-c" to definde config file path.
        * If you do not use "-c" difinde will read default file named: "config.yaml" in the same folder.
    3. Default config
        * if step 2's config file can not read or not exist will use default config

* Default Config setting as below:
```yaml
PROJECT_NAME: MonitorCore
DURATION: 15
ETCD_HOST: localhost
ETCD_PORT: 2379
ETCD_PREFIX: /BBOS/MonitorCore
NSQ_HOST: localhost
NSQ_PORT: 4150
NSQ_TOPIC: BBOS_TO_ME
HTTP_PORT: 9453
ERROR_CODE_HTTP_METHOD_NOT_ALLOWED: 12000001
ERROR_MSG_HTTP_METHOD_NOT_ALLOWED: "Invalid HTTP request method."
ERROR_CODE_UNPROCESSABLE_ENTITY: 120000002
ERROR_MSG_UNPROCESSABLE_ENTITY: "Invalid HTTP request parameter."
```
* Exameple to use
```bash
# use default config
./MonitorCore

# use flag to read config
./MonitorCore -c "/tmp/myConfig.yaml"

# use Environment
BBOS_MC_HTTP_PORT=9455 ./MonitorCore

# Mix above is fine to use
# depend on Priority: 1.Environment 2.Flag 3.Default config
BBOS_MC_HTTP_PORT=9455 ./MonitorCore -c "/tmp/myConfig.yaml"
```

# What can you do when MonitorCore started

#### Get Info

```bash
curl http://{HOST}:{HOST}/info

# host: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
# Response:
{
    "data": {
        "duration": 15,
        "id": 1515488607694174496,
        "member_list": [
            "1515488607694174496"
        ],
        "mission_count": 1,
        "name": "TEST",
        "role": "master",
        "version": "0.0.2"
    },
    "status": "ok"
}
```

#### Add/Edit mission

```
[POST] http://{HOST}:{PORT}/put
```
```bash
# curl in shell example
curl -X POST --data 'id=1&owner=1000001&no=A00001&name=Is anyone hit the Jackpot&duration=10&duration_unit=MINUTE&condition={"op":">", "value":"2000000"}&extrainfo={"date":"2018-01-01"}' http://localhost:9453/put

# HOST: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
# Response:
#     Success:
{
    "data": {},
    "status": "ok"
}

#     Invalid HTTP request method:
{
    "data": {
        "code": "12000001",
        "extraInfo": null,
        "msg": "Invalid HTTP request method."
    },
    "status": "error"
}

#     Invalid HTTP request parameter:
{
    "data": {
        "code": "120000002",
        "extraInfo": null,
        "msg": "Invalid HTTP request parameter."
    },
    "status": "error"
}
```

#### Delete mission

```
[POST] http://{HOST}:{PORT}/del
```
```bash
# delete by mission id
curl -X POST --data "id=1,3,5,7,9" http://localhost:9453/del
# delete by onwer id
curl -X POST --data "owner=100001,100003,100005,100007,100009" http://localhost:9453/del
# delete by mission no
curl -X POST --data "no=A00001,A00003,A00005,A00007,A00009" http://localhost:9453/del

# HOST: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
# Response:
#     Success:
{
    "data": {
        "update_count": 5
    },
    "status": "ok"
}

#     Invalid HTTP request method:
{
    "data": {
        "code": "12000001",
        "extraInfo": null,
        "msg": "Invalid HTTP request method."
    },
    "status": "error"
}

#     Invalid HTTP request parameter:
{
    "data": {
        "code": "120000002",
        "extraInfo": null,
        "msg": "Invalid HTTP request parameter."
    },
    "status": "error"
}
```

## Data format to NSQ

```json
{"id":1,"owner":"1000001","no":"A00001","name":"Is anyone hit the Jackpot","duration":10,"duration_unit":"MINUTE","condition":"{\"op\":\">\", \"value\":\"2000000\"}","extrainfo":"{\"date\":\"2018-01-01\"}"}
```

```json
{
    "id": 1,
    "owner": "1000001",
    "no": "A00001",
    "name": "Is anyone hit the Jackpot",
    "duration": 10,
    "duration_unit": "MINUTE",
    "condition": "{\"op\":\">\", \"value\":\"2000000\"}",
    "extrainfo": "{\"date\":\"2018-01-01\"}"
}
```


## Parameter describe

| Name | Type | Describe | Example |
|---|---|---|---|
| id | Integer | Mission primary key | 1 |
| owner | String | User id by mission owner | 1000001 |
| no | String | Mission's type | A00001 |
| name | String | Mission's name | Is anyone hit the Jackpot? |
| duration | Integer | Execution interval | 10 |
| duration_unit | String | Execution interval unit (MINUTE, HOUR, DAY, WEEK, MONTH) | MINUTE |
| condition | Json String | Excute mission according to condition | {"op":">","value":"2000000"} |
| extrainfo | Json String | More data for API | {"date":"2018-01-01"} |