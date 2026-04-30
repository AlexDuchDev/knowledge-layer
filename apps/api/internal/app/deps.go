package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/knowledgelayer/api/internal/ai/privacy"
	"github.com/knowledgelayer/api/internal/answertrace"
	"github.com/knowledgelayer/api/internal/audit"
	"github.com/knowledgelayer/api/internal/blobstore"
	"github.com/knowledgelayer/api/internal/cache"
	"github.com/knowledgelayer/api/internal/chunks"
	"github.com/knowledgelayer/api/internal/config"
	"github.com/knowledgelayer/api/internal/connectoroauth"
	"github.com/knowledgelayer/api/internal/contentblocks"
	"github.com/knowledgelayer/api/internal/contenthub"
	"github.com/knowledgelayer/api/internal/embeddings"
	"github.com/knowledgelayer/api/internal/extracted_meeting_tasks"
	"github.com/knowledgelayer/api/internal/governance"
	"github.com/knowledgelayer/api/internal/graphrag"
	graphragextract "github.com/knowledgelayer/api/internal/graphrag/extract"
	neo4jgraph "github.com/knowledgelayer/api/internal/graphrag/graphdb/neo4j"
	"github.com/knowledgelayer/api/internal/home"
	"github.com/knowledgelayer/api/internal/identity_access"
	"github.com/knowledgelayer/api/internal/ingestion_connectors"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/asana"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/confluence"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/filesystem"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/fireflies"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/gmail"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/googlecalendar"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/http_url"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/openapi_v3"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/hubspot"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/intercom"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/jira"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/linear"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/mattermost"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/microsoft365"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/notion"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/slack"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/telegram"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/trello"
	"github.com/knowledgelayer/api/internal/ingestion_connectors/adapters/zendesk"
	"github.com/knowledgelayer/api/internal/knowledge_core"
	"github.com/knowledgelayer/api/internal/knowledge_jobs"
	"github.com/knowledgelayer/api/internal/llm"
	auditops "github.com/knowledgelayer/api/internal/modules/audit_ops/app"
	klmcp "github.com/knowledgelayer/api/internal/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/knowledgelayer/api/internal/oauth_proxy"
	"github.com/knowledgelayer/api/internal/onboarding"
	"github.com/knowledgelayer/api/internal/opensearch"
	"github.com/knowledgelayer/api/internal/platform/metrics"
	"github.com/knowledgelayer/api/internal/platform/permissions"
	"github.com/knowledgelayer/api/internal/platform/queue"
	"github.com/knowledgelayer/api/internal/presetcatalog"
	"github.com/knowledgelayer/api/internal/retrieval"
	"github.com/knowledgelayer/api/internal/retrieval_intelligence"
	"github.com/knowledgelayer/api/internal/review"
	"github.com/knowledgelayer/api/internal/role_builder"
	"github.com/knowledgelayer/api/internal/scenario_builder"
	"github.com/knowledgelayer/api/internal/search"
	"github.com/knowledgelayer/api/internal/surfacing"
)

