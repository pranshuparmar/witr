{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };
  outputs =
    {
      self,
      nixpkgs,
    }:
    let
      inherit (nixpkgs) lib;
    in
    {
      packages = lib.genAttrs [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ] (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = if self ? rev then "git-${builtins.substring 0 7 self.rev}" else "dirty";
          commit = self.rev or "dirty";
          buildDate = pkgs.lib.concatStringsSep "-" [
            (builtins.substring 0 4 self.lastModifiedDate)
            (builtins.substring 4 2 self.lastModifiedDate)
            (builtins.substring 6 2 self.lastModifiedDate)
          ];
        in
        {
          default = pkgs.buildGoModule {
            pname = "witr";
            inherit version;
            src = lib.cleanSourceWith {
              src = ./.;
              filter =
                path: _:
                let
                  pathRelative = lib.removePrefix (toString ./.) (toString path);
                in
                builtins.any (p: lib.hasPrefix p pathRelative) [
                  "/go.mod"
                  "/internal"
                  "/pkg"
                  "/cmd"
                  "/docs"
                  "/vendor"
                ];
            };

            vendorHash = null;
            # The man page generator is a build tool, not something to install.
            excludedPackages = [ "internal/tools/docgen" ];
            ldflags = [
              "-X github.com/pranshuparmar/witr/internal/version.Version=v${version}"
              "-X github.com/pranshuparmar/witr/internal/version.Commit=${commit}"
              "-X github.com/pranshuparmar/witr/internal/version.BuildDate=${buildDate}"
            ];

            nativeBuildInputs = [ pkgs.installShellFiles ];
            postInstall = ''
              installManPage ./docs/cli/witr.1
            '';

            meta = {
              description = "Why is this running?";
              homepage = "https://github.com/pranshuparmar/witr";
              license = lib.licenses.asl20;
            };
          };
        }
      );

      formatter = lib.genAttrs [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ] (
        system: nixpkgs.legacyPackages.${system}.nixpkgs-fmt
      );

      apps = lib.genAttrs [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ] (system: {
        default = {
          type = "app";
          program = "${self.packages.${system}.default}/bin/witr";
        };
      });
    };
}
