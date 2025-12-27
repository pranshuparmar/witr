{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
  };
  outputs =
    { self, nixpkgs }:
    {
      packages = nixpkgs.lib.genAttrs [ "x86_64-linux" "aarch64-linux" ] (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
          version = "0.1.0";
          commit = if self ? rev then self.rev else "dirty";
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
            src = pkgs.lib.cleanSource ./.;
            vendorHash = null;
            ldflags = [
              "-X main.version=v${version}"
              "-X main.commit=${commit}"
              "-X main.buildDate=${buildDate}"
            ];
          };
        }
      );
    };
}
