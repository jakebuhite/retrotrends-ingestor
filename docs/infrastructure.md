# Infrastructure Setup

This document is a resource-by-resource reference for provisioning the AWS infrastructure required by RetroTrends. It covers both the ingestor and API environments. Resources are ordered by dependency — create them top to bottom.

All infrastructure lives in a single AWS region (e.g. `us-east-1`). Choose one and be consistent.

---

## Prerequisites

- AWS account with admin access (or scoped IAM permissions covering VPC, ECS, RDS, ECR, IAM, Secrets Manager, EventBridge, ACM, and CloudWatch)
- A registered domain name with DNS managed in Route 53 (or externally) for the API's ALB certificate
- Docker installed locally (to push initial base images or verify builds)
- AWS CLI configured (`aws configure`)

---

## Setup Order

Resources must be created in this order due to dependencies:

```
1.  VPC & Networking
2.  Security Groups
3.  Secrets Manager
4.  RDS PostgreSQL
5.  ECR Repositories
6.  CloudWatch Log Groups
7.  IAM Roles
8.  ECS Cluster
9.  ECS Task Definitions
10. ACM Certificate
11. Application Load Balancer
12. ECS Service (API)
13. EventBridge Scheduler (Ingestor)
14. GitHub OIDC Provider & CI/CD Roles
```

---

## 1. VPC & Networking

Create a dedicated VPC. Do not use the default VPC.

**VPC**
- CIDR: `10.0.0.0/16`
- Enable DNS hostnames: yes
- Enable DNS resolution: yes

**Subnets** — create in two Availability Zones (e.g. `us-east-1a`, `us-east-1b`)

| Name | Type | AZ | CIDR |
|------|------|----|------|
| `retrotrends-public-1a` | Public | `us-east-1a` | `10.0.1.0/24` |
| `retrotrends-public-1b` | Public | `us-east-1b` | `10.0.2.0/24` |
| `retrotrends-private-1a` | Private | `us-east-1a` | `10.0.11.0/24` |
| `retrotrends-private-1b` | Private | `us-east-1b` | `10.0.12.0/24` |

**Internet Gateway**
- Create one IGW and attach it to the VPC.

**NAT Gateway**
- Create one NAT Gateway in `retrotrends-public-1a` (single NAT is sufficient for this workload).
- Allocate an Elastic IP and associate it.

**Route Tables**

*Public route table* (attach to both public subnets):
- `0.0.0.0/0` → Internet Gateway

*Private route table* (attach to both private subnets):
- `0.0.0.0/0` → NAT Gateway

---

## 2. Security Groups

All security groups belong to the VPC created above.

### `retrotrends-alb-sg`
Allows public HTTPS traffic to the load balancer.

| Direction | Protocol | Port | Source |
|-----------|----------|------|--------|
| Inbound | TCP | 443 | `0.0.0.0/0` |
| Inbound | TCP | 80 | `0.0.0.0/0` (redirect to 443) |
| Outbound | All | All | `0.0.0.0/0` |

### `retrotrends-api-sg`
Allows the ALB to reach the API container. No direct public access.

| Direction | Protocol | Port | Source |
|-----------|----------|------|--------|
| Inbound | TCP | 8080 | `retrotrends-alb-sg` |
| Outbound | All | All | `0.0.0.0/0` |

### `retrotrends-ingestor-sg`
The ingestor makes no inbound connections. Outbound covers RDS and external APIs (eBay, IGDB) via NAT.

| Direction | Protocol | Port | Source |
|-----------|----------|------|--------|
| Outbound | All | All | `0.0.0.0/0` |

### `retrotrends-rds-sg`
Allows only the two ECS task security groups to reach the database.

| Direction | Protocol | Port | Source |
|-----------|----------|------|--------|
| Inbound | TCP | 5432 | `retrotrends-api-sg` |
| Inbound | TCP | 5432 | `retrotrends-ingestor-sg` |
| Outbound | All | All | `0.0.0.0/0` |

---

## 3. Secrets Manager

Create secrets **before** RDS — the DB secret will be referenced in the RDS setup and in the ECS task definitions.

