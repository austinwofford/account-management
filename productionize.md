# Release It!

## How would I architect this service from an infrastructure and systems standpoint?
I would keep the architecture similar to the current demo implementation:
A Go webserver, stateless JWT management, and Postgres. 

However, there are some things that would/could be added for production:
- Load balancing
- Managed Postgres with read replicas and primary/secondary for failover
- Likely some caching for rate limiting across pods and decreasing DB reads for refresh token validation
- Caching the access tokens so they can be immediately blacklisted if needed
- Kafka (or another messaging queue system) for async handling of email verification or other background account validation steps
- Container orchestration with Kubernetes
- Real secrets management with Vault or similar
- Some real observability and monitoring! Prometheus and Grafana, aggregated logs, tracing, alerts
- Infrastructure as code with Terraform
- A CI/CD pipeline

## What does productionizing this service mean to me?
Productionizing means implementing those above architectural changes and developing SLOs to answer the question: 
"How many 9's do we need and how much time/money can we spend to get them?".

Some main foci would be:
- Disaster Recovery: Automated backups, multi-region or multi-AZ deployments
- Durability/Reliability: graceful shutdown and degradation
- Performance: response times for a service like this need to be very fast (I would push for ~sub 100ms p95)
- Security: check dependency vulnerabilities, add rate limiting and backpressure, max request size enforcement, audit logging, SSL (likely handled upstream by proxy)
- Observability: metrics, logging, alerting, and tracing plus "business" metrics/dashboards
- Testing: load testing and integration testing
- Deployment: a CI/CD pipeline

## If given more time, what areas of implementation would I focus on?
- Metrics and tracing: request latency, error rates, RPS, active sessions, db connections, and resource usage
- Alerting
- Improved request validation with detailed error responses 
- Rate limiting
- Better logging of request IDs for correlating events to the requests
- Switch to a log aggregator like Loki or similar instead of using Dozzle
- Implement more security focused fields in the database like: ip_address, last_login_at, failed_login_attempts, etc. 
- pprof entry via the webserver gated behind config vars
- Deploy this with a production focused docker-compose.yml to a single server with a provider like Digital Ocean for further experimentation

## How would my solution scale? Anything I would change under high loads, etc?
I expect this service would scale very well considering its simplicity and Go is very performant. It would be fun to deploy a service like this on a single server and see how much it could handle. 
The obvious bottleneck is currently the database. The first approach I would take to improving this is optimizing SQL queries and the database access patterns.

However, if I were expecting high loads I would move toward:
- Horizontal scaling with Kubernetes
- Redis for caching refresh tokens and limiting DB accesses
- Load balancing with multi-region deployments
- Other things mentioned above

## How would I deploy this service?
1. I would setup the proper infrastructure:
- A reverse proxy with load balancing and TLS termination
- Kubernetes
- 2 replicas of the service with auto-scaling
- Managed Postgres with automatic backups and read replicas in multiple AZs
- Redis (if caching is added)
- Monitoring and observability tools

2. Define this infrastructure in code with Terraform

3. A deployment pipeline that:
- Runs tests
- Migrations
- Secrets injection
- Blue/Green deploy with healthchecks and automatic rollbacks

4. Development to production looks like:
- Local development
- PR Review
- Merge to main releases to staging
- Monitor metrics dashboards and logs
- Test in staging
- GitHub release triggers production deployment
- Monitor metrics dashboards and logs

## How did I use AI for this?
I used a few AI tools during this process. 
- I generally use ChatGPT for basic research and finding documentation these days. I've recently began using Gemini for some searching and find that it's comparable to when Google search was _good_. I used both of these tools for researching stateless authentication workflows and to validate possible issues with my approach in adapting this in a multi-service architecture.
- I use Claude Code for development. I find that it is best for discrete and well explained tasks.
- For this service in particular, I used Claude code to generate some testing (I didn't want to write test cases for password validation!). I also found it very helpful in outlining a professional looking README and validating configuration files for docker and Caddy. However, it didn't get any of these correct on the first try.
- One very compelling use case for these tools is having active peer review sessions during development.

## How would I extend the use of AI tooling in production?
I don't think I would attempt to build "AI" functionality into an account management/authentication service, but one thing I would try next time is writing a very explicit and detailed `claude.md` file that describes my preferences and standards for development. I also think with code generation tools, having good tests (including integration tests and manual testing/QA) becomes even more important.