type Deps struct {
	Pool                  *pgxpool.Pool
	Identity              *identity_access.Repo
	Access                *identity_access.AccessEvaluator
	Permissions           *permissions.Resolver
	Entities              *knowledge_core.EntityRepo
	Ingestion             *ingestion_connectors.Service
	AuditOps              *auditops.Service
	Jobs                  *knowledge_jobs.JobService
	Review                *review.Service
	ExtractedMeetingTasks *extracted_meeting_tasks.Service
	Search                *search.Service
	Digest                *knowledge_jobs.DigestRunner
	Retrieval             *retrieval_intelligence.Service
	JobQueue              *queue.Publisher
	BlobStore             blobstore.BlobStore
	PolicyExceptions      *governance.PolicyExceptionService
	OwnerRemediation      *governance.OwnerRemediation
	AnswerFeedback        *governance.AnswerFeedbackService
	ApprovalQueue         *governance.ApprovalQueue
	AnswerTrace           *answertrace.Repo
	PrivacyGateway        *privacy.PrivacyGateway
	Chunks                *chunks.Service
	Embeddings            *embeddings.Service
	Retriever             *retrieval.Service
	Home                  *home.Builder
	ContentHub            *contenthub.Service
	ContentBlocks         *contentblocks.Service
	Follows               *surfacing.FollowRepo
	RoleBuilder           *role_builder.Services
	ScenarioBuilder       *scenario_builder.Services
	PresetCatalog         *presetcatalog.Bundle
	Onboarding            *onboarding.Service
	GraphRAG              *graphrag.GraphStore
	GraphRAGExtract       *graphragextract.Service
	// Cache + invalidator (v0.4.0). Cache is cache.Null() when CACHE_L1_ENABLED=false
	// so handlers can call methods unconditionally; Invalidator is wired into
	// state-changing paths (entity publish, role grant, policy update, feed config patch).
	Cache       cache.Cache
	Invalidator *cache.Invalidator
	// OAuth 2.1 proxy fronting the operator's OIDC issuer (v0.5.0). Nil
	// when OAUTH_PROXY_ENABLED=false; routes 404 in that case.
	OAuthProxy *oauth_proxy.Server
	// MCP server (v0.5.1). Nil when MCP_ENABLED=false. httpserver.Mount
	// passes it (with the OAuth proxy as bearer verifier) to klmcp.Mount.
	MCPServer *mcpserver.MCPServer
}

