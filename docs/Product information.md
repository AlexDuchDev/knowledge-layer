Organizational Memory / Company Brain
1. Обзор проекта
1.1 Суть
Проект представляет собой платформу organizational memory / company brain для компаний, которым нужно не просто хранить документы, а превращать разрозненные знания, обсуждения, решения и артефакты в управляемую систему памяти и knowledge operations.
Система должна:
подключать разные источники знаний;
забирать из них данные по управляемым правилам;
превращать входящие сигналы в структурированные knowledge objects;
обеспечивать поиск, retrieval и AI-assisted synthesis;
запускать knowledge jobs по расписанию, по событию или вручную;
строго соблюдать разграничение доступов;
поддерживать governance, review, approval и lifecycle контента.
1.2 Проблема
В большинстве компаний знания живут в:
Slack;
email;
Telegram;
meeting notes и transcript tools;
Jira/Trello/boards;
Notion/Confluence/Google Docs;
головах отдельных людей.
Из-за этого возникают типовые проблемы:
теряется контекст решений;
знания зависят от конкретных людей;
новые сотрудники долго входят в контекст;
одни и те же вопросы повторяются;
процессы и договоренности расползаются;
невозможно быстро собрать качественную сводку по нужному процессу;
AI-слой без governance становится опасным.
1.3 Цель проекта
Построить взрослую платформу organizational memory и knowledge operations, которая:
хранит не только документы, но и причины решений, связи и историю;
умеет читать разные источники как controlled source feeds;
умеет выполнять process-bound knowledge jobs;
обеспечивает permission-aware retrieval;
работает как управляемая knowledge infrastructure.

2. Продуктовое определение
2.1 Что это не такое
Система не является:
обычной knowledge base;
красивым workspace в Notion;
простым AI-chat над корпоративными данными;
файловым архивом.
2.2 Что это такое
Система является:
organizational memory platform;
company brain infrastructure;
knowledge operations engine;
governed retrieval and synthesis layer;
controlled connector-based knowledge ingestion system.
2.3 Основная формула
Systems of record -> Ingestion -> Normalization -> Knowledge Core -> Search/Retrieval -> AI Synthesis -> Governance/Operations

3. Ключевые требования
3.1 Функциональные требования
Система должна:
подключать разные внешние источники знаний;
поддерживать Telegram ingestion mode в v1;
поддерживать knowledge jobs;
поддерживать ручной, scheduled, event-driven и window-based запуск jobs;
позволять настраивать output policy для результатов;
обеспечивать разграничение доступов по доменам, типам объектов, объектам и действиям;
позволять AI работать только в разрешенном scope;
хранить version history и audit trail;
поддерживать review/approval/lifecycle для материалов;
поддерживать source-of-truth модель.
3.2 Нефункциональные требования
Система должна быть:
управляемой;
расширяемой;
безопасной;
traceable;
пригодной для enterprise-leaning сценариев;
пригодной для поэтапного расширения без переписывания foundation.

4. Версия 1: scope
4.1 Что входит в первую версию
Источники
Telegram ingestion mode;
Slack;
Email;
Fireflies;
Granola;
Jira;
Trello;
Notion;
Google Drive / Docs.
Основные knowledge domains
Decisions;
Projects;
Processes / SOP;
Policies;
Meetings;
Insights.
Основные системные возможности
connector framework;
source feed configuration;
knowledge jobs engine;
access model;
entity model;
versioning;
audit trail;
hybrid retrieval;
AI with scoped retrieval and citations;
admin/governance UI.
4.2 Ограничение по Telegram для v1
В первой версии Telegram используется только как controlled ingestion source.
Система читает только те Telegram-чаты, которые:
явно подключены;
имеют owner;
имеют заданный knowledge scope;
имеют domain;
имеют sensitivity policy;
имеют разрешенные knowledge jobs.
Telegram не используется во v1 как универсальный output channel и не используется как unrestricted bot interface.

5. Архитектурные принципы
5.1 Принцип 1. Knowledge objects важнее файлов
Система должна строиться вокруг canonical entities, а не вокруг папок и документов.
5.2 Принцип 2. Raw context не теряется
Исходные сигналы и сырой контекст могут храниться и использоваться как source context, но source of truth определяется отдельно.
5.3 Принцип 3. AI не является authority layer
AI работает поверх retrieval и governance, но не определяет правду и не обходит политики доступа.
5.4 Принцип 4. Access control проверяется до retrieval
Пользователь и AI получают только тот контекст, который разрешен до генерации ответа.
5.5 Принцип 5. Материалы наследуют правила автоматически
Во v1 правила доступа в основном задаются на уровне доменов, источников, процессов и ролей. Материалы наследуют их автоматически. Ручные overrides используются для исключений.
5.6 Принцип 6. Взрослость системы определяется контролем, а не количеством функций
Взрослая версия означает строгую модель доступа, governance, lifecycle, audit и connectors, а не бесконечный scope.

