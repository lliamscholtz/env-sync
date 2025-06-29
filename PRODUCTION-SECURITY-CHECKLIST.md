# Production Security Checklist for env-sync

Use this checklist to ensure secure deployment of env-sync in production environments.

## 📋 Pre-Deployment Security Checklist

### 🔐 Encryption Key Management

**Key Generation & Storage**
- [ ] Generated encryption keys using `env-sync generate-key` (cryptographically secure)
- [ ] Created separate encryption keys for each environment (dev/staging/prod)
- [ ] Stored keys in enterprise password manager or secure key management system
- [ ] Never used `--key` CLI parameter (visible in process lists)
- [ ] Key files have `chmod 600` permissions if using file storage
- [ ] Added `.env-sync-key*` patterns to `.gitignore`

**Key Distribution**
- [ ] Distributed keys through secure channels (encrypted messaging, password managers)
- [ ] Documented key distribution process for team onboarding
- [ ] Established key rotation schedule (quarterly recommended)
- [ ] Tested key rotation procedure in non-production environment

### 🏢 Azure Key Vault Security

**Access Control**
- [ ] Configured Azure RBAC with principle of least privilege
- [ ] Granted only required permissions:
  - [ ] `Microsoft.KeyVault/vaults/secrets/read` for pull operations
  - [ ] `Microsoft.KeyVault/vaults/secrets/write` for push operations
  - [ ] `Microsoft.KeyVault/vaults/secrets/delete` only if key rotation needed
- [ ] Created separate service principals/managed identities per environment
- [ ] Removed development/test access from production Key Vault

**Network Security**
- [ ] Configured private endpoints for Key Vault (production environments)
- [ ] Set up Key Vault firewall rules to restrict IP access
- [ ] Disabled public network access if using private endpoints
- [ ] Configured DNS private zones for private endpoint resolution

**Audit & Compliance**
- [ ] Enabled Key Vault audit logging
- [ ] Configured log analytics workspace for Key Vault logs
- [ ] Set up monitoring alerts for:
  - [ ] Failed access attempts
  - [ ] Unusual access patterns
  - [ ] Administrative operations (create/delete secrets)
- [ ] Enabled soft delete and purge protection
- [ ] Documented compliance requirements (SOC2, GDPR, etc.)

### 🖥️ System-Level Security

**File System Security**
- [ ] Configuration files have `chmod 600` permissions:
  - [ ] `.env-sync.yaml`
  - [ ] `.env-sync.*.yaml`
  - [ ] `.env*` files
- [ ] Key files stored in secure directory (`~/.config/env-sync/`)
- [ ] Proper directory permissions (`chmod 700` for config directories)
- [ ] No sensitive files in version control

**Process Security**
- [ ] Running env-sync with dedicated service account (not root)
- [ ] Service account has minimal required permissions
- [ ] Process monitoring configured (no key exposure in process lists)
- [ ] Container security context configured if using containers:
  - [ ] Non-root user
  - [ ] Read-only root filesystem
  - [ ] Dropped capabilities
  - [ ] Security context constraints

### 🏗️ Deployment Configuration

**Environment Separation**
- [ ] Separate configuration files per environment:
  - [ ] `.env-sync.dev.yaml`
  - [ ] `.env-sync.staging.yaml`
  - [ ] `.env-sync.prod.yaml`
- [ ] Different Key Vaults for each environment
- [ ] Environment-specific encryption keys
- [ ] No cross-environment access permissions

**CI/CD Security**
- [ ] Encryption keys stored in CI/CD secret management
- [ ] Azure authentication using OIDC/service principals (not passwords)
- [ ] Branch protection rules for production deployments
- [ ] Required approvals for production changes
- [ ] Audit trail for all deployments
- [ ] No sensitive data in CI/CD logs

## 🚀 Deployment Security Checklist

### **Development Environment**
- [ ] Using `key_source: file` or `key_source: prompt`
- [ ] Key files in `.gitignore`
- [ ] Development Key Vault isolated from production
- [ ] Pull-only file watching mode (`env-sync watch` without `--push`)

### **Staging Environment**
- [ ] Using `key_source: env` with secure environment variable management
- [ ] Separate staging Key Vault and encryption key
- [ ] Automated testing of security controls
- [ ] Same security configuration as production (for testing)

### **Production Environment**
- [ ] Using `key_source: env` with production-grade secret management
- [ ] Private Key Vault with network restrictions
- [ ] Monitoring and alerting fully configured
- [ ] Backup and disaster recovery procedures tested
- [ ] Security incident response plan documented

## 📊 Operational Security Checklist

### 🔄 Regular Maintenance

**Monthly Reviews**
- [ ] Review Key Vault access logs for anomalies
- [ ] Verify file permissions on configuration files
- [ ] Check for unauthorized access attempts
- [ ] Review team access permissions

**Quarterly Tasks**
- [ ] Rotate encryption keys using `env-sync rotate-key`
- [ ] Review and update Azure RBAC permissions
- [ ] Test incident response procedures
- [ ] Update security documentation
- [ ] Security training for team members

**Annual Reviews**
- [ ] Comprehensive security assessment
- [ ] Penetration testing (if required)
- [ ] Update threat model and risk assessment
- [ ] Review and update security policies
- [ ] Third-party security audit (if required)

### 🚨 Incident Response Readiness

**Preparation**
- [ ] Incident response plan documented and tested
- [ ] Emergency contacts and escalation procedures defined
- [ ] Backup encryption keys stored securely
- [ ] Communication plan for security incidents
- [ ] Recovery procedures documented and tested

**Detection & Monitoring**
- [ ] Real-time monitoring of Key Vault access
- [ ] Automated alerts for security events
- [ ] Log aggregation and analysis tools configured
- [ ] Baseline behavior patterns established

## ✅ Validation & Testing

### **Security Testing**
- [ ] Penetration testing of Key Vault access controls
- [ ] Vulnerability scanning of deployment infrastructure
- [ ] Security code review completed
- [ ] Encryption key exposure testing (process lists, memory dumps)
- [ ] Access control testing (privilege escalation attempts)

### **Operational Testing**
- [ ] Key rotation procedure tested end-to-end
- [ ] Disaster recovery scenarios tested
- [ ] Backup and restore procedures validated
- [ ] Team security training completed
- [ ] Incident response drill conducted

### **Compliance Validation**
- [ ] Security controls documented and mapped to requirements
- [ ] Audit evidence collected and organized
- [ ] Compliance reporting mechanisms tested  
- [ ] Third-party assessment completed (if required)

## 🔍 Post-Deployment Monitoring

### **Continuous Monitoring**
- [ ] Key Vault access patterns monitored
- [ ] Configuration drift detection
- [ ] Security baseline monitoring
- [ ] Performance impact assessment
- [ ] Cost optimization review

### **Regular Reporting**
- [ ] Security metrics dashboard configured
- [ ] Monthly security posture reports
- [ ] Compliance status reporting
- [ ] Risk assessment updates
- [ ] Stakeholder communication plan

---

## 📞 Emergency Contacts

**Security Incident Response:**
- Primary: [Security Team Lead]
- Secondary: [DevOps Lead]
- Escalation: [CISO/Security Manager]

**Technical Support:**
- Infrastructure: [Cloud Team]
- Application: [Development Team]
- Azure Support: [Account Manager]

---

**✅ Checklist Complete Date:** _______________

**Reviewed By:** _______________

**Next Review Date:** _______________

---

> **Note:** This checklist should be customized based on your organization's specific security requirements, compliance obligations, and risk tolerance. Regular updates should be made as security best practices evolve.