#!/usr/bin/env bash

set -e -x -u

version=$(cat version-semver/number)

cli_sha256=$(shasum -a 256 compiled-darwin/bosh-cli-*-darwin-amd64 | cut -d ' ' -f 1)

pushd homebrew-tap
  cat <<EOF > bosh-cli.rb
class BoshCli < Formula
  desc "New BOSH CLI (beta)"
  homepage "https://bosh.io/docs/cli-v2.html"
  version "${version}"
  url "https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-#{version}-darwin-amd64"
  sha256 "${cli_sha256}"

  depends_on :arch => :x86_64

  option "without-bosh2", "Don't rename binary to 'bosh2'. Useful if the old Ruby CLI is not needed."

  def install
    binary_name = build.without?("bosh2") ? "bosh" : "bosh2"
    bin.install "bosh-cli-#{version}-darwin-amd64" => binary_name
  end

  test do
    system "#{bin}/#{binary_name} --help"
  end
end
EOF

  git add bosh-cli.rb
  if ! [ -z "$(git status --porcelain)" ];
  then
    git config --global user.email "cf-bosh-eng@pivotal.io"
    git config --global user.name "BOSH CI"
    git commit -m "Release bosh-cli ${version}"
  else
    echo "no new version to commit"
  fi
popd

cp -R homebrew-tap update-brew-formula-output
