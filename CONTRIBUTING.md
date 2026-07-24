# Contributing to YOLO DevOps Platform

Thank you for contributing to the YOLO DevOps Platform project! As a professional, structured team, we maintain high standards of code quality, linting, testing, and continuous integration.

## Contribution Workflow

1. **Fork/Branch**: Create a descriptive branch (e.g., `feature/add-kubernetes-hpa` or `bugfix/fix-db-connection-leak`).
2. **Pre-commit Hooks**: Install and configure `pre-commit` to verify code quality locally before pushing.
3. **Commit Messages**: We follow conventional commits guidelines:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation updates
   - `style:` for changes that do not affect code logic (formatting, missing semi-colons, etc.)
   - `refactor:` for code changes that neither fix a bug nor add a feature
   - `test:` for adding or modifying tests
   - `ci:` for CI/CD changes
   - `chore:` for updating build tasks, package manager configs, etc.
4. **Pull Requests**: Pull requests must pass all CI checks (linting, tests, security scans) before merge approval.

## Testing Locally

- Backend: `go test ./...`
- Local Environment: `docker compose up -d`
- Kubernetes: `helm lint ./deploy/helm/yolo-app`
