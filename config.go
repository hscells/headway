package headway

type Config struct {
	SlackClientID         string `json:"slack_client_id"`
	SlackClientSecret     string `json:"slack_client_secret"`
	SlackOAuthRedirectURI string `json:"slack_oauth_redirect_uri"`
	SlackOAuthBotToken    string `json:"slack_oauth_bot_token"`
	Cookie                string `json:"cookie"`
}
