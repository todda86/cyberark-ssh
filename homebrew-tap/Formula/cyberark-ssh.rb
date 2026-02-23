class CyberarkSsh < Formula
  desc "CLI wrapper that simplifies SSH/SCP through CyberArk PSMP"
  homepage "https://github.com/todda86/cyberark-ssh"
  url "https://github.com/todda86/cyberark-ssh/archive/refs/tags/v1.0.0.tar.gz"
  version "1.0.0"
  sha256 "0bce3d07ffdcff83c7dfa290f1f8025ab10fcf4a217b5c2267cf919b91c30fdd"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "."
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