6. Доменная модель
6.1 Основные bounded domains
Identity & Access
Knowledge Core
Workflow & Governance
Ingestion & Connectors
Retrieval & Intelligence
Platform Operations
Knowledge Operations
6.2 Identity & Access
Отвечает за:
users;
teams;
roles;
domains;
sensitivity levels;
access policies;
overrides;
action permissions.
6.3 Knowledge Core
Отвечает за:
entities;
entity types;
metadata;
links;
versions;
provenance;
canonical state.
6.4 Workflow & Governance
Отвечает за:
lifecycle states;
review tasks;
approvals;
freshness;
owners;
policy enforcement.
6.5 Ingestion & Connectors
Отвечает за:
connectors;
source feeds;
sync modes;
parsing;
normalization;
deduplication;
ingestion jobs.
6.6 Retrieval & Intelligence
Отвечает за:
keyword search;
semantic retrieval;
hybrid retrieval;
chunking;
embeddings;
reranking;
Q&A;
summarization;
extraction.
6.7 Platform Operations
Отвечает за:
audit;
observability;
notifications;
queue jobs;
metrics;
system alerts;
execution traces.
6.8 Knowledge Operations
Отвечает за:
knowledge jobs;
job templates;
triggers;
source scopes;
output routes;
execution runs;
result artifacts.

7. Источники знаний и коннекторы
7.1 Категории источников
Communication
Slack
Email
Telegram
Meeting / conversation tools
Fireflies
Granola
manual transcript upload
Project/task systems
Jira
Trello
Documentation systems
Notion
Google Drive / Docs
7.2 Connector framework
Каждый коннектор должен поддерживать:
auth model;
source mapping;
sync mode;
event support;
content parsing;
metadata extraction;
deduplication;
scheduling policy;
retry/error policy.
7.3 Режимы подключения
Full import
Incremental sync
Event-driven ingestion
7.4 Telegram ingestion mode
Для каждого Telegram source feed задаются:
owner;
domain;
knowledge scope;
sensitivity level;
allowed jobs/processes;
ingestion mode.
Пример:
source: telegram://finance_ops_internal
owner: Head of Finance
domain: Finance
scope: operational finance discussions
sensitivity: domain_restricted
jobs: weekly summary, issue extraction, decision extraction

8. Knowledge Core
8.1 Canonical entity types
На первом этапе используются следующие canonical types:
Decision
Project
Initiative
SOP
Process
Policy
Meeting
Incident
Experiment
Insight
Customer Insight
Role Handbook
Team Handbook
Template
Reference Document
8.2 Базовые поля сущности
Каждая сущность должна иметь как минимум:
id
type
title
body/content
owner
domain
sensitivity_level
source
canonical_status
approval_status
freshness_status
created_at
updated_at
review_due_at
process_binding
access_policy_id
8.3 Связи
Сущности должны поддерживать explicit relations, например:
related_to
derived_from
decides
affects
created_in_meeting
owned_by
linked_to_project
learned_from_incident
8.4 Versioning
Каждая важная сущность должна поддерживать version history.
8.5 Provenance
У каждого объекта должна сохраняться информация:
откуда он пришел;
каким job/process был создан;
каким пользователем подтвержден/опубликован;
на каких raw inputs основан.

9. Модель доступов
9.1 Общий принцип
Доступы настраиваются на трех уровнях:
пользователи;
материалы;
процессы / jobs.
Во v1 большинство правил задается вручную на уровне roles, teams, domains, source feeds и jobs. Материалы наследуют правила автоматически.
9.2 Пользовательские настройки
Для каждого пользователя вручную настраиваются:
team membership;
roles;
domain access;
action permissions;
optional special overrides.
Пример: Пользователь: CMO
team: Marketing
role: Domain Owner
Marketing: edit/approve
Cross-functional: view
Finance: deny
Legal: deny
Engineering: deny
9.3 Настройки материалов
Большинство материалов наследуют правила автоматически по следующим источникам:
source feed policy;
entity type policy;
output policy job-а;
domain default policy.
Ручные overrides применяются только к исключениям.
9.4 Настройки процессов / jobs
Для каждого knowledge job задаются:
source scopes;
allowed operators;
output domain;
output sensitivity;
publication mode;
review requirement;
sanitization rules.
9.5 Access layers
Система должна поддерживать:
domain-based access;
entity-type access;
object-level access;
action-level permissions;
inheritance and exception rules.
9.6 Sensitivity levels
Рекомендуемые уровни:
public_internal
team_restricted
domain_restricted
leadership_restricted
strictly_confidential
9.7 Access resolution flow
При каждом запросе система должна:
проверить identity;
проверить global deny rules;
проверить object-level ACL;
проверить domain access;
проверить entity-type rules;
проверить action permission;
проверить sensitivity level;
вернуть allow/deny.
9.8 AI и доступы
AI работает только на permission-scoped retrieval.
Алгоритм:
определить пользователя;
определить его allowed scope;
выполнить retrieval только по разрешенным объектам;
передать в LLM только допустимый контекст;
сохранить answer trace;
вернуть ответ с citations.

10. Knowledge Operations Engine
10.1 Назначение
Knowledge Operations Engine — это слой, который позволяет выполнять операции над знаниями по процессам, триггерам и расписанию.
10.2 Типы knowledge jobs
Summarization jobs
Extraction jobs
Consolidation jobs
Monitoring jobs
Transformation jobs
Publishing jobs
10.3 Режимы запуска
Manual
Scheduled
Event-driven
Window-based
Conditional
10.4 Примеры
Weekly daily digest
Источник:
Telegram daily chat
Slack daily channel
Granola/Fireflies transcripts
Операция:
summarize progress, blockers, risks, decisions
Результат:
weekly digest entity
optional review task
Planning summary
Источник:
planning transcript
Jira board
planning notes
Операция:
extract commitments, risks, decisions, open questions
Результат:
planning summary entity
linked Decision objects
review task for owner
10.5 Важные свойства job-а
Каждый job должен иметь:
purpose;
source scope;
trigger;
config;
output route;
access policy;
review policy;
execution logs.

