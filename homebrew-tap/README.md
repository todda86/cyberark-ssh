# Homebrew Tap for cyberark-ssh

Private Homebrew tap for distributing `cyberark-ssh`.

## Setup (one-time)

### 1. Host this tap

Push the `homebrew-tap/` directory to a Git repo on your internal server. The repo **must** be named `homebrew-cyberark` (the `homebrew-` prefix is required by Homebrew convention):

```
your-internal-git-server/twa7331/homebrew-cyberark
```

The repo structure should be:

```
homebrew-cyberark/
└── Formula/
    └── cyberark-ssh.rb
```

### 2. Host the source tarball

Create a versioned tarball of the `cyberark-ssh` source and host it on your internal server. For example:

```bash
cd /Users/twa7331/dev/cyberark-ssh
git archive --format=tar.gz --prefix=cyberark-ssh-1.0.0/ -o cyberark-ssh-1.0.0.tar.gz HEAD
```

Upload `cyberark-ssh-1.0.0.tar.gz` to your internal server and note the URL.

Then get the SHA256:

```bash
shasum -a 256 cyberark-ssh-1.0.0.tar.gz
```

### 3. Update the formula

Edit `Formula/cyberark-ssh.rb` and replace:
- `url` — with the actual tarball URL
- `sha256` — with the actual hash from step 2
- `homepage` — with the repo URL

### 4. Push the tap repo

```bash
cd homebrew-tap
git init
git remote add origin https://your-internal-git-server/twa7331/homebrew-cyberark.git
git add .
git commit -m "Add cyberark-ssh formula"
git push -u origin main
```

## Installation (for users)

### Add the tap

```bash
brew tap twa7331/cyberark https://your-internal-git-server/twa7331/homebrew-cyberark.git
```

### Install

```bash
brew install twa7331/cyberark/cyberark-ssh
```

### Upgrade

```bash
brew upgrade cyberark-ssh
```

### Uninstall

```bash
brew uninstall cyberark-ssh
brew untap twa7331/cyberark
```

## Releasing a new version

1. Update the source, tag a release:
   ```bash
   git tag v1.1.0
   git push --tags
   ```

2. Create and upload a new tarball:
   ```bash
   git archive --format=tar.gz --prefix=cyberark-ssh-1.1.0/ -o cyberark-ssh-1.1.0.tar.gz v1.1.0
   # upload to your internal server
   ```

3. Update the formula:
   ```bash
   # In the homebrew-cyberark repo
   # Update version, url, and sha256 in Formula/cyberark-ssh.rb
   git commit -am "Bump cyberark-ssh to 1.1.0"
   git push
   ```

4. Users run:
   ```bash
   brew update
   brew upgrade cyberark-ssh
   ```

## Alternative: HEAD-only install (no tarballs)

If you don't want to manage release tarballs, you can simplify the formula to always build from the latest commit. Replace the `url`/`sha256`/`version` lines with:

```ruby
head "https://your-internal-git-server/twa7331/cyberark-ssh.git", branch: "main"
```

Users would then install with:

```bash
brew install --HEAD twa7331/cyberark/cyberark-ssh
```

And upgrade with:

```bash
brew reinstall twa7331/cyberark/cyberark-ssh
```

The downside is no version tracking — `brew outdated` won't know when there are updates.