### `retrotrends/db`
Stores the RDS master credentials. Use the **RDS-managed secret** option when creating the RDS instance (AWS will auto-create and auto-rotate this secret). If creating manually:

```json
{
  "username": "retrotrends",
  "password": "<generated>",
  "host": "<rds-endpoint>",
  "port": 5432,
  "dbname": "retrotrends"
}
```

### `retrotrends/ebay`
eBay API credentials for the ingestor.

```json
{
  "client_id": "<ebay-client-id>",
  "client_secret": "<ebay-client-secret>"
}
```

### `retrotrends/igdb`
IGDB (Twitch) OAuth credentials for the ingestor backfill job.

```json
{
  "client_id": "<igdb-client-id>",
  "client_secret": "<igdb-client-secret>"
}
```

---

## 4. RDS PostgreSQL

**Subnet group**
- Name: `retrotrends-rds-subnet-group`
- Subnets: both private subnets (`retrotrends-private-1a`, `retrotrends-private-1b`)

**Parameter group**
- Family: `postgres16`
- Name: `retrotrends-pg16`
- Customise if needed (e.g. `pg_trgm` extension is enabled by default in PostgreSQL 16 — no parameter change required, just run `CREATE EXTENSION` after the DB is created)

**Instance**

| Setting | Value |
|---------|-------|
| Engine | PostgreSQL 16 |
| Instance class | `db.t4g.medium` |
| Storage | 20 GB gp3, autoscaling enabled (max 100 GB) |
| Multi-AZ | No (single instance for MVP; enable for production hardening) |
| DB identifier | `retrotrends` |
| Initial database name | `retrotrends` |
| Master username | `retrotrends` |
| Master password | Managed by Secrets Manager (use "Manage master credentials in Secrets Manager") |
| VPC | RetroTrends VPC |
| Subnet group | `retrotrends-rds-subnet-group` |
| Security group | `retrotrends-rds-sg` |
| Public access | No |
| Backup retention | 7 days |
| Deletion protection | Enabled |

After the instance is available, run the initial schema migration by connecting from a bastion host or via RDS Proxy / SSM Session Manager tunneling:

```sql
\c retrotrends
-- then run migrations/001_initial_schema.sql from the API repo
```

---

## 5. ECR Repositories

Create two private ECR repositories.

| Repository name | Scan on push | Lifecycle policy |
|-----------------|-------------|-----------------|
| `retrotrends-ingestor` | Yes | Keep last 10 images |
| `retrotrends-api` | Yes | Keep last 10 images |

**Lifecycle policy** (apply to both repos):

```json
{
  "rules": [
    {
      "rulePriority": 1,
      "description": "Keep last 10 images",
      "selection": {
        "tagStatus": "any",
        "countType": "imageCountMoreThan",
        "countNumber": 10
      },
      "action": { "type": "expire" }
    }
  ]
}
```

---

## 6. CloudWatch Log Groups

Create log groups before the ECS task definitions reference them.

| Log group name | Retention |
|---------------|-----------|
| `/ecs/retrotrends-ingestor` | 30 days |
| `/ecs/retrotrends-api` | 30 days |

---

## 7. IAM Roles

### ECS Task Execution Role
Used by the ECS agent to pull images from ECR and write logs to CloudWatch. This role is shared by all tasks.

- **Name:** `retrotrends-ecs-execution-role`
- **Trust policy:** `ecs-tasks.amazonaws.com`
- **Managed policies to attach:**
  - `AmazonECSTaskExecutionRolePolicy` (ECR pull + CloudWatch Logs)
- **Inline policy** — allows reading the three Secrets Manager secrets:

```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:GetSecretValue"],
  "Resource": [
    "arn:aws:secretsmanager:us-east-1:*:secret:retrotrends/db*",
    "arn:aws:secretsmanager:us-east-1:*:secret:retrotrends/ebay*",
    "arn:aws:secretsmanager:us-east-1:*:secret:retrotrends/igdb*"
  ]
}
```

### ECS Task Role — Ingestor
Runtime permissions for the ingestor container itself. Currently no AWS API calls are made from within the ingestor (all config comes via environment variables), so this role can start empty. Add permissions here if the ingestor ever needs to write to S3, publish to SNS, etc.

