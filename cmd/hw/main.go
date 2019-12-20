package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hscells/headway"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"
)

const index string = `
<html>
<head>
<title>headway server</title>
<meta name="viewport" content="width=device-width, initial-scale=1">
<style>
* {
	margin: 0;
	padding: 0;
	font-family: Helvetica, Arial, Sans-Serif;
}
body {
	background: #000;
	color: #fff;
	margin: 1em;
}
progress {
	width: 100%;
	height: 32px;
	background: #222;
}
.box {
	border: 1px solid #222;
	padding: 8px;
}
</style>
</head>
<body>
	{{ range $c, $p := .Progress }}
	<div class="box">
		<div><b>{{ $c }}</b> - {{ $p.CurrentProgress }}/{{ $p.TotalProgress }}</div>
		<div><em>{{ $p.Comment}}</em></div>
		<div>{{ $p.LastUpdate }}</div>
		<div><progress value="{{ $p.CurrentProgress }}" max="{{ $p.TotalProgress }}"></progress></div>
	</div>
	{{ end }}
	<small>last updated: {{ .LastUpdated }}</small>
<script type="text/javascript">
window.setTimeout(function() {
	location.reload();
}, 5000)
</script>
</body>
</html>
`

type data struct {
	Progress    map[string]headway.Progress
	LastUpdated time.Time
}

func main() {

	progress := make(map[string]headway.Progress)
	var mu sync.Mutex

	tmpl, err := template.New("index").Parse(index)
	if err != nil {
		panic(err)
	}

	g := gin.New()

	g.GET("/", func(c *gin.Context) {
		err := tmpl.Execute(c.Writer, data{
			Progress:    progress,
			LastUpdated: time.Now(),
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusOK)
		return
	})

	g.PUT("/", func(c *gin.Context) {
		var p headway.Progress
		if err := c.ShouldBindQuery(&p); err == nil {
			p.LastUpdate = time.Now()
			mu.Lock()
			progress[p.Name] = p
			mu.Unlock()
			c.Status(http.StatusOK)
			return
		} else {
			log.Println(err)
			c.Status(http.StatusBadRequest)
			return
		}
	})

	// Remove old progress bars every 1 hour that haven't been updated for over a day.
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			mu.Lock()
			for k, v := range progress {
				if time.Since(v.LastUpdate) > 1*time.Hour {
					delete(progress, k)
				}
			}
			mu.Unlock()
		}
	}()

	panic(g.Run(":7777"))
}
