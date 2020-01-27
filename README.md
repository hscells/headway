# headway

_Monitor the progress of stuff, and alert you if it goes wrong._

To install:

```bash
go get -u github.com/hscells/headway/cmd/hw
```

To create/update a progress bar, send a `PUT` request to the server:

```
curl -X PUT "localhost:7777/?secret=ABC-XYZ-123&name=example&current=10&total=100&comment=test"
```

Only the `comment` parameter is optional. Both the `current` and `total` must be send each bar update. The `name` parameter is also always required.
The `secret` parameter will be made available once you log in through Slack.

An additional, optional, `message` parameter can be used to send a direct message to yourself on Slack, as an alert. For example:

```
curl -X PUT "localhost:7777/?secret=ABC-XYZ-123&message=task%20crashed%20on%20line%2042"
```

## todo

 - documentation for slack setup