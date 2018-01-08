# Prepared before use

1. NSQd started.
2. ETCd started.

# What can you do when MonitorCore started

#### Get Info

```bash
curl http://{HOST}:{HOST}/info

# host: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
```

#### Add mission

```
[POST] http://{HOST}:{PORT}/put
```
```bash
# curl in shell example
curl -X POST --data "id=1&owner=1000001&no=A00001&name=Is anyone hit the Jackpot&duration=10&duration_unit=MINUTE&condition={"op":">", "value":"2000000"}&extrainfo={"date":"2018-01-01"}" http://localhost:9453/put

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
| duration | Integer | execution interval | 10 |
| duration_unit | String | duration unit | MINUTE |
| condition | Json String | A condition to excute mission | {"op":">","value":"2000000"} |
| extrainfo | Json String | mission primary key | {"date":"2018-01-01"} |