11. Workflow & Governance
11.1 Lifecycle states
Примеры:
SOP / Policy
Draft
In Review
Approved
Active
Stale
Archived
Decision
Proposed
Confirmed
Superseded
Archived
Project
Draft
Active
Blocked
Completed
Archived
11.2 Review
Для важных сущностей должны задаваться:
owner;
review_due_at;
reviewer;
freshness rule.
11.3 Approval
Для selected entity types должны быть approval flows.
11.4 Freshness
Контент должен иметь freshness tracking и stale detection.

12. Search, Retrieval и AI
12.1 Виды retrieval
exact / keyword search;
filtered retrieval;
semantic retrieval;
hybrid retrieval;
relation-aware retrieval;
permission-aware retrieval;
freshness-aware retrieval.
12.2 AI функции
summarization;
entity extraction;
decision extraction;
action item extraction;
suggested links;
duplicate detection;
stale detection;
scoped Q&A.
12.3 Правила AI
AI не является source of truth;
AI не видит недоступные объекты;
AI должен возвращать supporting entities / citations;
AI ответы traceable;
AI не публикует автоматом критичные материалы без review.

13. Техническая архитектура
13.1 Технологический стек
Frontend
Next.js
React
TypeScript
Tailwind
TanStack Query
Backend
Go
modular monolith
background workers
webhook processors
Data
PostgreSQL
pgvector
Redis
S3-compatible storage
OpenSearch
13.2 Почему modular monolith
На старте и ранней взрослой версии modular monolith дает:
более низкую сложность;
более быстрый запуск;
контроль над bounded domains;
возможность позже выделять отдельные сервисы.
13.3 Верхнеуровневые технические компоненты
frontend app;
backend API;
workers;
connector processors;
PostgreSQL;
Redis;
S3 storage;
OpenSearch;
observability stack.

14. Высокоуровневая модель данных
14.1 Основные сущности
users
teams
roles
domains
access_policies
entities
entity_versions
entity_links
entity_permissions
source_feeds
connectors
raw_artifacts
chunks
embeddings
review_tasks
approval_flows
knowledge_jobs
job_runs
job_outputs
audit_events
notifications
14.2 Таблицы доступа
Нужны таблицы и поля для:
user-role bindings;
domain grants;
action permissions;
entity ACL;
policy overrides.
14.3 Таблицы knowledge operations
Нужны таблицы и поля для:
job definitions;
triggers;
source scopes;
execution runs;
outputs;
routing and policy configuration.

15. Административные интерфейсы
15.1 Users & Access
Функции:
назначение команд;
назначение ролей;
domain permissions;
special overrides.
15.2 Source Feeds
Функции:
подключение источника;
назначение owner;
назначение domain;
sensitivity;
ingestion mode;
allowed jobs;
sync controls.
15.3 Knowledge Jobs
Функции:
создание job-а;
настройка trigger;
настройка source scope;
настройка output policy;
выбор publication mode;
review requirements.
15.4 Governance Center
Функции:
review queues;
approval queues;
stale content;
policy exceptions;
audit trace.

16. Как это работает на практике
16.1 Пример Telegram source feed
Шаги:
Админ подключает Telegram chat.
Задает owner.
Задает domain.
Задает sensitivity.
Задает allowed jobs.
Выбирает ingestion mode.
16.2 Пример job
Job: weekly summary from Telegram finance chat
source: telegram://finance_ops_internal
operators: CFO, Finance Ops Lead
output domain: Finance
output sensitivity: domain_restricted
publication: review required
16.3 Пример наследования прав
Если Telegram source feed имеет domain Finance и sensitivity domain_restricted, то по умолчанию summary materials, созданные из него, получают те же ограничения, если job configuration не задает другой controlled output policy.

17. Roadmap реализации
Этап 1. Foundation
users, teams, roles, domains
access model
entities CRUD
source feeds
Telegram ingestion mode
audit log
base search
Этап 2. Knowledge operations
knowledge jobs
triggers
job runs
output policies
review requirements
Этап 3. Retrieval & AI
chunking
embeddings
hybrid retrieval
permission-aware AI Q&A
summarization/extraction
Этап 4. Governance hardening
approvals
lifecycle states
freshness rules
review center
policy exceptions
Этап 5. Connector expansion
Slack
Email
Fireflies
Granola
Jira
Trello
Notion
Google Drive

18. Риски проекта
18.1 Основные риски
переусложнение онтологии;
отсутствие четких source-of-truth правил;
неуправляемый AI scope;
попытка настраивать доступы вручную на каждый объект;
отсутствие owner-ов у feed-ов и jobs;
попытка сразу покрыть все сценарии;
слабая дисциплина governance.
18.2 Принципы снижения рисков
использовать ограниченный canonical entity model;
запускать взрослую foundation, но фазировать rollout;
задавать default inheritance policies;
использовать ручные overrides только для исключений;
не позволять AI обходить permission engine;
фиксировать policy на уровне users + sources + jobs.

19. Позиционирование проекта
Проект можно описывать как:
Organizational Memory & Knowledge Operations Platform
или
Company Brain Infrastructure with Controlled Ingestion, Governance and AI Retrieval
Ключевое отличие проекта:
он объединяет memory, operations, governance и AI;
он работает с реальными процессами, а не только с хранением;
он проектируется под controlled enterprise-leaning usage.

