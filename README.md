# headway

_Monitor the progress of stuff!_

To create/update a progress bar, send a `PUT` request to the server:

```
curl -X PUT "localhost:7777/p/send?name=example&current=10&total=100&comment=test"
```

Only the `comment` parameter is optional. Both the `current` and `total` must be send each bar update. The `name` parameter is also always required.