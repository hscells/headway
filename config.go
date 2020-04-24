package headway

import (
	"io/ioutil"
	"os"
	"path"
)

type Config struct {
	SlackClientID         string `json:"slack_client_id"`
	SlackClientSecret     string `json:"slack_client_secret"`
	SlackOAuthRedirectURI string `json:"slack_oauth_redirect_uri"`
	SlackOAuthBotToken    string `json:"slack_oauth_bot_token"`
	Cookie                string `json:"cookie"`
}

func LoadSecrets(dir string) (map[string]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	secrets := make(map[string]string)

	for _, file := range files {
		if !file.IsDir() {
			p := path.Join(dir, file.Name())

			f, err := os.Open(p)
			if err != nil {
				return nil, err
			}

			b, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}

			secrets[file.Name()] = string(b)

			err = f.Close()
			if err != nil {
				return nil, err
			}
		}
	}
	return secrets, nil
}
