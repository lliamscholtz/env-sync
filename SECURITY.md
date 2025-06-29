# Security Best Practices for env-sync

This document provides comprehensive security guidance for deploying and using env-sync in production environments.

## 🔐 Encryption Key Management

### Key Generation and Distribution

**✅ Best Practices:**
- Generate keys using `env-sync generate-key` (uses cryptographically secure random generation)
- Use different encryption keys for different environments (dev/staging/production)
- Rotate keys quarterly or after any suspected compromise
- Store keys in enterprise password managers (1Password, Bitwarden, etc.)

**❌ Security Risks to Avoid:**
- Never use the `--key` CLI parameter in production (visible in process lists)
- Never commit keys to version control
- Never store keys in plain text files without proper permissions
- Never reuse keys across different projects or environments

### Secure Key Storage Options

**1. Environment Variables (Recommended for CI/CD)**
```bash
export ENVSYNC_ENCRYPTION_KEY="<base64-key>"
```
- Secure environment variable management in CI/CD systems
- Use secret management features (GitHub Secrets, Azure DevOps Variables)
- Ensure environment isolation between different deployment stages

**2. Key Files (For Local Development)**
```bash
# Create key file with restricted permissions
echo "<base64-key>" > .env-sync-key
chmod 600 .env-sync-key
echo ".env-sync-key" >> .gitignore
```

**3. Interactive Prompt (Most Secure for Manual Operations)**
- Use `key_source: prompt` for maximum security
- Key never stored on disk or in environment
- Suitable for one-time operations or highly sensitive environments

## 🏢 Azure Key Vault Security

### Access Control and Permissions

**Azure RBAC Configuration:**
```bash
# Principle of least privilege - grant minimal required permissions
az role assignment create \
  --role "Key Vault Secrets User" \
  --assignee <user-or-service-principal> \
  --scope /subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault-name>
```

**Required Permissions:**
- `Microsoft.KeyVault/vaults/secrets/read` - Pull secrets
- `Microsoft.KeyVault/vaults/secrets/write` - Push secrets
- `Microsoft.KeyVault/vaults/secrets/delete` - Key rotation (if needed)

**Security Hardening:**
```bash
# Enable Key Vault access logging
az monitor diagnostic-settings create \
  --name "KeyVaultAuditLogs" \
  --resource "/subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault-name>" \
  --logs '[{"category":"AuditEvent","enabled":true}]'

# Enable soft delete and purge protection
az keyvault update \
  --name <vault-name> \
  --enable-soft-delete true \
  --enable-purge-protection true
```

### Network Security

**Private Endpoints (Recommended for Production):**
```bash
# Create private endpoint for Key Vault
az network private-endpoint create \
  --name <vault-name>-pe \
  --resource-group <rg> \
  --vnet-name <vnet> \
  --subnet <subnet> \
  --private-connection-resource-id "/subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault-name>" \
  --group-id vault \
  --connection-name <connection-name>
```

**Firewall Rules:**
```bash
# Restrict access to specific IP ranges
az keyvault network-rule add \
  --name <vault-name> \
  --ip-address <your-ip-range>

# Deny default access
az keyvault update \
  --name <vault-name> \
  --default-action Deny
```

## 🖥️ System-Level Security

### File System Permissions

**Configuration Files:**
```bash
# Secure configuration file permissions
chmod 600 .env-sync.yaml
chmod 600 .env-sync.*.yaml

# Secure environment files
chmod 600 .env*

# Secure key files
chmod 600 .env-sync-key*
```

**Directory Structure:**
```bash
# Recommended directory permissions
mkdir -p ~/.config/env-sync
chmod 700 ~/.config/env-sync

# Move sensitive files to secure location
mv .env-sync.yaml ~/.config/env-sync/
mv .env-sync-key ~/.config/env-sync/
```

### Process Security

**Running env-sync Securely:**
```bash
# Use dedicated service account for production
sudo useradd -r -s /bin/false env-sync-service

# Set proper ownership
sudo chown env-sync-service:env-sync-service /opt/env-sync/
sudo chmod 750 /opt/env-sync/

# Run with minimal privileges
sudo -u env-sync-service env-sync pull
```

## 🏗️ Production Deployment Patterns

### Multi-Environment Setup

**Environment Isolation:**
```yaml
# .env-sync.prod.yaml
vault_url: https://prod-vault.vault.azure.net/
secret_name: app-prod-secrets
env_file: .env.prod
key_source: env  # Use ENVSYNC_PROD_KEY environment variable
```

```yaml
# .env-sync.dev.yaml  
vault_url: https://dev-vault.vault.azure.net/
secret_name: app-dev-secrets
env_file: .env.dev
key_source: env  # Use ENVSYNC_DEV_KEY environment variable
```

### CI/CD Integration

**GitHub Actions Example:**
```yaml
name: Deploy Production
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v3
      
      - name: Azure Login
        uses: azure/login@v1
        with:
          creds: ${{ secrets.AZURE_CREDENTIALS }}
      
      - name: Sync Environment Variables
        env:
          ENVSYNC_ENCRYPTION_KEY: ${{ secrets.ENVSYNC_PROD_KEY }}
        run: |
          env-sync pull --sync-file .env-sync.prod.yaml
```

