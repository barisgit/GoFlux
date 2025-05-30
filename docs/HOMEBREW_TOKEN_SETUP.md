# Homebrew Token Setup Guide

## Why You Need This

The `403 Resource not accessible by integration` error occurs because the default `GITHUB_TOKEN` in GitHub Actions doesn't have write access to external repositories like your `homebrew-goflux` tap.

## How It Works

- `GITHUB_TOKEN` (automatic) → Used for main repository operations (releases, etc.)
- `HOMEBREW_TAP_TOKEN` (manual) → Used only for writing to the `homebrew-goflux` repository

## Step-by-Step Solution

### 1. Create a Personal Access Token (PAT)

1. Go to GitHub → Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token (classic)"
3. Set expiration to "No expiration" (or your preferred duration)
4. Select these scopes:
   - ✅ `public_repo` (for access to public repositories)
   - ✅ `Contents` permission when setting up the token

### 2. Add Token to Repository Secrets

1. Go to your `goflux` repository
2. Settings → Secrets and variables → Actions
3. Click "New repository secret"
4. Name: `HOMEBREW_TAP_TOKEN`
5. Value: Paste the token you just created

### 3. Test the Release

Create a new tag and push:

```bash
git tag v0.1.7
git push origin v0.1.7
```

## What Happens Next

When the release runs, GoReleaser will:

1. Build all binaries ✅ (already working)
2. Create GitHub release ✅ (already working)  
3. Create formula file and push to `homebrew-goflux` ✅ (will work with token)

Users can then install with:

```bash
brew install barisgit/goflux/flux
```

## Alternative: Manual Formula Update

If you prefer not to use a PAT, you can manually update the Homebrew formula after each release by copying the generated formula from the GoReleaser output to your `homebrew-goflux` repository.