20. Следующие документы
После этого базового master-document логично подготовить отдельные документы:
Product Requirements Document (PRD)
Architecture Specification
Access Model Specification
Connector Framework Specification
Knowledge Jobs Specification
Data Model / ERD
API Contract v1
Admin Panel Specification
Implementation Plan by Epics
Security & Governance Rules

Architecture Specification
1. Purpose
Этот документ определяет техническую архитектуру платформы Organizational Memory / Company Brain для передачи в разработку.
Система проектируется как production-grade foundation с controlled ingestion, governed knowledge core, permission-aware retrieval, AI orchestration и knowledge operations engine.

2. Architectural Goals
Система должна:
поддерживать multiple source connectors;
поддерживать Telegram ingestion mode в первой версии;
обеспечивать controlled source feed mapping;
хранить canonical knowledge objects;
поддерживать versioning, provenance и audit;
обеспечивать strict access control;
поддерживать process-bound knowledge jobs;
поддерживать retrieval и AI only within allowed scope;
быть расширяемой без переписывания foundation.

3. High-Level Architecture
Платформа состоит из следующих bounded domains:
Identity & Access Domain
Knowledge Core Domain
Workflow & Governance Domain
Ingestion & Connectors Domain
Retrieval & Intelligence Domain
Platform Operations Domain
Knowledge Operations Domain
High-level flow
External Systems -> Connectors -> Raw Artifacts -> Normalization -> Canonical Entities -> Search / Retrieval -> AI / Jobs -> Published Outputs / Governance

4. Technology Stack
Frontend
Next.js
React
TypeScript
Tailwind CSS
TanStack Query
Backend
Go
modular monolith architecture
background workers
webhook processors
Data & Storage
PostgreSQL as system of record
pgvector for embeddings
OpenSearch for advanced retrieval and filtering
Redis for queues, caching and transient execution state
S3-compatible storage for raw artifacts and attachments
AI
OpenAI API
prompt orchestration in backend only
no direct model access to unrestricted corpus
Infrastructure
Docker
separate services for web app, API, workers, connector processors
secrets manager
centralized logs
metrics and tracing

5. Architectural Style
5.1 Chosen style
System uses modular monolith backend with clear internal domain boundaries.
5.2 Why modular monolith
Reasons:
lower operational complexity;
faster implementation;
easier consistency across access, audit and entity lifecycle;
easier delivery by small team;
supports later extraction of ingestion/search workers if needed.
5.3 Not chosen initially
no early microservices split;
no graph database in v1 foundation;
no custom rich collaborative editor in first implementation.

6. Domain Breakdown
6.1 Identity & Access Domain
Responsibilities:
users
teams
roles
domains
access policies
sensitivity levels
object overrides
action permissions
Core objects:
User
Team
Role
Domain
AccessPolicy
EntityACL
UserDomainGrant
6.2 Knowledge Core Domain
Responsibilities:
canonical entities
entity types
metadata
relations
versions
provenance
canonical state
Core objects:
Entity
EntityType
EntityVersion
EntityLink
EntityMetadata
SourceReference
6.3 Workflow & Governance Domain
Responsibilities:
lifecycle states
approvals
review tasks
freshness
ownership
publication status
Core objects:
ReviewTask
ApprovalFlow
LifecycleState
FreshnessRule
PublicationPolicy
6.4 Ingestion & Connectors Domain
Responsibilities:
connectors
source feeds
sync definitions
ingestion jobs
parsers
normalization
deduplication
Core objects:
Connector
SourceFeed
SyncJob
RawArtifact
ImportSession
ParserResult
6.5 Retrieval & Intelligence Domain
Responsibilities:
keyword retrieval
semantic retrieval
hybrid retrieval
reranking
embeddings
answer traces
extraction services
summarization services
Core objects:
Chunk
Embedding
RetrievalQuery
RetrievalResult
AnswerTrace
ExtractionResult
6.6 Platform Operations Domain
Responsibilities:
audit
observability
job run logs
notifications
system alerts
metrics
retries
Core objects:
AuditEvent
JobRun
Notification
SystemMetric
Alert
6.7 Knowledge Operations Domain
Responsibilities:
knowledge jobs
templates
triggers
source scopes
output routing
execution orchestration
Core objects:
KnowledgeJob
JobTemplate
JobTrigger
SourceScope
JobOutputPolicy
JobExecutionRun
ResultArtifact

7. Core Architectural Rules
7.1 Entities are first-class
System must be designed around typed canonical entities, not folders.
7.2 Source of truth is explicit
Each domain/object type must define source-of-truth rules.
7.3 AI is downstream from permissions
AI receives only pre-filtered allowed context.
7.4 Materials inherit policies
Most materials inherit access and governance rules from source feeds, domains and jobs.
7.5 Overrides are exceptional
Object-level custom ACL must be rare and governed.
7.6 Raw context is preserved
Raw artifacts may be retained as source context but remain distinct from canonical approved knowledge objects.

