# SRE Runbook: High HTTP 5xx Error Rate (`HighHttpErrorRate`)

## Alert Overview
- **Trigger**: More than 5% of HTTP requests over a 5-minute window returned status 5xx.
- **Severity**: Critical
- **Impact**: End-users experiencing request degradation or API transaction failures.

---

## Initial Triage Steps

1. **Verify Metric Dashboard**:
   - Check Grafana dashboard at `http://grafana.monitoring.svc:3000/d/yolo-dashboard`.
   - Identify whether the error spike affects all routes or specific paths (`/api/tasks`, `/readyz`).

2. **Inspect Application Logs**:
   ```bash
   kubectl logs -n yolo-app -l app=yolo-api --tail=100 -f | jq 'select(.level=="ERROR")'
   ```
   Look for:
   - Database connection timeouts or pool exhaustion errors (`connection refused`, `context deadline exceeded`).
   - Query panics or unhandled nil pointer exceptions.

3. **Check Database Health**:
   ```bash
   # Check pod readiness
   kubectl get pods -n yolo-app -l app=yolo-api

   # Query database readiness endpoint directly
   kubectl exec -n yolo-app -it deploy/yolo-api -- wget -qO- http://localhost:8080/readyz
   ```

---

## Remediation & Recovery

### Scenario A: Database Connectivity Failure
1. Check RDS status in AWS Console / Terraform state.
2. If connection pool is saturated, temporarily scale up pod replicas or adjust `DB_MAX_OPEN_CONNS`:
   ```bash
   kubectl scale deployment yolo-api -n yolo-app --replicas=5
   ```

### Scenario B: Bad Code Deployment / Rollout Regression
1. Roll back to previous known stable revision:
   ```bash
   kubectl rollout undo deployment/yolo-api -n yolo-app
   kubectl rollout status deployment/yolo-api -n yolo-app
   ```

2. Verify recovery:
   ```bash
   curl -i http://api.yolo-app.local/readyz
   ```