- **Name:** `retrotrends-ingestor-task-role`
- **Trust policy:** `ecs-tasks.amazonaws.com`
- **Policies:** None (placeholder for future use)

### ECS Task Role — API
Runtime permissions for the API container.

- **Name:** `retrotrends-api-task-role`
- **Trust policy:** `ecs-tasks.amazonaws.com`
- **Policies:** None (the API is read-only against RDS; credentials come via environment variable from Secrets Manager at task start)

---

## 8. ECS Cluster

- **Name:** `retrotrends`
- **Infrastructure:** AWS Fargate (no EC2 capacity providers needed)
- **Container Insights:** Enabled (CloudWatch metrics per task)

---

## 9. ECS Task Definitions

Both task definitions use Fargate. The container image URIs use the ECR registry URLs from step 5.

### Ingestor Task Definition

The ingestor image is a single binary that accepts a subcommand. The task definition does not hardcode the subcommand — it is supplied as a command override when EventBridge triggers a run.

| Setting | Value |
|---------|-------|
| Family | `retrotrends-ingestor` |
| Launch type | Fargate |
| CPU | 512 (.5 vCPU) |
| Memory | 1024 (1 GB) |
| Task execution role | `retrotrends-ecs-execution-role` |
| Task role | `retrotrends-ingestor-task-role` |
| Network mode | `awsvpc` |

**Container definition:**

| Field | Value |
|-------|-------|
| Name | `ingestor` |
| Image | `<ecr-registry>/retrotrends-ingestor:latest` |
| Essential | Yes |
| Log driver | `awslogs` |
| Log options | group: `/ecs/retrotrends-ingestor`, region: `us-east-1`, stream prefix: `ecs` |

**Environment variables** (injected from Secrets Manager using `valueFrom`):

| Env var | Secret ARN / key |
|---------|-----------------|
| `DATABASE_URL` | `retrotrends/db` → construct from host/port/dbname/user/pass |
| `EBAY_CLIENT_ID` | `retrotrends/ebay` → `client_id` |
| `EBAY_CLIENT_SECRET` | `retrotrends/ebay` → `client_secret` |
| `IGDB_CLIENT_ID` | `retrotrends/igdb` → `client_id` |
| `IGDB_CLIENT_SECRET` | `retrotrends/igdb` → `client_secret` |

**Plain environment variables** (set in the task definition directly):

| Env var | Value |
|---------|-------|
| `EBAY_DAILY_CALL_LIMIT` | `5000` |
| `INGEST_MAX_PAGES` | `5` |

### API Task Definition

| Setting | Value |
|---------|-------|
| Family | `retrotrends-api` |
| Launch type | Fargate |
| CPU | 1024 (1 vCPU) |
| Memory | 2048 (2 GB) |
| Task execution role | `retrotrends-ecs-execution-role` |
| Task role | `retrotrends-api-task-role` |
| Network mode | `awsvpc` |

**Container definition:**

| Field | Value |
|-------|-------|
| Name | `api` |
| Image | `<ecr-registry>/retrotrends-api:latest` |
| Essential | Yes |
| Port mappings | 8080 TCP |
| Log driver | `awslogs` |
| Log options | group: `/ecs/retrotrends-api`, region: `us-east-1`, stream prefix: `ecs` |

**Environment variables** (from Secrets Manager):

| Env var | Secret |
|---------|--------|
| `DATABASE_URL` | `retrotrends/db` |

**Plain environment variables:**

| Env var | Value |
|---------|-------|
| `SERVER_PORT` | `8080` |

**Health check:**
- Command: `curl -f http://localhost:8080/actuator/health || exit 1`
- Interval: 30s, timeout: 5s, retries: 3, start period: 60s

---

## 10. ACM Certificate

Request a public certificate for the API domain (e.g. `api.retrotrends.io`).

- **Validation method:** DNS validation (add the CNAME record to Route 53 or your DNS provider)
- **Region:** Must be in the same region as the ALB (`us-east-1`)

Wait for the certificate status to become `Issued` before creating the ALB listener.

---

## 11. Application Load Balancer

### ALB