8. Ingestion Architecture
8.1 Supported source categories
Communication:
Telegram
Slack
Email
Meetings:
Fireflies
Granola
manual transcript uploads
Work management:
Jira
Trello
Documentation:
Notion
Google Drive / Docs
8.2 Connector model
Each connector must support:
auth configuration
source discovery
source feed mapping
sync mode
error policy
retry policy
raw artifact persistence
metadata extraction
normalization
8.3 Sync modes
full import
incremental sync
event-driven sync
manual sync
8.4 Telegram v1 architecture
Telegram is supported only as controlled ingestion source.
Each Telegram source feed must define:
feed owner
mapped domain
knowledge scope
sensitivity level
allowed jobs
sync mode
Telegram messages are ingested only from explicitly connected chats.
8.5 Ingestion pipeline
Receive or fetch source artifact
Persist raw artifact
Parse content and metadata
Normalize content
Resolve source feed policy
Classify target entity type or draft type
Create/update entity or draft artifact
Create chunks and embeddings
Create provenance links
Emit audit and job events

9. Knowledge Core Architecture
9.1 Canonical entity types for first implementation
Decision
Project
Initiative
SOP
Process
Policy
Meeting
Incident
Experiment
Insight
CustomerInsight
TeamHandbook
RoleHandbook
Template
ReferenceDocument
9.2 Required entity attributes
id
type
title
body
owner_user_id
owner_team_id
domain_id
sensitivity_level
source_feed_id
canonical_status
approval_status
freshness_status
process_binding
access_policy_id
created_at
updated_at
review_due_at
archived_at
9.3 Links
Entities must support typed relations such as:
related_to
derived_from
created_from_source
created_in_meeting
linked_to_project
affects
supersedes
learned_from_incident
9.4 Versioning
All material entity updates must create entity versions.
9.5 Provenance
Every entity must retain references to:
source feed
raw artifact(s)
producing job if any
publishing actor if any

10. Retrieval Architecture
10.1 Retrieval modes
keyword search
filtered retrieval
semantic retrieval
hybrid retrieval
relation-aware retrieval
permission-aware retrieval
freshness-aware retrieval
10.2 Default search mode
Default mode must be hybrid retrieval with access checks and ranking.
10.3 Ranking signals
keyword relevance
semantic similarity
domain/type relevance
freshness
approval status
source-of-truth priority
relation proximity
10.4 AI retrieval flow
Resolve user identity
Resolve allowed scope
Retrieve only allowed objects/chunks
Rerank
Build answer context
Generate answer with citations
Persist answer trace

11. Governance Architecture
11.1 Lifecycle states
Examples:
SOP/Policy: Draft -> InReview -> Approved -> Active -> Stale -> Archived
Decision: Proposed -> Confirmed -> Superseded -> Archived
Project: Draft -> Active -> Blocked -> Completed -> Archived
11.2 Review rules
Important entities must support:
owner
reviewer
due date
freshness policy
11.3 Approval rules
Selected entity types require approval before publication.
11.4 Publication rules
Jobs may publish:
draft only
reviewed only
auto-published if low risk and policy allows

12. Deployment Architecture
Deployable components:
web frontend
API/backend
worker process
connector processor
PostgreSQL
Redis
S3-compatible storage
OpenSearch
observability stack
Recommended deployment separation:
frontend service
backend service
background workers
connector/webhook workers

13. Non-Goals for first implementation
multi-tenant public SaaS architecture
graph DB
custom collaborative editor
no-code workflow builder
unrestricted autonomous agents
universal ontology builder

14. Architecture Decisions Summary
Go modular monolith for backend
Postgres as primary system of record
OpenSearch + pgvector for retrieval
Telegram supported only as controlled ingestion source in v1
jobs and access are first-class architecture elements
AI always operates after permission filtering

Access Model Specification
1. Purpose
Документ определяет, как настраиваются и применяются доступы для пользователей, материалов и процессов/jobs.
Главная цель: обеспечить controlled access to knowledge across domains, types, objects and actions, including AI retrieval.

2. Design Principles
Most access rules are configured manually at policy level, not per object.
Materials inherit rules automatically whenever possible.
Object-level overrides are exceptions.
Jobs/processes have independent access rules.
AI must obey exactly the same permission model.

3. Access Subjects
Subjects that can receive permissions:
User
Team
Role
DomainOwner group
Admin group

4. Access Targets
Targets that can be protected:
Domain
EntityType
Entity
SourceFeed
KnowledgeJob
JobOutput
Admin feature

5. Permission Actions
Supported actions:
view
create
edit
approve
archive
export
manage_permissions
manage_sources
manage_jobs
manage_policies

6. Access Layers
6.1 Domain-based access
Each user is granted access to one or more domains.
Examples of domains:
Finance
Legal
Marketing
Product
Engineering
Operations
Leadership
CrossFunctional
HR
6.2 Entity-type access
Within allowed domains, permissions can vary by entity type.
Example: A user may view Finance/Policy but not Finance/PayrollDocument.
6.3 Object-level access
Specific objects may define explicit ACL overrides.
6.4 Action-level access
A user may view an object but not edit or approve it.
6.5 Process/job-level access
Users may have permission to view results of a job but not run, edit or publish that job.

7. Sensitivity Levels
Supported values:
public_internal
team_restricted
domain_restricted
leadership_restricted
strictly_confidential
Sensitivity affects visibility resolution in addition to grants.

