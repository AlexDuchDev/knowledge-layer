package httpserver

import (
	"encoding/json"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/knowledgelayer/api/internal/app"
	"github.com/knowledgelayer/api/internal/httpcontext"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
)

// registerConnectorIntegrationRoutes wires onboarding discovery endpoints (minimal admin setup).
func registerConnectorIntegrationRoutes(f *fiber.App, d *app.Deps) {
	f.Post("/integrations/jira/list-projects", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID        uuid.UUID `json:"domain_id"`
			JiraSiteBaseURL string    `json:"jira_site_base_url"`
			JiraEmail       string    `json:"jira_email"`
			JiraAPIToken    string    `json:"jira_api_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]string{
			"jira_site_base_url": strings.TrimSpace(body.JiraSiteBaseURL),
			"jira_email":         strings.TrimSpace(body.JiraEmail),
			"jira_api_token":     strings.TrimSpace(body.JiraAPIToken),
		})
		cfg, err := ingestion_connectors.ParseJiraFeedConfig(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		list, err := d.Ingestion.ListJiraProjects(c.Context(), cfg)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/trello/list-boards", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID     uuid.UUID `json:"domain_id"`
			TrelloAPIKey string    `json:"trello_api_key"`
			TrelloToken  string    `json:"trello_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]string{
			"trello_api_key": strings.TrimSpace(body.TrelloAPIKey),
			"trello_token":   strings.TrimSpace(body.TrelloToken),
		})
		cfg, err := ingestion_connectors.ParseTrelloFeedConfig(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		list, err := d.Ingestion.ListTrelloBoards(c.Context(), cfg)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/asana/list-projects", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID                 uuid.UUID `json:"domain_id"`
			AsanaPersonalAccessToken string    `json:"asana_personal_access_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListAsanaProjects(c.Context(), body.AsanaPersonalAccessToken)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/linear/list-teams", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID     uuid.UUID `json:"domain_id"`
			LinearAPIKey string    `json:"linear_api_key"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListLinearTeams(c.Context(), body.LinearAPIKey)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/slack/list-channels", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID uuid.UUID `json:"domain_id"`
			BotToken string    `json:"bot_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListSlackChannels(c.Context(), body.BotToken)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/confluence/list-spaces", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID          uuid.UUID `json:"domain_id"`
			ConfluenceBaseURL string    `json:"confluence_base_url"`
			ConfluenceAuth    string    `json:"confluence_auth"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListConfluenceSpaces(c.Context(), body.ConfluenceBaseURL, body.ConfluenceAuth)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/notion/search", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID               uuid.UUID `json:"domain_id"`
			NotionIntegrationToken string    `json:"notion_integration_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListNotionSearchResults(c.Context(), body.NotionIntegrationToken)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/google-calendar/list-calendars", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID       uuid.UUID       `json:"domain_id"`
			ServiceAccount json.RawMessage `json:"service_account"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		if len(body.ServiceAccount) == 0 {
			return fiber.NewError(fiber.StatusBadRequest, "service_account required")
		}
		wrapped, mErr := json.Marshal(map[string]json.RawMessage{"service_account": body.ServiceAccount})
		if mErr != nil {
			return fiber.NewError(fiber.StatusBadRequest, mErr.Error())
		}
		sa, err := ingestion_connectors.ServiceAccountJSONFromConnectorConfig(wrapped)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		list, err := d.Ingestion.ListGoogleCalendars(c.Context(), sa)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/google-drive/list-folders", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID       uuid.UUID       `json:"domain_id"`
			ServiceAccount json.RawMessage `json:"service_account"`
			ParentFolderID string          `json:"parent_folder_id"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]json.RawMessage{"service_account": body.ServiceAccount})
		sa, err := ingestion_connectors.ServiceAccountJSONFromConnectorConfig(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		list, err := d.Ingestion.ListGoogleDriveFolders(c.Context(), sa, body.ParentFolderID)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/mattermost/list-channels", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID          uuid.UUID `json:"domain_id"`
			MattermostBaseURL string    `json:"mattermost_base_url"`
			MattermostToken   string    `json:"mattermost_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		raw, _ := json.Marshal(map[string]string{
			"mattermost_base_url": strings.TrimSpace(body.MattermostBaseURL),
			"mattermost_token":    strings.TrimSpace(body.MattermostToken),
		})
		cfg, err := ingestion_connectors.ParseMattermostFeedConfig(raw)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		list, err := d.Ingestion.ListMattermostChannels(c.Context(), cfg)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})

	f.Post("/integrations/zendesk/list-views", func(c *fiber.Ctx) error {
		principal, err := httpcontext.RequirePrincipal(c)
		if err != nil {
			return err
		}
		var body struct {
			DomainID         uuid.UUID `json:"domain_id"`
			ZendeskSubdomain string    `json:"zendesk_subdomain"`
			ZendeskEmail     string    `json:"zendesk_email"`
			ZendeskAPIToken  string    `json:"zendesk_api_token"`
		}
		if err := c.BodyParser(&body); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid json")
		}
		if _, err := resolveIntegrationOnboardingDomain(c, d, principal, body.DomainID); err != nil {
			return err
		}
		list, err := d.Ingestion.ListZendeskViews(c.Context(), body.ZendeskSubdomain, body.ZendeskEmail, body.ZendeskAPIToken)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(list)
	})
}
