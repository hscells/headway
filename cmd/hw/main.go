package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hako/durafmt"
	"github.com/hscells/headway"
	"html/template"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
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
a:clicked {
	color: #aaa;
}
a { 
	color: #eee;
}
ul {
	margin-left: 2em;
}
</style>
</head>
<body>
	{{ range $c, $p := .Progress }}
	<div class="box">
		<div><b class="name">{{ $p.Name }}</b> - {{ $p.CurrentProgress }}/{{ $p.TotalProgress }}</div>
		<div><em>{{ $p.Comment}}</em></div>
		<ul style="font-size: 11px">
			<li>started: {{ $p.Started.Format "Jan 02, 2006 15:04:05 UTC" }}</li>
			<li>last updated: {{ $p.LastUpdate.Format "Jan 02, 2006 15:04:05 UTC" }}</li>
			<li>last item took: {{ $p.LastTook }}</li>
			<li>time elapsed: {{ $p.Elapsed }}</li>
			<li>time remaining: {{ $p.Remaining }}</li>
		</ul>
		<div><progress value="{{ $p.CurrentProgress }}" max="{{ $p.TotalProgress }}"></progress></div>
	</div>
	{{ end }}
	<div>
		<p>sort by <a href="javascript:addSort('progress')">progress</a> | <a href="javascript:addSort('updated')">last updated</a></p>
		<p>filter by:</p>
		<ul>
		{{ range $c, $p := .Filters }}
			<li><a href="javascript:addFilter('{{$p}}')">{{$p}}</a></li>
		{{ end }}
		</ul>
	</div>
	<small>last updated: {{ .LastUpdated }}</small>
<script type="text/javascript">
window.setTimeout(function() {
	location.reload();
}, 5000)
function addFilter(filter) {
	var url = new URL(window.location);
	url.searchParams.set("filter", filter);
	window.location = url.href
}
function addSort(sort) {
	var url = new URL(window.location);
	url.searchParams.set("sort", sort);
	window.location = url.href
}
</script>
</body>
</html>
`

type data struct {
	Progress    []*headway.Progress
	Filters     []string
	LastUpdated string
}

func main() {

	progress := make(map[string]*headway.Progress)
	var logsLastUpdated []*headway.Progress
	var logsProgress []*headway.Progress
	var filters []string
	var mu sync.Mutex

	tmpl, err := template.New("index").Parse(index)
	if err != nil {
		panic(err)
	}

	g := gin.Default()

	g.GET("/", func(c *gin.Context) {
		s := c.Query("sort")
		f := c.Query("filter")

		// Determine which ordering of logs should be presented.
		mu.Lock()
		logs := make([]*headway.Progress, len(logsProgress))
		switch s {
		case "progress":
			copy(logs, logsProgress)
		default:
			copy(logs, logsLastUpdated)
		}
		mu.Unlock()

		// Filter logs to those that contain the filter keyword.
		if len(f) > 0 {
			n := 0
			for _, p := range logs {
				if strings.Contains(p.Name, f) {
					logs[n] = p
					n++
				}
			}
			logs = logs[:n]
		}

		// Execute the template with the chosen (subset of) logs.
		err := tmpl.Execute(c.Writer, data{
			Progress:    logs,
			Filters:     filters,
			LastUpdated: time.Now().Format(time.RFC822),
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
		return
	})

	g.PUT("/", func(c *gin.Context) {
		var p *headway.Progress
		if err := c.ShouldBindQuery(&p); err == nil {
			mu.Lock()
			if _, ok := progress[p.Name]; ok {
				now := time.Now()

				//numItems := p.CurrentProgress - progress[p.Name].CurrentProgress

				p.Started = progress[p.Name].Started
				p.LastUpdate = progress[p.Name].LastUpdate
				p.LastCompleted = now.Sub(p.LastUpdate)
				p.LastUpdate = now

				// https://stackoverflow.com/questions/933242/smart-progress-bar-eta-computation

				decayP := (math.E-1)*(p.CurrentProgress/p.TotalProgress) + 1
				weight := math.Exp(-1 / (p.TotalProgress * decayP))
				slowness := (p.TotalProgress * time.Second.Seconds()) * p.LastCompleted.Seconds()

				p.RateEstimate = progress[p.Name].RateEstimate
				if p.RateEstimate == 0 {
					p.RateEstimate = now.Sub(p.Started).Seconds() * ((p.TotalProgress - p.CurrentProgress) / p.CurrentProgress)
				}

				rateEst := (p.RateEstimate * weight) + (slowness * (1.0 - weight))
				remaining := (1.0 - (p.CurrentProgress / p.TotalProgress)) * rateEst

				p.Remaining = durafmt.Parse(time.Duration(remaining) * time.Second).LimitFirstN(2).String()
				p.RateEstimate = rateEst

			} else {
				p.LastUpdate = time.Now()
				p.Started = p.LastUpdate
			}

			p.LastTook = durafmt.Parse(p.LastCompleted).LimitFirstN(2).String()
			p.Elapsed = durafmt.Parse(time.Now().Sub(p.Started)).LimitFirstN(2).String()

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
			logsLastUpdated = make([]*headway.Progress, 0)
			logsProgress = make([]*headway.Progress, 0)
			mu.Unlock()
		}
	}()

	// Sort progress bars.
	go func() {
		for {
			time.Sleep(1 * time.Second)

			filters = make([]string, 0)
			seenFilters := make(map[string]struct{})
			logsLastUpdated = make([]*headway.Progress, len(progress))
			logsProgress = make([]*headway.Progress, len(progress))

			mu.Lock()
			i := 0
			for k, v := range progress {

				// Find filters.
				split := strings.Split(k, " ")
				for _, kw := range split {
					if _, ok := seenFilters[kw]; ok {
						continue
					}
					if kw[0] == '@' {
						filters = append(filters, kw)
						seenFilters[kw] = struct{}{}
					}
				}

				// Populate the lists.
				logsLastUpdated[i] = v
				logsProgress[i] = v
				i++
			}

			// Sort filters in alphabetical order.
			sort.Strings(filters)

			// Sort by last update.
			sort.Slice(logsLastUpdated, func(i, j int) bool {
				return logsLastUpdated[i].LastUpdate.After(logsLastUpdated[j].LastUpdate)
			})
			// Sort by progress.
			sort.Slice(logsProgress, func(i, j int) bool {
				return (logsProgress[i].CurrentProgress / logsProgress[i].TotalProgress) > (logsProgress[j].CurrentProgress / logsProgress[j].TotalProgress)
			})
			mu.Unlock()
		}
	}()

	panic(g.Run(":7777"))
}