8. Access Configuration Model
8.1 User access configuration
Each user must have:
primary team
one or more roles
domain grants
action grants
optional denies
optional overrides
8.2 Material access configuration
Most materials must inherit:
domain from source feed or creator context
default access policy from domain/entity type
output access policy from producing job
sensitivity from source or publication policy
8.3 Process/job configuration
Each job must define:
who may run it
who may edit it
who may review/publish outputs
what source feeds it may read
what domain the outputs belong to
what sensitivity the outputs receive

9. Default Inheritance Rules
9.1 From source feed
If material comes from a source feed, it inherits:
domain
source owner
sensitivity default
access policy default
9.2 From entity type
If entity type defines stricter rules, they override domain defaults.
9.3 From knowledge job
If material is produced by a knowledge job, job output policy may define:
output domain
output sensitivity
review requirement
publication target
9.4 Manual overrides
Manual overrides may be applied only by authorized roles.

10. Resolution Order
For every request, backend resolves access in this order:
identity authenticated
global deny rules
object-level deny/allow overrides
domain access
entity-type access
action permission
sensitivity compatibility
final allow/deny
This logic must execute on backend for every read/write/retrieval request.

11. AI Access Rules
AI must not query unrestricted corpus.
For every AI request:
resolve user identity
resolve allowed domains/types/objects
resolve sensitivity constraints
retrieve only allowed chunks
send only allowed context to LLM
store answer trace
AI outputs must include citations to supporting entities.

12. Admin Configuration Flows
12.1 Users & Access screen
Admin/configurator sets:
team
roles
domain grants
action grants
optional denies
exceptional object overrides
12.2 Source Feed screen
Admin/configurator sets:
owner
domain
knowledge scope
sensitivity
allowed jobs
sync mode
12.3 Knowledge Job screen
Admin/configurator sets:
source scope
allowed operators
output domain
output sensitivity
publication mode
review requirement

13. Example Policies
Example 1: CMO
Team: Marketing
Role: DomainOwner
Marketing: edit/approve
CrossFunctional: view
Finance: deny
Legal: deny
Engineering: deny
Example 2: Finance Telegram source feed
Domain: Finance
Sensitivity: domain_restricted
Allowed jobs: weekly summary, decision extraction
Outputs default to Finance/domain_restricted unless job overrides.
Example 3: Executive weekly digest job
Readable sources: selected Finance, Legal, Ops feeds
Operators: COO, Chief of Staff
Output domain: Leadership
Output sensitivity: leadership_restricted
Publication mode: reviewed

14. Access Model Summary
v1 access is configured manually at policy level through users, source feeds and jobs. Most materials inherit automatically. Fine-grained object ACL exists for exceptions only.

Knowledge Jobs Specification
1. Purpose
Knowledge Jobs are configured operations that read from allowed sources, process knowledge signals and produce structured outputs.
They are first-class product objects.

2. Supported Job Types
SummarizationJob
ExtractionJob
ConsolidationJob
MonitoringJob
TransformationJob
PublishingJob

3. Trigger Types
manual
scheduled
event_driven
window_based
conditional

4. Job Core Fields
Each KnowledgeJob must contain:
id
name
template_type
purpose
description
source_scope
trigger_type
trigger_config
output_policy
publication_mode
review_required
allowed_operator_roles
allowed_operator_users
config_json
active
created_at
updated_at

5. Source Scope
A job may read only explicitly declared source feeds.
Source scope may include:
one or more Telegram feeds
one or more Slack channels
one or more Jira boards
one or more Notion spaces
one or more meeting transcript feeds
Source scope must not default to entire corpus.

6. Trigger Definitions
6.1 Manual
Run on demand by authorized user.
6.2 Scheduled
Run by cron-like schedule. Example: every Friday at 17:00.
6.3 Event-driven
Run when event occurs. Examples:
new transcript imported
board updated
source feed sync completed
6.4 Window-based
Run over defined aggregation window. Example: summarize last 7 days of daily chats.
6.5 Conditional
Run when rule evaluates to true. Example: 3 or more blockers detected in a week.

7. Output Policies
Each job must define:
output entity type(s)
output domain
output sensitivity
publication mode
optional sanitization rules
optional linked review task creation
Publication modes
draft_only
reviewed_publish
auto_publish

8. Execution Flow
Validate operator or scheduler trigger
Resolve source scope
Resolve job read permissions
Read raw/canonical inputs
Build processing context
Execute extraction/summarization/transformation
Create output artifacts/entities
Apply output policy
Create review task if required
Persist run log and audit trail

9. Required Safety Rules
Jobs may read only from declared source scope.
Jobs may publish only to declared output domain.
Output sensitivity must be explicit.
Cross-domain publication requires explicit policy.
High-sensitivity outputs should default to reviewed_publish.
Job outputs must retain provenance links.

10. Example Job Definitions
Weekly Daily Digest
Purpose: Summarize progress, blockers, risks and decisions from daily communications.
Inputs:
Telegram team daily chat
Slack daily channel
Granola daily meeting notes
Trigger:
scheduled weekly
Outputs:
WeeklyDigest entity
optional ReviewTask
Planning Summary Job
Purpose: Generate structured planning summary after planning session.
Inputs:
planning transcript
Jira board state
planning notes
Trigger:
event-driven after transcript ingestion or manual
Outputs:
MeetingSummary entity
Decision entities
Action extraction artifact

11. Cursor Implementation Notes
The implementation should model jobs as persistent configurable records, not as hardcoded scripts. Processing pipeline should be reusable across trigger types. Job runs must be observable and auditable.

