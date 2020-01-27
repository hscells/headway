package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hako/durafmt"
	"github.com/hscells/headway"
	"github.com/nlopes/slack"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path"
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
			<li>owner: {{ $p.User }}</li>
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
		<p>Secret: {{ .Secret }}
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

const login = `
<html>
<head>
<title>headway server login</title>
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
<h1>Login</h1>
<p>Login using slack. Once you login you will see your secret you can push logs with.</p>
<a href="https://slack.com/oauth/authorize?client_id=112293530195.922970412213&scope=im:write&user_scope=identify">Click here to login.</a>
</body>
</html>
`

type data struct {
	Progress    []*headway.Progress
	Filters     []string
	LastUpdated string
	Secret      string
}

func randState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func main() {

	f, err := os.OpenFile("config.json", os.O_RDONLY, 0664)
	if err != nil {
		panic(err)
	}

	var config headway.Config
	err = json.NewDecoder(f).Decode(&config)
	if err != nil {
		panic(err)
	}

	bot := slack.New(config.SlackOAuthBotToken)

	g := gin.Default()
	tokens := make(map[string]string)
	secrets := make(map[string]string)
	usernames := make(map[string]string)
	store := cookie.NewStore([]byte(config.Cookie))
	g.Use(sessions.Sessions("slack-archive", store))

	progress := make(map[string]*headway.Progress)
	var logsLastUpdated []*headway.Progress
	var logsProgress []*headway.Progress
	var filters []string
	var mu sync.Mutex

	tmpl, err := template.New("index").Parse(index)
	if err != nil {
		panic(err)
	}

	// Middleware for redirecting for authentication.
	g.Use(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			if strings.Contains(c.Request.URL.Path, "/login") {
				c.Next()
				return
			}
			session := sessions.Default(c)
			token := session.Get("token")
			if token == nil || len(token.(string)) == 0 {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
			if _, ok := tokens[token.(string)]; !ok {
				c.Redirect(http.StatusFound, "/login")
				c.Abort()
				return
			}
		}
		c.Next()
	})

	g.GET("/", func(c *gin.Context) {
		session := sessions.Default(c)
		token := session.Get("token").(string)
		accessToken := tokens[token]

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
				if p.User == f {
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
			Secret:      secrets[accessToken],
		})
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Status(http.StatusOK)
		return
	})

	g.GET("/login", func(c *gin.Context) {
		_, err := c.Writer.WriteString(login)
		if err != nil {
			panic(err)
		}
		c.Status(http.StatusOK)
		return
	})
	g.GET("/logout", func(c *gin.Context) {
		session := sessions.Default(c)
		session.Clear()
	})
	g.GET("/login/oauth", func(c *gin.Context) {
		code := c.Query("code")
		accessToken, _, err := slack.GetOAuthToken(&http.Client{}, config.SlackClientID, config.SlackClientSecret, code, "")
		if err != nil {
			panic(err)
		}
		session := sessions.Default(c)
		token := randState()
		tokens[token] = accessToken
		session.Set("token", token)
		err = session.Save()
		if err != nil {
			panic(err)
		}

		s := slack.New(accessToken)
		ident, err := s.GetUserIdentity()
		if err != nil {
			panic(err)
		}
		err = os.MkdirAll("secrets", 0777)
		if err != nil {
			panic(err)
		}
		secretPath := path.Join("secrets", ident.User.ID)
		usernames[accessToken] = ident.User.Name
		if _, err := os.Stat(secretPath); os.IsNotExist(err) {
			secretsFile, err := os.OpenFile(secretPath, os.O_CREATE|os.O_WRONLY, 0664)
			if err != nil {
				panic(err)
			}
			secret := uuid.New().String()
			_, err = secretsFile.WriteString(secret)
			if err != nil {
				panic(err)
			}
			secrets[accessToken] = secret
		} else {
			secretsFile, err := os.OpenFile(secretPath, os.O_RDONLY, 0664)
			if err != nil {
				panic(err)
			}
			b, err := ioutil.ReadAll(secretsFile)
			if err != nil {
				panic(err)
			}
			secret := string(b)
			secrets[accessToken] = secret
		}
		c.Redirect(http.StatusFound, "/")
		return
	})

	g.PUT("/", func(c *gin.Context) {
		var p *headway.Progress
		if err := c.ShouldBindQuery(&p); err == nil {
			// Do not allow a PUT if the user isn't authorised to.
			foundSecret := false
			var accessToken string
			for token, secret := range secrets {
				if secret == p.Secret {
					accessToken = token
					foundSecret = true
				}
			}
			if !foundSecret {
				c.Status(http.StatusUnauthorized)
				return
			}

			if len(p.Message) > 0 {
				api := slack.New(accessToken)
				ident, err := api.GetUserIdentity()
				if err != nil {
					panic(err)
				}
				ch, ts, err := bot.PostMessage(ident.User.ID, slack.MsgOptionText(fmt.Sprintf("Ahoy <@%s>! New message:\n> %s", ident.User.ID, p.Message), false))
				fmt.Println(ch, ts)
				if err != nil {
					panic(err)
				}
				c.Status(http.StatusOK)
				return
			}

			mu.Lock()
			if _, ok := progress[p.Name]; ok {
				now := time.Now()

				//numItems := p.CurrentProgress - progress[p.Name].CurrentProgress

				p.Started = progress[p.Name].Started
				p.LastUpdate = progress[p.Name].LastUpdate
				p.LastCompleted = now.Sub(p.LastUpdate)
				p.User = progress[p.Name].User
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

				api := slack.New(accessToken)
				ident, err := api.GetUserIdentity()
				if err != nil {
					panic(err)
				}
				p.User = ident.User.Name
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
			mu.Lock()
			logsLastUpdated = make([]*headway.Progress, len(progress))
			logsProgress = make([]*headway.Progress, len(progress))
			i := 0
			for _, v := range progress {
				if _, ok := seenFilters[v.User]; !ok {
					seenFilters[v.User] = struct{}{}
					filters = append(filters, v.User)
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
