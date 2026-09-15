# SRE Runbook: Pod CrashLooping (`KubePodCrashLooping`)

## Alert Overview
- **Trigger**: Container in pod has restarted more than twice in the past 5 minutes.
- **Severity**: Critical
- **Impact**: Pod capacity reduction, potential service disruption, or cascading overload.

---

## Initial Triage Steps

1. **Identify Failing Pods**:
   ```bash
   kubectl get pods -n yolo-app -l app=yolo-api --field-selector=status.phase!=Running
   ```

2. **Inspect Pod Termination Reason**:
   ```bash
   kubectl describe pod <pod-name> -n yolo-app | grep -A 10 "Last State"
   ```
   Check termination reason:
   - `OOMKilled`: Container exceeded memory limit (512Mi).
   - `Error` (Exit code 1): Application startup failure or panic.
   - `Exit code 137`: SIGKILL received (usually kernel OOM killer).

3. **Read Crash Logs from Previous Instance**:
   ```bash
   kubectl logs <pod-name> -n yolo-app --previous
   ```

---

## Remediation & Recovery

### Scenario A: OOMKilled
1. Review memory consumption graph in Grafana.
2. If legitimate workload growth, increase pod memory limits in Helm `values.yaml` or directly:
   ```bash
   kubectl set resources deployment yolo-api -n yolo-app -c yolo-api --limits=memory=1Gi --requests=memory=256Mi
   ```

### Scenario B: Database Secret / Configuration Missing
1. Check if ConfigMap or Secret keys are present:
   ```bash
   kubectl get secret yolo-api-secret -n yolo-app
   kubectl get configmap yolo-api-config -n yolo-app
   ```
2. Re-apply configurations via Helm or ArgoCD:
   ```bash
   helm upgrade yolo-app deploy/helm/yolo-app -n yolo-app
   ```