ERD / Data Model
1. Core Tables
users
id
email
name
status
primary_team_id
created_at
updated_at
teams
id
name
domain_default_id
created_at
updated_at
roles
id
name
description
user_roles
id
user_id
role_id
team_id nullable
created_at
domains
id
name
sensitivity_default
access_policy_id nullable
access_policies
id
name
policy_json
created_at
updated_at
user_domain_grants
id
user_id
domain_id
allow_view
allow_create
allow_edit
allow_approve
allow_export
created_at
entity_types
id
code
name
domain_id nullable
default_access_policy_id nullable
schema_json nullable
entities
id
type_id
title
body
owner_user_id nullable
owner_team_id nullable
domain_id
sensitivity_level
source_feed_id nullable
canonical_status
approval_status
freshness_status
process_binding nullable
access_policy_id nullable
created_at
updated_at
review_due_at nullable
archived_at nullable
entity_versions
id
entity_id
version_number
body_snapshot
metadata_snapshot
changed_by_user_id
change_reason nullable
created_at
entity_links
id
from_entity_id
to_entity_id
relation_type
confidence nullable
created_at
entity_acl
id
entity_id
subject_type
subject_id
permission
effect
created_at
connectors
id
type
name
status
auth_config_ref
created_at
updated_at
source_feeds
id
connector_id
external_ref
name
owner_user_id nullable
owner_team_id nullable
domain_id
knowledge_scope
sensitivity_level
sync_mode
allowed_jobs_policy nullable
created_at
updated_at
raw_artifacts
id
source_feed_id
external_artifact_id
artifact_type
raw_storage_path
checksum
metadata_json
created_at
chunks
id
entity_id
chunk_index
text
token_count
created_at
embeddings
id
chunk_id
embedding_vector
model_name
created_at
knowledge_jobs
id
name
template_type
purpose
description
trigger_type
trigger_config_json
output_domain_id
output_sensitivity_level
publication_mode
review_required
config_json
active
created_at
updated_at
knowledge_job_sources
id
knowledge_job_id
source_feed_id
knowledge_job_operators
id
knowledge_job_id
subject_type
subject_id
permission
job_runs
id
knowledge_job_id
status
triggered_by_user_id nullable
started_at
finished_at nullable
input_count nullable
output_count nullable
logs_text nullable
error_text nullable
job_outputs
id
job_run_id
entity_id nullable
output_type
publication_status
created_at
review_tasks
id
entity_id
reviewer_user_id
owner_user_id nullable
due_at
status
created_at
approval_flows
id
entity_type_id
config_json
active
audit_events
id
actor_user_id nullable
action_type
target_type
target_id
old_value_json nullable
new_value_json nullable
source
created_at
notifications
id
user_id
type
payload_json
status
created_at

2. Relationship Summary
users -> teams many-to-one
users -> roles many-to-many via user_roles
users -> domains many-to-many via user_domain_grants
entities -> entity_types many-to-one
entities -> domains many-to-one
entities -> source_feeds many-to-one optional
entities -> entity_versions one-to-many
entities -> entity_links one-to-many both directions
entities -> entity_acl one-to-many
connectors -> source_feeds one-to-many
source_feeds -> raw_artifacts one-to-many
knowledge_jobs -> source_feeds many-to-many via knowledge_job_sources
knowledge_jobs -> operators many-to-many via knowledge_job_operators
knowledge_jobs -> job_runs one-to-many
job_runs -> job_outputs one-to-many
entities -> chunks one-to-many
chunks -> embeddings one-to-one or one-to-many by model version

3. Notes for Implementation
JSONB is acceptable for metadata/config fields in v1.
policy_json, schema_json and config_json should remain validated at application layer.
entity_versions should be created on every material mutation.
all retrieval paths must join or prefilter through access constraints.

API Contract v1
1. Conventions
REST JSON API
authenticated requests only except login/bootstrap
all read APIs must enforce access model on backend
pagination required for list endpoints
audit events emitted for mutations

2. Auth & Identity
POST /api/auth/login
Purpose: authenticate user
GET /api/me
Purpose: return current user profile, roles, teams, domains

3. Users & Access
GET /api/users
List users
GET /api/users/:id
Get user details
PATCH /api/users/:id/access
Update user access profile Body:
team_id
role_ids
domain_grants
denies
overrides
GET /api/domains
List domains
GET /api/roles
List roles

4. Connectors & Source Feeds
GET /api/connectors
List connectors
POST /api/connectors
Create connector Body:
type
name
auth_config
GET /api/source-feeds
List source feeds
POST /api/source-feeds
Create source feed Body:
connector_id
external_ref
name
owner_user_id
owner_team_id
domain_id
knowledge_scope
sensitivity_level
sync_mode
PATCH /api/source-feeds/:id
Update source feed
POST /api/source-feeds/:id/sync
Run sync

5. Entities
GET /api/entities
Query entities with filters:
domain_id
type_id
owner_user_id
approval_status
freshness_status
source_feed_id
q
POST /api/entities
Create entity
GET /api/entities/:id
Get entity details
PATCH /api/entities/:id
Update entity
GET /api/entities/:id/versions
List versions
GET /api/entities/:id/links
List typed relations
POST /api/entities/:id/links
Create relation
POST /api/entities/:id/review
Create or update review state
POST /api/entities/:id/approve
Approve entity if permitted
POST /api/entities/:id/archive
Archive entity