**Security Considerations for CI/CD:**
- Use environment-specific secrets
- Enable branch protection rules
- Require approval for production deployments
- Use OIDC for Azure authentication when possible
- Never log sensitive outputs

### Container Deployment

**Docker Security:**
```dockerfile
# Use non-root user
FROM alpine:latest
RUN adduser -D -s /bin/sh env-sync-user

# Copy binary and configs with proper ownership
COPY --chown=env-sync-user:env-sync-user env-sync /usr/local/bin/
COPY --chown=env-sync-user:env-sync-user .env-sync.yaml /app/

# Switch to non-root user
USER env-sync-user
WORKDIR /app

# Run with restricted capabilities
ENTRYPOINT ["env-sync"]
```

**Kubernetes Security:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: env-sync-key
type: Opaque
data:
  encryption-key: <base64-encoded-key>

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
      - name: app
        image: myapp:latest
        env:
        - name: ENVSYNC_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: env-sync-key
              key: encryption-key
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
```

## 🔄 Key Rotation Procedures

### Scheduled Key Rotation

**Quarterly Rotation Process:**
```bash
# 1. Generate new key
env-sync generate-key > new-key.txt

# 2. Distribute new key to team securely
# 3. Update all systems with new key
# 4. Rotate stored secrets

env-sync rotate-key \
  --old-key-source env \
  --new-key "$(cat new-key.txt)" \
  --sync-file .env-sync.prod.yaml

# 5. Verify rotation
env-sync pull --sync-file .env-sync.prod.yaml

# 6. Securely delete old key material
shred -vfz -n 3 new-key.txt
```

### Emergency Key Rotation

**In Case of Compromise:**
```bash
# 1. Immediately generate and deploy new key
env-sync generate-key

# 2. Update all production systems ASAP
# 3. Rotate all affected secrets
# 4. Review access logs for unauthorized access
# 5. Update incident response documentation
```

## 🚨 Incident Response

### Monitoring and Alerting

**Key Vault Access Monitoring:**
```bash
# Set up alerts for Key Vault access
az monitor activity-log alert create \
  --name "KeyVault-Unauthorized-Access" \
  --description "Alert on failed Key Vault access attempts" \
  --condition category=Security \
  --condition resourceId="/subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.KeyVault/vaults/<vault-name>"
```

**Log Analysis:**
```bash
# Review Key Vault audit logs
az monitor activity-log list \
  --resource-group <rg> \
  --start-time 2024-01-01 \
  --end-time 2024-01-31 \
  --query "[?resourceId.resourceType=='Microsoft.KeyVault/vaults']"
```

### Breach Response Checklist

**Immediate Actions:**
1. ✅ Identify scope of potential compromise
2. ✅ Generate new encryption keys immediately
3. ✅ Rotate all affected Key Vault secrets
4. ✅ Update all systems with new keys
5. ✅ Review and analyze access logs
6. ✅ Document timeline and impact
7. ✅ Notify stakeholders as required

## 📋 Security Checklist

### Pre-Production Checklist

**Infrastructure Security:**
- [ ] Key Vault access restricted by IP/network
- [ ] Private endpoints configured for Key Vault
- [ ] Audit logging enabled for Key Vault
- [ ] Soft delete and purge protection enabled
- [ ] RBAC permissions follow least privilege principle

**Application Security:**
- [ ] Different encryption keys per environment
- [ ] Configuration files have restricted permissions (600)
- [ ] No hardcoded secrets in code or configs
- [ ] Key files added to .gitignore
- [ ] Environment variables properly secured

**Operational Security:**
- [ ] Key rotation schedule established
- [ ] Incident response procedures documented
- [ ] Monitoring and alerting configured
- [ ] Backup and recovery procedures tested
- [ ] Team trained on security procedures

### Regular Security Reviews

**Monthly:**
- [ ] Review Key Vault access logs
- [ ] Verify configuration file permissions
- [ ] Check for unauthorized access attempts

**Quarterly:**
- [ ] Rotate encryption keys
- [ ] Review and update RBAC permissions
- [ ] Test incident response procedures
- [ ] Update security documentation

**Annually:**
- [ ] Comprehensive security assessment
- [ ] Update threat model
- [ ] Review and update security policies
- [ ] Team security training refresh

## 🆘 Support and Reporting

### Security Issue Reporting

If you discover a security vulnerability in env-sync:

1. **DO NOT** create a public GitHub issue
2. Email security concerns to the maintainers
3. Include detailed information about the vulnerability
4. Allow time for responsible disclosure

### Getting Help

For security-related questions:
- Review this documentation thoroughly
- Check existing GitHub discussions
- Consult Azure Key Vault security documentation
- Engage security professionals for production deployments

---

**Remember:** Security is a shared responsibility. This tool provides secure defaults, but proper deployment and operational security depend on following these best practices.