| Setting | Value |
|---------|-------|
| Name | `retrotrends-api-alb` |
| Scheme | Internet-facing |
| IP address type | IPv4 |
| VPC | RetroTrends VPC |
| Subnets | Both public subnets |
| Security group | `retrotrends-alb-sg` |

### Target Group

| Setting | Value |
|---------|-------|
| Name | `retrotrends-api-tg` |
| Target type | IP (required for Fargate) |
| Protocol | HTTP |
| Port | 8080 |
| VPC | RetroTrends VPC |
| Health check path | `/actuator/health` |
| Health check interval | 30s |
| Healthy threshold | 2 |
| Unhealthy threshold | 3 |

### Listeners

| Port | Protocol | Action |
|------|----------|--------|
| 80 | HTTP | Redirect to HTTPS (301) |
| 443 | HTTPS | Forward to `retrotrends-api-tg` |

Attach the ACM certificate to the port 443 listener.

---

## 12. ECS Service (API)

| Setting | Value |
|---------|-------|
| Cluster | `retrotrends` |
| Service name | `retrotrends-api` |
| Launch type | Fargate |
| Task definition | `retrotrends-api` (latest revision) |
| Desired count | 1 |
| Subnets | Both private subnets |
| Security group | `retrotrends-api-sg` |
| Public IP | Disabled |
| Load balancer | `retrotrends-api-alb` |
| Target group | `retrotrends-api-tg` |
| Container to load balance | `api:8080` |
| Deployment type | Rolling update |
| Min healthy percent | 100 |
| Max percent | 200 |

**Deployment circuit breaker:** Enable with automatic rollback. This rolls back to the previous task definition revision automatically if the new deployment fails health checks.

---

## 13. EventBridge Scheduler (Ingestor)

Both schedules target the same ECS cluster and task definition. The subcommand is passed as the container command override.

### Ingest Schedule

| Setting | Value |
|---------|-------|
| Name | `retrotrends-ingest` |
| Schedule expression | `cron(0 2 * * ? *)` (02:00 UTC daily) |
| Target | ECS `RunTask` |
| Cluster | `retrotrends` |
| Task definition | `retrotrends-ingestor` (latest revision) |
| Launch type | Fargate |
| Subnets | Both private subnets |
| Security group | `retrotrends-ingestor-sg` |
| Container override | `ingestor` → command: `["ingest"]` |
| Execution role | `retrotrends-ecs-execution-role` |

**Scheduler execution role** — EventBridge needs permission to call `ecs:RunTask` and `iam:PassRole`:

- **Name:** `retrotrends-eventbridge-scheduler-role`
- **Trust policy:** `scheduler.amazonaws.com`
- **Inline policy:**

```json
{
  "Effect": "Allow",
  "Action": ["ecs:RunTask"],
  "Resource": "arn:aws:ecs:us-east-1:*:task-definition/retrotrends-ingestor:*"
},
{
  "Effect": "Allow",
  "Action": ["iam:PassRole"],
  "Resource": [
    "arn:aws:iam::*:role/retrotrends-ecs-execution-role",
    "arn:aws:iam::*:role/retrotrends-ingestor-task-role"
  ]
}
```

### Revisit Schedule

Same configuration as above, with:
- **Name:** `retrotrends-revisit`
- **Schedule expression:** `cron(0 4 * * ? *)` (04:00 UTC daily)
- **Container override:** `ingestor` → command: `["revisit"]`

---

## 14. GitHub OIDC Provider & CI/CD Roles

This enables GitHub Actions to authenticate to AWS without long-lived access keys. See [cicd.md](cicd.md) for the full CI/CD context.

### OIDC Identity Provider

Create one OIDC provider in IAM (shared by both repos):

| Setting | Value |
|---------|-------|
| Provider URL | `https://token.actions.githubusercontent.com` |
| Audience | `sts.amazonaws.com` |

Fetch and confirm the thumbprint from GitHub's OIDC endpoint.

### CI/CD Role — Ingestor

- **Name:** `retrotrends-github-ingestor-deploy`
- **Trust policy:** Allows `sts:AssumeRoleWithWebIdentity` from the OIDC provider, scoped to the ingestor repo on `main`:

```json
{
  "Effect": "Allow",
  "Principal": {
    "Federated": "arn:aws:iam::<account-id>:oidc-provider/token.actions.githubusercontent.com"
  },
  "Action": "sts:AssumeRoleWithWebIdentity",
  "Condition": {
    "StringEquals": {
      "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
      "token.actions.githubusercontent.com:sub": "repo:jakebuhite/retrotrends-ingestor:ref:refs/heads/main"
    }
  }
}
```

- **Inline policy permissions:**

```json
[
  { "Effect": "Allow", "Action": ["ecr:GetAuthorizationToken"], "Resource": "*" },
  { "Effect": "Allow", "Action": [
      "ecr:BatchCheckLayerAvailability", "ecr:PutImage",
      "ecr:InitiateLayerUpload", "ecr:UploadLayerPart", "ecr:CompleteLayerUpload"
    ],
    "Resource": "arn:aws:ecr:us-east-1:*:repository/retrotrends-ingestor"
  },
  { "Effect": "Allow", "Action": [
      "ecs:RegisterTaskDefinition", "ecs:DescribeTaskDefinition"
    ],
    "Resource": "*"
  },
  { "Effect": "Allow", "Action": ["scheduler:UpdateSchedule", "scheduler:GetSchedule"],
    "Resource": [
      "arn:aws:scheduler:us-east-1:*:schedule/default/retrotrends-ingest",
      "arn:aws:scheduler:us-east-1:*:schedule/default/retrotrends-revisit"
    ]
  },
  { "Effect": "Allow", "Action": ["iam:PassRole"],
    "Resource": [
      "arn:aws:iam::*:role/retrotrends-ecs-execution-role",
      "arn:aws:iam::*:role/retrotrends-ingestor-task-role"
    ]
  }
]
```

### CI/CD Role — API

- **Name:** `retrotrends-github-api-deploy`
- **Trust policy:** Same structure as above but scoped to `repo:jakebuhite/retrotrends-api:ref:refs/heads/main`
- **Inline policy permissions:**

```json
[
  { "Effect": "Allow", "Action": ["ecr:GetAuthorizationToken"], "Resource": "*" },
  { "Effect": "Allow", "Action": [
      "ecr:BatchCheckLayerAvailability", "ecr:PutImage",
      "ecr:InitiateLayerUpload", "ecr:UploadLayerPart", "ecr:CompleteLayerUpload"
    ],
    "Resource": "arn:aws:ecr:us-east-1:*:repository/retrotrends-api"
  },
  { "Effect": "Allow", "Action": [
      "ecs:RegisterTaskDefinition", "ecs:DescribeTaskDefinition"
    ],
    "Resource": "*"
  },
  { "Effect": "Allow", "Action": ["ecs:UpdateService", "ecs:DescribeServices"],
    "Resource": "arn:aws:ecs:us-east-1:*:service/retrotrends/retrotrends-api"
  },
  { "Effect": "Allow", "Action": ["iam:PassRole"],
    "Resource": [
      "arn:aws:iam::*:role/retrotrends-ecs-execution-role",
      "arn:aws:iam::*:role/retrotrends-api-task-role"
    ]
  }
]
```

---

## Verification Checklist

After completing all steps, verify the stack is working end to end:

- [ ] RDS instance is `Available`; connect from a private subnet host and confirm `retrotrends` database exists
- [ ] Run the initial schema migration (`migrations/001_initial_schema.sql`) and confirm tables exist
- [ ] Push a test image to both ECR repositories and confirm the push succeeds
- [ ] Manually trigger the API ECS service; confirm the task reaches `RUNNING` and the health check passes at `/actuator/health`
- [ ] ALB DNS resolves and returns HTTP 200 at `https://api.retrotrends.io/v1/games`
- [ ] Manually trigger a one-off `retrotrends-ingestor` task with command `["backfill"]`; confirm it exits 0 and rows appear in the `games` table
- [ ] Manually trigger with command `["ingest"]`; confirm rows appear in `listings`
- [ ] Confirm both EventBridge schedules are `Enabled` in the scheduler console
- [ ] Push a commit to `main` in each repo and confirm the GitHub Actions deploy workflows complete successfully