6. Search & Retrieval
GET /api/search
Query params:
q
mode=keyword|semantic|hybrid
domain_id optional
type_id optional
POST /api/ask
Body:
question
filters optional Response:
answer
citations
supporting_entities
trace_id

7. Knowledge Jobs
GET /api/jobs
List knowledge jobs
POST /api/jobs
Create knowledge job Body:
name
template_type
purpose
description
trigger_type
trigger_config
output_domain_id
output_sensitivity_level
publication_mode
review_required
config_json
source_feed_ids
operator_bindings
GET /api/jobs/:id
Get job details
PATCH /api/jobs/:id
Update job
POST /api/jobs/:id/run
Run job manually
GET /api/jobs/:id/runs
List job runs
GET /api/job-runs/:id
Get run details

8. Review & Governance
GET /api/review-tasks
List review tasks
PATCH /api/review-tasks/:id
Update review task status
GET /api/audit-events
List audit events with filters

9. Errors
Standard response format:
code
message
details optional
Important codes:
unauthorized
forbidden
validation_error
not_found
conflict
job_execution_failed
sync_failed

10. Cursor Notes
Keep handler/service/repository split clean.
All list endpoints need pagination and filters.
Never trust frontend for access checks.
Ask endpoint must log answer trace with citations.

Epics / Implementation Plan
Epic 1. Identity & Access Foundation
Goal: Implement users, teams, roles, domains and permission resolution.
Backend tasks:
design tables for users, teams, roles, domains, grants
implement permission resolution service
implement access middleware/helpers
implement user access update API
Frontend tasks:
users list page
user access editor
roles/domains lookup UI
Acceptance:
backend enforces domain/type/action permissions
admin can assign access profiles

Epic 2. Connector & Source Feed Foundation
Goal: Implement connectors and source feed model.
Backend tasks:
connectors CRUD
source feeds CRUD
sync job model
raw artifact persistence
connector abstraction layer
Frontend tasks:
connectors page
source feed creation/edit UI
Acceptance:
admin can configure a source feed with owner, domain, scope and sensitivity

Epic 3. Telegram Ingestion v1
Goal: Support Telegram as controlled ingestion source.
Backend tasks:
Telegram connector implementation
fetch/import messages from explicit connected chats
store raw artifacts
map feed policies to imported artifacts
emit ingestion events
Frontend tasks:
Telegram source feed setup form
sync/run controls
Acceptance:
system reads only explicitly connected Telegram chats
imported artifacts are assigned domain/sensitivity through feed policy

Epic 4. Knowledge Core
Goal: Implement canonical entities, types, versions and links.
Backend tasks:
entity types seed
entity CRUD
entity versioning
entity linking
provenance support
Frontend tasks:
entity list page
entity detail page
entity edit/create forms
relation display
Acceptance:
entities can be created, edited, versioned and linked

Epic 5. Search & Retrieval
Goal: Implement keyword and hybrid retrieval.
Backend tasks:
chunking pipeline
embeddings generation
OpenSearch integration
retrieval service
permission-aware filtering in retrieval
Frontend tasks:
search UI
entity results list
filters by domain/type/status
Acceptance:
users can search only across allowed scope

Epic 6. AI Ask & Summarization
Goal: Implement scoped Q&A and summarization.
Backend tasks:
ask orchestrator
summarization service
answer trace storage
citations generation
Frontend tasks:
Ask UI
answer panel with citations
Acceptance:
AI answers only on allowed retrieved context and returns citations

Epic 7. Knowledge Jobs Engine
Goal: Implement persistent configurable jobs.
Backend tasks:
jobs CRUD
job trigger model
job execution orchestration
job run logs
source scope resolution
output policy application
Frontend tasks:
jobs list
job editor
manual run UI
run history UI
Acceptance:
job can be configured with source feeds, trigger and output policy

Epic 8. Review & Governance
Goal: Implement lifecycle, review tasks and approvals.
Backend tasks:
review task model
approval flow model
lifecycle transitions
freshness tracking
Frontend tasks:
review queue
approval actions
stale content indicators
Acceptance:
review-required outputs cannot publish without required review

Epic 9. Audit & Observability
Goal: Implement audit events and operational visibility.
Backend tasks:
audit event emission
metrics hooks
job/sync error tracking
Frontend tasks:
audit log view
job run diagnostics
sync status UI
Acceptance:
mutations and runs are traceable

Epic 10. Additional Connectors
Goal: Expand connectors after Telegram foundation.
Targets:
Slack
Email
Fireflies
Granola
Jira
Trello
Notion
Google Drive
Acceptance:
each connector conforms to source feed architecture and policy inheritance rules

Delivery Order Recommendation
Epic 1 Identity & Access Foundation
Epic 2 Connector & Source Feed Foundation
Epic 3 Telegram Ingestion v1
Epic 4 Knowledge Core
Epic 5 Search & Retrieval
Epic 7 Knowledge Jobs Engine
Epic 8 Review & Governance
Epic 6 AI Ask & Summarization
Epic 9 Audit & Observability
Epic 10 Additional Connectors

Cursor Handoff Notes
For Cursor implementation:
keep domain modules separated inside Go codebase;
enforce access at service layer and retrieval layer;
do not hardcode Telegram logic into generic ingestion interfaces;
treat knowledge jobs as first-class persisted objects;
make every mutation auditable;
implement inheritance rules before object-level overrides;
ensure API handlers remain thin and orchestration stays in services.