func NewDeps(pool *pgxpool.Pool, cfg config.Config) (*Deps, error) {
	entities := knowledge_core.NewEntityRepo(pool)
	// L1 cache: enabled only when CACHE_L1_ENABLED=true. Workers and CLI tools
	// may inject cache.Null() instead and the rest of the system stays
	// behaviour-equivalent (every Get is a miss, every Set is a no-op).
	var l1 cache.Cache = cache.Null()
	if cfg.CacheL1Enabled {
		ttl := time.Duration(cfg.CacheL1TTLSeconds) * time.Second
		bc, bErr := cache.NewBigcache(cache.BigcacheOptions{DefaultTTL: ttl, MaxMB: cfg.CacheL1MaxMB})
		if bErr != nil {
			// Cache failure is never fatal — fall back to null cache and
			// log. The instance remains correct, just slower.
			fmt.Printf("app: L1 cache init failed, falling back to null: %v\n", bErr)
		} else {
			l1 = bc
		}
	}
	invalidator := cache.NewInvalidator(l1)
	jobPub, err := queue.NewPublisher(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("app: job queue publisher: %w", err)
	}

	// /metrics enrichment (Phase 2.2.2): wire on-scrape collectors that surface
	// pool / queue state without background goroutines. Job-run and connector
	// histograms are recorded inline by their respective packages.
	metrics.RegisterPoolStats(pool)
	if strings.TrimSpace(cfg.RedisURL) != "" {
		if rOpt, parseErr := redis.ParseURL(cfg.RedisURL); parseErr == nil {
			insp := asynq.NewInspector(asynq.RedisClientOpt{
				Addr:     rOpt.Addr,
				Username: rOpt.Username,
				Password: rOpt.Password,
				DB:       rOpt.DB,
			})
			metrics.RegisterQueueDepth(insp, []string{"default", "connectors", "ingestion", "retrieval", "ai", "governance"})
		}
	}
	chunksSvc := chunks.NewService(pool, jobPub)
	entities.SetOnEntityPersisted(func(ctx context.Context, id uuid.UUID) error {
		if err := chunksSvc.OnEntityPersisted(ctx, id); err != nil {
			return err
		}
		if jobPub != nil && jobPub.Enabled() && cfg.Neo4jURL != "" {
			// GraphRAG extraction runs async; citations-first answers remain anchored to chunks/embeddings.
			_ = jobPub.EnqueueGraphRAGExtractEntity(ctx, id)
		}
		return nil
	})
	rev := review.NewService(pool)
	extractedTasks := extracted_meeting_tasks.NewService(pool)
	digest := knowledge_jobs.NewDigestRunner(pool, entities, rev)
	// EntitySummarizer is built below after PrivacyGateway. JobService
	// construction is deferred until both runners (digest + summarizer)
	// are ready so the orchestrator never holds a nil summarizer with
	// the entity_summarize case enabled.
	osClient := opensearch.NewClient(cfg.OpenSearchURL)
	access := identity_access.NewAccessEvaluator(pool)
	perms := permissions.NewResolver(access)
	searchSvc := search.NewService(pool, osClient, perms)
	aud := audit.NewService(pool)
	at := answertrace.NewRepo(pool)
	embSvc := embeddings.NewService(pool, perms)
	embedClient, _ := llm.NewOpenAIFromEnv()
	retriever := retrieval.NewService(searchSvc, embSvc, embedClient)
	policyRepo := privacy.NewPolicyRepo(pool)
	policySvc := privacy.NewSensitiveDataPolicyService(policyRepo)
	var privVault *privacy.SecureEntityMapStore
	codec, codecErr := privacy.NewVaultCodecFromEnv()
	if codecErr == nil {
		privVault = privacy.NewSecureEntityMapStore(pool, codec, aud)
	} else if cfg.IsProduction() {
		// Fail-closed: ValidateAPI/ValidateWorker should have rejected this combination
		// already, but guard composition root in case wiring is reached without validation
		// (tests, alternative entrypoints).
		return nil, fmt.Errorf("app: privacy vault init failed in APP_ENV=production: %w", codecErr)
	}
	privGW := privacy.NewPrivacyGateway(privacy.GatewayConfig{
		Policy:   policySvc,
		Sanitize: privacy.NewSanitizationService(nil),
		Vault:    privVault,
		Access:   access,
		Audit:    aud,
	})
	summarizer := knowledge_jobs.NewEntitySummarizer(pool, privGW)
	jobs := knowledge_jobs.NewJobService(pool, digest, summarizer, jobPub)

	var graphStore *graphrag.GraphStore
	if cfg.Neo4jURL != "" {
		driver, err := neo4jgraph.Connect(context.Background(), neo4jgraph.Config{
			URL:      cfg.Neo4jURL,
			Username: cfg.Neo4jUser,
			Password: cfg.Neo4jPassword,
		})
		if err != nil {
			return nil, fmt.Errorf("app: neo4j connect: %w", err)
		}
		graphStore = graphrag.NewGraphStore(driver)
	}

	retrieval := retrieval_intelligence.NewService(pool, entities, access, searchSvc, at, retriever, privGW, graphStore)
	adapterReg := ingestion_connectors.NewRegistry(
		telegram.Adapter{},
		slack.Adapter{},
		mattermost.Adapter{},
		http_url.Adapter{},
		filesystem.Adapter{},
		openapi_v3.Adapter{},
		confluence.Adapter{},
		asana.Adapter{},
		linear.Adapter{},
		hubspot.Adapter{},
		zendesk.Adapter{},
		notion.Adapter{},
		jira.Adapter{},
		trello.Adapter{},
		fireflies.Adapter{},
		googlecalendar.Adapter{},
		microsoft365.Adapter{},
		gmail.Adapter{},
		intercom.Adapter{},
	)
	ch := contenthub.NewService(pool)
	follows := surfacing.NewFollowRepo(pool)
	homeB := &home.Builder{
		Access:     access,
		Entities:   entities,
		Review:     rev,
		Follows:    follows,
		ContentHub: ch,
	}
	rb := role_builder.NewServices(pool, access)
	sb := scenario_builder.NewServices(pool)
	presetBundle := presetcatalog.NewBundle(pool, rb, sb, jobs)
	onb := onboarding.NewService(pool, presetBundle)
	bs, err := blobstore.FromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("app: blobstore: %w", err)
	}
	ingestionSvc := ingestion_connectors.NewService(pool, adapterReg)
	gOAuth, mOAuth := connectoroauth.OAuthConfigs(cfg)
	ingestionSvc.ConfigureConnectorOAuth(gOAuth, mOAuth)
	// v0.3.0: when an adapter writes via PersistNormalizedRecord, fire the
	// chunks rebuild synchronously. The 24+ inline INSERTs that did not get
	// refactored fall back to the periodic backfill in connectorworker.
	ingestionSvc.SetOnNormalizedRecordPersisted(func(ctx context.Context, normID uuid.UUID) error {
		return chunksSvc.OnNormalizedRecordPersisted(ctx, normID)
	})

	var graphExtract *graphragextract.Service
	if graphStore != nil {
		extractLLM, err := llm.NewOpenAIFromEnv()
		if err == nil {
			graphExtract = &graphragextract.Service{
				Pool:   pool,
				Chunks: chunksSvc,
				Graph:  graphStore,
				LLM:    extractLLM,
			}
		}
	}

	// OAuth 2.1 proxy (v0.5.0) — opt-in via OAUTH_PROXY_ENABLED. Production
	// hardening rejects bad config at config.ValidateAPI; here we just wire
	// the runtime objects when enabled.
	var oauthSrv *oauth_proxy.Server
	if cfg.OAuthProxyEnabled {
		issuer := cfg.OAuthProxyIssuer
		if issuer == "" {
			issuer = strings.TrimRight(cfg.AppPublicURL, "/")
		}
		callback := cfg.OAuthProxyCallback
		if callback == "" {
			callback = issuer + "/oauth/callback"
		}
		idp, err := oauth_proxy.NewIDPClient(context.Background(), oauth_proxy.IDPConfig{
			Issuer:       cfg.OIDCIssuerURL,
			ClientID:     cfg.OIDCClientID,
			ClientSecret: cfg.OIDCClientSecret,
			RedirectURL:  callback,
		})
		if err != nil {
			if cfg.IsProduction() {
				return nil, fmt.Errorf("app: oauth proxy init: %w", err)
			}
			// Local/staging: log and continue without OAuth so the rest of
			// the API stays runnable for non-MCP flows.
			fmt.Printf("app: oauth proxy disabled (idp init failed): %v\n", err)
		} else {
			oauthSrv = &oauth_proxy.Server{
				IDP:         idp,
				Clients:     oauth_proxy.NewClientRepo(pool),
				Bridge:      oauth_proxy.NewTokenBridge(pool, cfg.OIDCSubColumn),
				SecretKey:   []byte(cfg.OAuthSecretKey),
				Issuer:      issuer,
				RedirectURL: callback,
			}
		}
	}

	// MCP server (v0.5.1). Hardening guarantees MCPEnabled implies
	// OAuthProxyEnabled, but if the proxy init failed (oauthSrv stayed nil)
	// in non-prod, MCP also stays nil — handlers expect a working bearer
	// verifier and there's no point spinning up a dead endpoint.
	var mcpSrv *mcpserver.MCPServer
	if cfg.MCPEnabled && oauthSrv != nil {
		mcpSrv, _ = klmcp.New(klmcp.Deps{
			Access:     access,
			Search:     searchSvc,
			Retrieval:  retrieval,
			Entities:   entities,
			ServerName: "knowledge-layer-mcp",
			ServerVer:  cfg.BuildVersion,
		})
	}

	return &Deps{
		Pool:                  pool,
		Identity:              identity_access.NewRepo(pool),
		Access:                access,
		RoleBuilder:           rb,
		ScenarioBuilder:       sb,
		PresetCatalog:         presetBundle,
		Onboarding:            onb,
		Permissions:           perms,
		Entities:              entities,
		Ingestion:             ingestionSvc,
		AuditOps:              auditops.NewService(aud),
		Jobs:                  jobs,
		Review:                rev,
		ExtractedMeetingTasks: extractedTasks,
		Search:                searchSvc,
		Digest:                digest,
		Retrieval:             retrieval,
		JobQueue:              jobPub,
		BlobStore:             bs,
		PolicyExceptions:      governance.NewPolicyExceptionService(pool),
		OwnerRemediation:      governance.NewOwnerRemediation(pool),
		AnswerFeedback:        governance.NewAnswerFeedbackService(pool),
		ApprovalQueue:         governance.NewApprovalQueue(pool),
		AnswerTrace:           at,
		PrivacyGateway:        privGW,
		Chunks:                chunksSvc,
		Embeddings:            embSvc,
		Retriever:             retriever,
		Home:                  homeB,
		ContentHub:            ch,
		ContentBlocks:         contentblocks.NewService(pool),
		Follows:               follows,
		GraphRAG:              graphStore,
		GraphRAGExtract:       graphExtract,
		Cache:                 l1,
		Invalidator:           invalidator,
		OAuthProxy:            oauthSrv,
		MCPServer:             mcpSrv,
	}, nil
}
