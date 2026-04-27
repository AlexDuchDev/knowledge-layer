package connectoroauth

import (
	"fmt"
	"strings"

	"github.com/knowledgelayer/api/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// OAuthConfigs builds optional oauth2.Config values for Gmail and Microsoft Graph (nil if client id unset).
func OAuthConfigs(cfg config.Config) (gmail *oauth2.Config, microsoft *oauth2.Config) {
	api := cfg.APIPublicOrigin()
	if id := strings.TrimSpace(cfg.GmailOAuthClientID); id != "" {
		gmail = &oauth2.Config{
			ClientID:     id,
			ClientSecret: cfg.GmailOAuthClientSecret,
			RedirectURL:  api + "/integrations/oauth/gmail/callback",
			Scopes:       []string{"https://www.googleapis.com/auth/gmail.readonly"},
			Endpoint:     google.Endpoint,
		}
	}
	if id := strings.TrimSpace(cfg.MicrosoftOAuthClientID); id != "" {
		tenant := strings.TrimSpace(cfg.MicrosoftOAuthTenant)
		if tenant == "" {
			tenant = "common"
		}
		microsoft = &oauth2.Config{
			ClientID:     id,
			ClientSecret: cfg.MicrosoftOAuthClientSecret,
			RedirectURL:  api + "/integrations/oauth/microsoft/callback",
			Scopes: []string{
				"offline_access",
				"openid",
				"profile",
				"https://graph.microsoft.com/User.Read",
				"https://graph.microsoft.com/Mail.Read",
				"https://graph.microsoft.com/ChannelMessage.Read.All",
				"https://graph.microsoft.com/Files.Read.All",
				"https://graph.microsoft.com/Calendars.Read",
				"https://graph.microsoft.com/Team.ReadBasic.All",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant),
				TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
			},
		}
	}
	return gmail, microsoft
}
