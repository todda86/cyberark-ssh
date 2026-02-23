# Homebrew Tap for cyberark-ssh

Private Homebrew tap for distributing `cyberark-ssh`.

## Setup (one-time)

### 1. Host this tap

Push the `homebrew-tap/` directory to a separate Git repo on GitHub. The repo **must** be named `homebrew-cyberark` (the `homebrew-` prefix is required by Homebrew convention):

```
https://github.com/todda86/homebrew-cyberark
```

The repo structure should be:

```
homebrew-cyberark/
└── Formula/
    └── cyberark-ssh.rb
```

### 2. Push the tap repo

```bash
cd homebrew-tap
git init
git remote add origin https://github.com/todda86/homebrew-cyberark.git
git add .
git commit -m "Add cyberark-ssh formula"
git push -u origin main
```

## Installation (for users)

### Add the tap

```bash
brew tap todda86/cyberark
```

### Install

```bash
brew install todda86/cyberark/cyberark-ssh
```

### Upgrade

```bash
brew upgrade cyberark-ssh
```

### Uninstall

```bash
brew uninstall cyberark-ssh
brew untap todda86/cyberark
```

## Releasing a new version

1. Update the source, tag a release:
   ```bash
   cd /Users/twa7331/dev/cyberark-ssh
   git tag v1.1.0
   git push --tags
   ```

2. Get the SHA256 of the new release tarball:
   ```bash
   curl -sL https://github.com/todda86/cyberark-ssh/archive/refs/tags/v1.1.0.tar.gz | shasum -a 256
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
