
# How to build

### Step 1

use 'git clone' clone project to src in $GO_PATH

### step 2

Get into project folder './MonitorCore'

### step 3

Build executable file with command 'go build'

### step 4

You will get executable file named "MonitorCore" or "MonitorCore.exe" (in windows) then enjoy it.


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

#### Put mission

```
[POST] http://{HOST}:{PORT}/put
```
```bash
# curl in shell example
curl -X POST --data "id=1&owner=edwin&no=A0000001&name=有人中了大樂透&duration=1&duration_unit=MINUTE&op=>&value=200000000&extrainfo={"buy_date":"today", "win_date":"yesterday"}" http://localhost:9453/put

# HOST: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
# Params:
#     id => user_panel_detail.id (REQUIRE)
#     owner => user_panel_detail.owner (REQUIRE)
#     no => user_panel_detail.api_no (REQUIRE)
#     name => user_panel_detail.name (REQUIRE)
#     duration => user_panel_detail.duration (REQUIRE)
#     duration_unit => user_panel_detail.duration_unit (REQUIRE)
#     op => Operator to compare value. (REQUIRE)
#     value => a value to compare with this mission. (REQUIRE)
#     extrainfo => user_panel_detail.extrainfo (REQUIRE)

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
{"id":1,"owner":"edwin","no":"A0000001","name":"有人中了大樂透","duration":1,"duration_unit":"MINUTE","op":">","value":"200000000","extrainfo":"{\"buy_date\":\"today\", \"win_date\":\"yesterday\"}"}
```

```json
{
    "id": 1,
    "owner": "edwin",
    "no": "A0000001",
    "name": "有人中了大樂透",
    "duration": 1,
    "duration_unit": "MINUTE",
    "op": ">",
    "value": "200000000",
    "extrainfo": "{\"buy_date\":\"today\", \"win_date\":\"yesterday\"}"
}
```