
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
curl -X POST --data "no=A004&userId=9876543210&title=put+test&duration=20&extrainfo={\"info\":\"put+extrainfo\"}&op=<=&value=450000" http://localhost:9453/put

# HOST: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
# Params:
#     no => define mission like id is unique. (REQUIRE)
#     userId => difine the mission is belone to who. (REQUIRE)
#     title => a string decribe the mission. (REQUIRE)
#     duration => How long will be send to compare. (REQUIRE)
#     extrainfo => unexpect data, format to JSON. (REQUIRE)
#     op => Operator to compare value. (REQUIRE)
#     value => a value to compare with this mission. (REQUIRE)
# Response:
#     Success:  OK
#     Params Wrong: Parameter is wrong! need: no, userId, title, duration, extrainfo, op, value
```

## Data format to NSQ

```json
{"No":"A003","Title":"put test","UserID":"9876543210","Schema":{"Duration":2,"Extrainfo":"{\"info\":\"put extrainfo\"}","Op":"<=","Value":"450000"}}
```

```json
{
    "No":"A003",
    "Title":"put test",
    "UserID":"9876543210",
    "Schema":{
        "Duration":2,
        "Extrainfo":"{\"info\":\"put extrainfo\"}",
        "Op":"<=",
        "Value":"450000"
    }
}
```