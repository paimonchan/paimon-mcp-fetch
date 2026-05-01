# frozen_string_literal: true

class PaimonMcpFetch < Formula
  desc "Web content fetching MCP server built with Go"
  homepage "https://github.com/paimonchan/paimon-mcp-fetch"
  url "https://github.com/paimonchan/paimon-mcp-fetch/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/paimonchan/paimon-mcp-fetch.git", branch: "main"

  depends_on "go" => :build

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "./cmd/paimon-mcp-fetch/"
  end

  test do
    assert_match "paimon-mcp-fetch", shell_output("#{bin}/paimon-mcp-fetch --version 2>&1", 1)
  end
end
