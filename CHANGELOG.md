# Changelog

## Latest Updates (December 2024)

### 🎉 Major Feature Additions

#### Smart Conflict Detection & Resolution
- **Intelligent Key-Level Detection**: Now detects actual conflicts (same key, different values) instead of just content differences
- **Diff Display**: Shows exactly which keys conflict with local vs remote values
- **Multiple Resolution Strategies**: 
  - `manual` (default): User confirmation with diff display
  - `local`: Always use local changes
  - `remote`: Always use remote changes  
  - `merge`: Create conflict markers for manual resolution
  - `backup`: Create timestamped backups and use local
- **Push Integration**: Automatic conflict detection before every push operation

#### Robust File Change Detection
- **Atomic Write Support**: Handles all modern editors (VS Code, vim, nano, etc.)
- **Multiple Consecutive Changes**: Detects every file change, not just the first one
- **Smart Timing**: Reduced pull-prevention window from 10s to 3s
- **Health Checks**: Periodic verification that file watcher is still active
- **File Recreation Handling**: Automatically re-establishes watching after atomic writes

#### Enhanced User Experience
- **Push Enabled by Default**: File watcher now enables push with confirmation prompts for safety
- **Comprehensive Debug Mode**: Use `--debug` flag for detailed file event logging
- **Safe Defaults**: Manual conflict resolution and push confirmation enabled by default
- **Clear Status Messages**: Improved feedback for all operations

#### Configuration Improvements
- **Default Conflict Strategy**: New configs automatically set `conflict_strategy: "manual"`
- **Auto Backup Setting**: New `auto_backup` configuration option
- **Enhanced Config Examples**: Updated documentation with conflict strategy examples

### 🔧 Technical Improvements

#### File Watcher Enhancements
- **Improved Event Processing**: Better handling of WRITE, CREATE, and RENAME events
- **Parent Directory Watching**: Watches both file and parent directory for atomic writes
- **Event Filtering**: Smarter filtering of irrelevant file system events
- **Debug Logging**: Comprehensive event tracking with timing information

#### Conflict Detection Algorithm
- **Direct Key Comparison**: Replaced hash-based detection with direct key-value comparison
- **No False Positives**: Adding new keys or changing non-overlapping keys won't trigger conflicts
- **Custom Parse Logic**: Handles quoted values, comments, and complex .env formats
- **Backup Creation**: Automatic timestamped backups in `.env-sync-backups/` directory

#### Testing & Quality
- **Comprehensive Test Suite**: All 42 tests passing across 8 packages
- **Atomic Write Testing**: Specific tests for multiple consecutive file changes
- **Conflict Resolution Testing**: Tests for all conflict strategies
- **Integration Testing**: End-to-end testing of file watcher with real file operations

### 📚 Documentation Updates

#### README.md
- **Updated Features Section**: Added conflict detection, smart file watching, debug mode
- **Enhanced Workflow Examples**: Updated daily workflow with new conflict detection
- **File Watcher Documentation**: Comprehensive section on conflict detection and resolution strategies
- **Troubleshooting Guide**: Added debug mode and conflict detection troubleshooting

#### CLAUDE.md  
- **Architecture Updates**: Updated with latest conflict detection and file watching improvements
- **Development Guidelines**: Enhanced with file change detection and conflict resolution patterns
- **Common Operations**: Updated file watching modes and conflict resolution strategies

### 🚀 What's New for Users

#### For Development Teams
```bash
# Smart conflict detection with diff display
env-sync watch                    # Safe mode with manual conflict resolution
env-sync push                     # Automatic conflict detection before push

# Debug file change issues
env-sync watch --debug           # See detailed file system events

# Different conflict strategies per environment
env-sync watch --sync-file .env-sync.dev.yaml    # Manual resolution (safe)
env-sync watch --sync-file .env-sync.qa.yaml     # Automatic backups
```

#### Example Conflict Detection Output
```
⚠️ Conflict detected! Remote version has different values.

🔍 Conflicting keys:
  • API_KEY:
    Local:  user_a_value
    Remote: user_b_value

🚀 Push with local changes (this will overwrite remote)? [y/N]:
```

#### Backup Strategy Example
When using `conflict_strategy: "backup"`, conflicts automatically create:
```
.env-sync-backups/
├── local-20241229-143022.env
└── remote-20241229-143022.env
```

### 🔧 Breaking Changes
- **Push Default**: File watcher now defaults to push-enabled mode (with confirmation)
- **Config Format**: New configs include `conflict_strategy` and `auto_backup` fields

### 🐛 Bug Fixes
- **Fixed**: File watcher only detecting first change (now detects all consecutive changes)
- **Fixed**: Empty conflicting keys in conflict detection (now shows actual conflicting keys)
- **Fixed**: File watcher losing track after atomic writes (now robust re-watching)
- **Fixed**: False conflict detection for non-overlapping changes (now only real conflicts)

### 🏗️ Internal Improvements
- **Enhanced Error Handling**: Better error messages and debugging information
- **Code Organization**: Improved separation between file watching and conflict resolution
- **Performance**: Optimized file change detection and conflict checking
- **Maintainability**: Better test coverage and documentation for future development

## Migration Guide

### For Existing Users
1. **File Watcher Behavior**: The watcher now enables push by default with confirmation prompts
   - To keep old behavior: Use `env-sync watch --push=false`
   - To disable prompts: Use `env-sync watch --confirm=false`

2. **Configuration Files**: New `.env-sync.yaml` files will include conflict strategy settings
   - Existing files will continue to work (defaults to manual strategy)
   - Consider adding `conflict_strategy: "manual"` to existing configs

3. **Conflict Detection**: Push operations now include automatic conflict detection
   - No action needed - this adds safety without breaking existing workflows
   - Use `env-sync push` as before, now with conflict protection

### For Teams
1. **Choose Conflict Strategies**: Decide on appropriate strategies per environment
   - Development: `manual` (safe, collaborative)
   - QA: `backup` (automatic with history)
   - Production: `manual` (extra caution)

2. **Enable Debug Mode**: Use `--debug` flag when troubleshooting file changes
   - Helps identify editor-specific behaviors
   - Useful for diagnosing file watching issues

3. **Update Documentation**: Update team docs with new conflict resolution workflows
   - Include examples of conflict resolution prompts
   - Document chosen conflict strategies per environment