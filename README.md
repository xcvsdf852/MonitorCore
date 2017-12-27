
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

```bash
curl -X POST --data "no=A003&userId=9876543210&title=put+test&duration=20&extrainfo={\"info\":\"put+extrainfo\"} &op=<=&value=450000" http://{HOST}:{HOST}/put

# host: Which machine that you started.
# PORT: A param named "HTTP_HOST" define in config.yaml
```