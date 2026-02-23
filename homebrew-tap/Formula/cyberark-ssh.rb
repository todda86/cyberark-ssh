class CyberarkSsh < Formula
  desc "CLI wrapper that simplifies SSH/SCP through CyberArk PSMP"
  homepage "https://your-internal-git-server/cyberark-ssh"
  # ---------------------------------------------------------------
  # OPTION A: Point to a release tarball (recommended for versioning)
  #   url "https://your-internal-git-server/cyberark-ssh/archive/v1.0.0.tar.gz"
  #   sha256 "PUT_SHA256_HERE"
  #
  # OPTION B: Point to HEAD of a branch (always builds latest)
  #   head "https://your-internal-git-server/cyberark-ssh.git", branch: "main"
  #
  # For now we use a local path for testing. Replace with your URL.
  # ---------------------------------------------------------------
  url "https://your-internal-git-server/cyberark-ssh/archive/v1.0.0.tar.gz"
  version "1.0.0"
  sha256 "REPLACE_WITH_ACTUAL_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "."
  end

  def post_install
    config = "#{Dir.home}/.cyberark-ssh.yaml"
    unless File.exist?(config)
      ohai "Run 'cyberark-ssh init' to create a starter config at #{config}"
    end
  end

  test do
    assert_match "CyberArk SSH wrapper", shell_output("#{bin}/cyberark-ssh help 2>&1", 0)
  end
end
