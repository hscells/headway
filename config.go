package headway

type Config struct {
	SlackClientID     string `json:"slack_client_id"`
	SlackClientSecret string `json:"slack_client_secret"`
	Cookie            string `json:"cookie"`
}
